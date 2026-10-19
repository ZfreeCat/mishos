package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"mishosapi/db"
	modelsdb "mishosapi/models/db"
)

// CalendarService generates iCalendar (RFC 5545) feeds for the user's
// upcoming episodes and followed movies, plus the per-show calendar files
// used by the "add to calendar" buttons.
type CalendarService struct{}

const icsProdId = "-//Mishos//Mishos Calendar//EN"

// CalendarEvent is a single VEVENT ready for serialization.
type CalendarEvent struct {
	UID         string
	Summary     string
	Description string
	Start       time.Time
	End         time.Time
}

// GetOrCreateToken returns the user's calendar subscription token,
// generating and persisting a new one on first use.
func (cs CalendarService) GetOrCreateToken(userId uint) (string, error) {
	var user modelsdb.User
	if err := db.DB.Preload("UserConfig").First(&user, userId).Error; err != nil {
		return "", err
	}

	cfg := user.UserConfig
	if cfg.ID == 0 {
		return "", errTmdbConfigMissing
	}

	if cfg.CalendarToken != "" {
		return cfg.CalendarToken, nil
	}

	token, err := randomCalendarToken()
	if err != nil {
		return "", err
	}

	cfg.CalendarToken = token
	if err := db.DB.Save(cfg).Error; err != nil {
		return "", err
	}

	return token, nil
}

// GetCalendarICS builds the full upcoming calendar feed (followed shows'
// future episodes plus followed movies' premiere dates).
func (cs CalendarService) GetCalendarICS(userId uint) (string, error) {
	events := []CalendarEvent{}

	// upcoming episodes of followed tv shows
	var upcoming []struct {
		EpisodeID uint
		ShowID    uint
		ShowName  string
		Name      string
		Aired     *time.Time
		Runtime   *uint16
		SeasonNum uint8
		Number    uint16
		Summary   string
	}

	if err := db.DB.
		Model(&modelsdb.Episode{}).
		Select(`
      episodes.id as EpisodeID,
      shows.id as ShowID,
      shows.name as ShowName,
      episodes.name as Name,
      episodes.aired as Aired,
      episodes.runtime as Runtime,
      seasons.number as SeasonNum,
      episodes.number as Number,
      episodes.summary as Summary
    `).
		Joins(`
      inner join seasons on seasons.id = episodes.season_id
      inner join shows on shows.id = seasons.show_id
      inner join followed_shows on followed_shows.show_id = shows.id
    `).
		Where(`followed_shows.user_id = ? and episodes.aired >= ?`, userId, time.Now()).
		Order("episodes.aired asc").
		Scan(&upcoming).Error; err != nil {
		return "", err
	}

	for _, ep := range upcoming {
		if ep.Aired == nil {
			continue
		}
		end := *ep.Aired
		if ep.Runtime != nil && *ep.Runtime > 0 {
			end = ep.Aired.Add(time.Duration(*ep.Runtime) * time.Minute)
		}
		events = append(events, CalendarEvent{
			UID:         fmt.Sprintf("mishos-ep-%d@mishos", ep.EpisodeID),
			Summary:     fmt.Sprintf("%s S%02dE%02d", ep.ShowName, ep.SeasonNum, ep.Number),
			Description: ep.Name,
			Start:       *ep.Aired,
			End:         end,
		})
	}

	// followed movies' premiere dates
	var movies []struct {
		ShowID      uint
		Name        string
		Premiered   *time.Time
		ShowSummary string
	}

	if err := db.DB.
		Model(&modelsdb.Show{}).
		Select(`
      shows.id as ShowID,
      shows.name as Name,
      shows.premiered as Premiered,
      shows.summary as ShowSummary
    `).
		Joins(`
      inner join followed_shows on followed_shows.show_id = shows.id
    `).
		Where(`followed_shows.user_id = ? and shows.media_type = ? and shows.premiered is not null`, userId, "movie").
		Order("shows.premiered asc").
		Scan(&movies).Error; err != nil {
		return "", err
	}

	for _, m := range movies {
		if m.Premiered == nil {
			continue
		}
		events = append(events, CalendarEvent{
			UID:         fmt.Sprintf("mishos-movie-%d@mishos", m.ShowID),
			Summary:     fmt.Sprintf("%s (Movie)", m.Name),
			Description: m.ShowSummary,
			Start:       *m.Premiered,
			End:         *m.Premiered,
		})
	}

	return buildICS(events), nil
}

