package modelsdb

import "time"

type UserConfig struct {
	Model
	NotifierTimezone string
	NotifierType     string
	NotifierUrl      string
	Theme            string
	HideSpoilers     bool
	// TMDB account binding
	TmdbSessionId   string
	TmdbAccountId   string
	TmdbUsername    string
	TmdbSyncEnabled bool
	TmdbLastSyncAt  *time.Time
	// iCalendar subscription token (Apple Calendar / any CalDAV client)
	CalendarToken string
	UserID        uint
}