// GetShowCalendarICS builds a single-show calendar file: the future episodes
// for a tv show, or the premiere event for a movie.
func (cs CalendarService) GetShowCalendarICS(showId uint) (string, error) {
	var show modelsdb.Show
	if err := db.DB.First(&show, showId).Error; err != nil {
		return "", err
	}

	events := []CalendarEvent{}

	if show.MediaType == "movie" {
		if show.Premiered != nil {
			events = append(events, CalendarEvent{
				UID:         fmt.Sprintf("mishos-movie-%d@mishos", show.ID),
				Summary:     fmt.Sprintf("%s (Movie Premiere)", show.Name),
				Description: derefStr(show.Summary),
				Start:       *show.Premiered,
				End:         *show.Premiered,
			})
		}
		return buildICS(events), nil
	}

	var upcoming []struct {
		EpisodeID uint
		Name      string
		Aired     *time.Time
		Runtime   *uint16
		SeasonNum uint8
		Number    uint16
		Summary   string
	}

	if err := db.DB.
		Model(&modelsdb.Episode{}).
		Select(`
      episodes.id as EpisodeID,
      episodes.name as Name,
      episodes.aired as Aired,
      episodes.runtime as Runtime,
      seasons.number as SeasonNum,
      episodes.number as Number,
      episodes.summary as Summary
    `).
		Joins(`
      inner join seasons on seasons.id = episodes.season_id
    `).
		Where(`seasons.show_id = ? and episodes.aired >= ?`, show.ID, time.Now()).
		Order("episodes.aired asc").
		Scan(&upcoming).Error; err != nil {
		return "", err
	}

	for _, ep := range upcoming {
		if ep.Aired == nil {
			continue
		}
		end := *ep.Aired
		if ep.Runtime != nil && *ep.Runtime > 0 {
			end = ep.Aired.Add(time.Duration(*ep.Runtime) * time.Minute)
		}
		events = append(events, CalendarEvent{
			UID:         fmt.Sprintf("mishos-ep-%d@mishos", ep.EpisodeID),
			Summary:     fmt.Sprintf("%s S%02dE%02d", show.Name, ep.SeasonNum, ep.Number),
			Description: ep.Name,
			Start:       *ep.Aired,
			End:         end,
		})
	}

	return buildICS(events), nil
}

func buildICS(events []CalendarEvent) string {
	var b strings.Builder
	b.WriteString("BEGIN:VCALENDAR\r\n")
	b.WriteString("VERSION:2.0\r\n")
	b.WriteString(fmt.Sprintf("PRODID:%s\r\n", icsProdId))
	b.WriteString("CALSCALE:GREGORIAN\r\n")
	b.WriteString("METHOD:PUBLISH\r\n")
	b.WriteString("X-WR-CALNAME:Mishos Upcoming\r\n")

	now := time.Now().UTC().Format("20060102T150405Z")

	for _, event := range events {
		b.WriteString("BEGIN:VEVENT\r\n")
		b.WriteString(fmt.Sprintf("UID:%s\r\n", icsEscape(event.UID)))
		b.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", now))
		b.WriteString(fmt.Sprintf("DTSTART:%s\r\n", icsTime(event.Start)))
		b.WriteString(fmt.Sprintf("DTEND:%s\r\n", icsTime(event.End)))
		b.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", icsEscape(event.Summary)))
		if event.Description != "" {
			b.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", icsEscape(event.Description)))
		}
		b.WriteString("END:VEVENT\r\n")
	}

	b.WriteString("END:VCALENDAR\r\n")
	return b.String()
}

func icsTime(t time.Time) string {
	return t.UTC().Format("20060102T150405Z")
}

// icsEscape escapes commas, semicolons, backslashes and newlines per RFC 5545.
func icsEscape(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, "\r\n", "\\n")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

func randomCalendarToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
