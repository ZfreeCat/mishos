package tvproviders

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	strip "github.com/grokify/html-strip-tags-go"
	"github.com/tidwall/gjson"

	modelsdb "mishosapi/models/db"
)

const (
	TmdbMediaTypeTv    = "tv"
	TmdbMediaTypeMovie = "movie"

	TmdbImageSizeOriginal = "original"
	TmdbImageSizeW500     = "w500"
	TmdbImageSizeW342     = "w342"
)

var tmdbHTTPClient = &http.Client{Timeout: 15 * time.Second}

// TmdbApiBase returns the TMDB API base URL. In China Mainland networks
// api.themoviedb.org is not directly reachable; point TMDB_API_URL at a
// reverse proxy / mirror if needed.
func TmdbApiBase() string {
	base := strings.TrimSpace(os.Getenv("TMDB_API_URL"))
	if base == "" {
		base = "https://api.themoviedb.org/3"
	}
	return strings.TrimSuffix(base, "/")
}

// TmdbImageBase returns the TMDB image CDN base URL (also not reachable in
// China Mainland; images are cached locally so this only matters for the
// prefetch/download step).
func TmdbImageBase() string {
	base := strings.TrimSpace(os.Getenv("TMDB_IMAGE_URL"))
	if base == "" {
		base = "https://image.tmdb.org/t/p"
	}
	return strings.TrimSuffix(base, "/")
}

func TmdbApiKey() (string, error) {
	key := strings.TrimSpace(os.Getenv("TMDB_API_KEY"))
	if key == "" {
		return "", errors.New("TMDB_API_KEY is not set: register at https://www.themoviedb.org/settings/api and add it to your environment")
	}
	return key, nil
}

// TmdbMediaItem is the common shape of a media entry returned by
// trending / upcoming / on_the_air / top_rated endpoints.
type TmdbMediaItem struct {
	ID           int     `json:"id"`
	MediaType    string  `json:"mediaType"`
	Name         string  `json:"name"`
	Overview     string  `json:"overview"`
	PosterPath   string  `json:"posterPath"`
	BackdropPath string  `json:"backdropPath"`
	FirstAirDate string  `json:"firstAirDate"`
	ReleaseDate  string  `json:"releaseDate"`
	VoteAverage  float64 `json:"voteAverage"`
	OriginalName string  `json:"originalName"`
}

// TmdbSearchResult mirrors a single search hit.
type TmdbSearchResult struct {
	ID           int    `json:"id"`
	MediaType    string `json:"mediaType"`
	Name         string `json:"name"`
	Overview     string `json:"overview"`
	PosterPath   string `json:"posterPath"`
	FirstAirDate string `json:"firstAirDate"`
	ReleaseDate  string `json:"releaseDate"`
}

// TmdbWatchlistItem mirrors one entry of the user's TMDB watchlist.
type TmdbWatchlistItem struct {
	ID           int     `json:"id"`
	MediaType    string  `json:"mediaType"`
	Name         string  `json:"name"`
	Overview     string  `json:"overview"`
	PosterPath   string  `json:"posterPath"`
	FirstAirDate string  `json:"firstAirDate"`
	ReleaseDate  string  `json:"releaseDate"`
	VoteAverage  float64 `json:"voteAverage"`
}

// TmdbRatedItem mirrors one entry of the user's TMDB rated list ("My Ratings").
type TmdbRatedItem struct {
	ID           int     `json:"id"`
	MediaType    string  `json:"mediaType"`
	Name         string  `json:"name"`
	Overview     string  `json:"overview"`
	PosterPath   string  `json:"posterPath"`
	FirstAirDate string  `json:"firstAirDate"`
	ReleaseDate  string  `json:"releaseDate"`
	VoteAverage  float64 `json:"voteAverage"`
	Rating       float64 `json:"rating"`
}

type TmdbClient struct{}

// TmdbImageUrl builds a full image URL from a TMDB image path.
func (tc TmdbClient) TmdbImageUrl(path string, size string) string {
	if path == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s%s", TmdbImageBase(), size, path)
}

// CreateRequestToken starts the v3 authentication flow (step 1).
func (tc TmdbClient) CreateRequestToken() (requestToken string, err error) {
	body, err := tc.get("authentication/token/new", nil)
	if err != nil {
		return "", err
	}

	json := gjson.Parse(body)
	token := json.Get("request_token").String()
	if token == "" {
		return "", fmt.Errorf("tmdb: could not create request token: %s", json.Get("status_message").String())
	}

	return token, nil
}

// AuthUrl returns the browser URL the user must open to approve the token.
func (tc TmdbClient) AuthUrl(requestToken string) string {
	return fmt.Sprintf("https://www.themoviedb.org/authenticate/%s", requestToken)
}

// CreateSession exchanges an authorized request token for a session id (step 3).
func (tc TmdbClient) CreateSession(requestToken string) (sessionId string, err error) {
	params := url.Values{}
	params.Add("request_token", requestToken)

	body, err := tc.get("authentication/session/new", &params)
	if err != nil {
		return "", err
	}

	json := gjson.Parse(body)
	session := json.Get("session_id").String()
	if session == "" {
		return "", fmt.Errorf("tmdb: could not create session (token not authorized?): %s", json.Get("status_message").String())
	}

	return session, nil
}

// GetAccountInfo returns the account id and username for a session.
func (tc TmdbClient) GetAccountInfo(sessionId string) (accountId string, username string, err error) {
	params := url.Values{}
	params.Add("session_id", sessionId)

	body, err := tc.get("account", &params)
	if err != nil {
		return "", "", err
	}

	json := gjson.Parse(body)
	account := json.Get("id").String()
	if account == "" {
		return "", "", fmt.Errorf("tmdb: could not load account: %s", json.Get("status_message").String())
	}

	return account, json.Get("username").String(), nil
}

// Search searches movies or tv shows on TMDB.
func (tc TmdbClient) Search(query string, mediaType string) ([]TmdbSearchResult, error) {
	params := url.Values{}
	params.Add("query", query)

	body, err := tc.get(fmt.Sprintf("search/%s", mediaType), &params)
	if err != nil {
		return nil, err
	}

	results := []TmdbSearchResult{}
	json := gjson.Parse(body)

	json.Get("results").ForEach(func(_, value gjson.Result) bool {
		results = append(results, TmdbSearchResult{
			ID:           int(value.Get("id").Int()),
			MediaType:    mediaType,
			Name:         value.Get("name").String(),
			Overview:     value.Get("overview").String(),
			PosterPath:   value.Get("poster_path").String(),
			FirstAirDate: value.Get("first_air_date").String(),
			ReleaseDate:  value.Get("release_date").String(),
		})
		return true
	})

	return results, nil
}

// GetWatchlist loads the user's watchlist for one media type.
func (tc TmdbClient) GetWatchlist(accountId string, sessionId string, mediaType string) ([]TmdbWatchlistItem, error) {
	params := url.Values{}
	params.Add("session_id", sessionId)

	body, err := tc.get(fmt.Sprintf("account/%s/watchlist/%s", accountId, mediaType), &params)
	if err != nil {
		return nil, err
	}

	results := []TmdbWatchlistItem{}
	json := gjson.Parse(body)

	json.Get("results").ForEach(func(_, value gjson.Result) bool {
		results = append(results, TmdbWatchlistItem{
			ID:           int(value.Get("id").Int()),
			MediaType:    mediaType,
			Name:         value.Get("name").String(),
			Overview:     value.Get("overview").String(),
			PosterPath:   value.Get("poster_path").String(),
			FirstAirDate: value.Get("first_air_date").String(),
			ReleaseDate:  value.Get("release_date").String(),
			VoteAverage:  value.Get("vote_average").Float(),
		})
		return true
	})

	return results, nil
}

// GetRated loads the user's rated movies or tv shows (My Ratings).
func (tc TmdbClient) GetRated(accountId string, sessionId string, mediaType string) ([]TmdbRatedItem, error) {
	// TMDB v3 uses /account/{id}/rated/movies (plural) - map movie -> movies
	apiMediaType := mediaType
	if apiMediaType == TmdbMediaTypeMovie {
		apiMediaType = "movies"
	}

	params := url.Values{}
	params.Add("session_id", sessionId)
	params.Add("sort_by", "created_at.asc")

	results := []TmdbRatedItem{}
	page := 1

	for {
		params.Set("page", strconv.Itoa(page))

		body, err := tc.get(fmt.Sprintf("account/%s/rated/%s", accountId, apiMediaType), &params)
		if err != nil {
			return nil, err
		}

		json := gjson.Parse(body)
		json.Get("results").ForEach(func(_, value gjson.Result) bool {
			name := value.Get("name").String()
			if name == "" {
				name = value.Get("title").String()
			}

			results = append(results, TmdbRatedItem{
				ID:           int(value.Get("id").Int()),
				MediaType:    mediaType,
				Name:         name,
				Overview:     value.Get("overview").String(),
				PosterPath:   value.Get("poster_path").String(),
				FirstAirDate: value.Get("first_air_date").String(),
				ReleaseDate:  value.Get("release_date").String(),
				VoteAverage:  value.Get("vote_average").Float(),
				Rating:       value.Get("rating").Float(),
			})
			return true
		})

		totalPages := int(json.Get("total_pages").Int())
		if page >= totalPages || totalPages == 0 {
			break
		}
		page++
	}

	return results, nil
}

// SetWatchlistItem adds (onWatchlist=true) or removes (onWatchlist=false) a
// movie/tv entry to/from the user's TMDB watchlist. Used by the two-way sync.
func (tc TmdbClient) SetWatchlistItem(accountId string, sessionId string, mediaType string, mediaId int, onWatchlist bool) error {
	key, err := TmdbApiKey()
	if err != nil {
		return err
	}

	base := TmdbApiBase()
	apiURL := fmt.Sprintf("%s/account/%s/watchlist?api_key=%s&session_id=%s", base, accountId, url.QueryEscape(key), url.QueryEscape(sessionId))

	payload, err := json.Marshal(map[string]any{
		"media_type": mediaType,
		"media_id":   mediaId,
		"watchlist":  onWatchlist,
	})
	if err != nil {
		return err
	}

	request, err := http.NewRequest("POST", apiURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := tmdbHTTPClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(response.Body)
		statusMessage := gjson.Get(string(bodyBytes), "status_message").String()
		return fmt.Errorf("tmdb: received error from provider with code %d: %s", response.StatusCode, statusMessage)
	}

	return nil
}

// GetTrending loads trending movies or tv shows for a time window (day|week).
func (tc TmdbClient) GetTrending(mediaType string, window string) ([]TmdbMediaItem, error) {
	if mediaType == "" {
		mediaType = TmdbMediaTypeTv
	}
	if window == "" {
		window = "week"
	}

	body, err := tc.get(fmt.Sprintf("trending/%s/%s", mediaType, window), nil)
	if err != nil {
		return nil, err
	}

	return parseMediaItems(body, mediaType)
}

// GetUpcomingMovies loads movies releasing in cinemas soon.
func (tc TmdbClient) GetUpcomingMovies() ([]TmdbMediaItem, error) {
	body, err := tc.get("movie/upcoming", nil)
	if err != nil {
		return nil, err
	}

	return parseMediaItems(body, TmdbMediaTypeMovie)
}

// GetOnTheAirTv loads tv shows airing in the next 7 days.
func (tc TmdbClient) GetOnTheAirTv() ([]TmdbMediaItem, error) {
	body, err := tc.get("tv/on_the_air", nil)
	if err != nil {
		return nil, err
	}

	return parseMediaItems(body, TmdbMediaTypeTv)
}

// GetNowPlayingMovies loads movies currently in cinemas.
func (tc TmdbClient) GetNowPlayingMovies() ([]TmdbMediaItem, error) {
	body, err := tc.get("movie/now_playing", nil)
	if err != nil {
		return nil, err
	}

	return parseMediaItems(body, TmdbMediaTypeMovie)
}

// GetTopRated loads top rated movies or tv shows (used by the trending tab).
func (tc TmdbClient) GetTopRated(mediaType string) ([]TmdbMediaItem, error) {
	if mediaType == "" {
		mediaType = TmdbMediaTypeTv
	}

	body, err := tc.get(fmt.Sprintf("%s/top_rated", mediaType), nil)
	if err != nil {
		return nil, err
	}

	return parseMediaItems(body, mediaType)
}

// GetShow loads a full show (details + seasons + episodes for tv) from TMDB.
func (tc TmdbClient) GetShow(tmdbId string, mediaType string) (*modelsdb.Show, error) {
	if mediaType == "" {
		mediaType = TmdbMediaTypeTv
	}

	body, err := tc.get(fmt.Sprintf("%s/%s", mediaType, tmdbId), nil)
	if err != nil {
		return nil, err
	}

	json := gjson.Parse(body)

	show := &modelsdb.Show{
		ProviderID:   StrPtr(tmdbId),
		ProviderType: "tmdb",
		MediaType:    mediaType,
		Name:         json.Get("name").String(),
	}

	if show.Name == "" {
		show.Name = json.Get("title").String()
	}

	show.Summary = StrPtr(strip.StripTags(json.Get("overview").String()))
	show.Language = StrPtr(json.Get("original_language").String())
	show.Status = StrPtr(json.Get("status").String())

	if show.MediaType == TmdbMediaTypeMovie {
		show.Premiered = parseTmdbDate(json.Get("release_date").String())
		show.Runtime = Uint16Ptr(json.Get("runtime").Uint())
		show.ImageMedium = StrPtr(tc.TmdbImageUrl(json.Get("poster_path").String(), TmdbImageSizeW342))
		show.ImageOriginal = StrPtr(tc.TmdbImageUrl(json.Get("poster_path").String(), TmdbImageSizeOriginal))
	} else {
		show.Premiered = parseTmdbDate(json.Get("first_air_date").String())
		show.ImageMedium = StrPtr(tc.TmdbImageUrl(json.Get("poster_path").String(), TmdbImageSizeW342))
		show.ImageOriginal = StrPtr(tc.TmdbImageUrl(json.Get("poster_path").String(), TmdbImageSizeOriginal))

		// pull seasons + episodes per season
		seasonsJson := json.Get("seasons").Array()

		for _, seasonValue := range seasonsJson {
			seasonNumber := uint8(seasonValue.Get("season_number").Int())

			season := &modelsdb.Season{
				ProviderID:  StrPtr(fmt.Sprintf("%s:s%d", tmdbId, seasonNumber)),
				Number:      seasonNumber,
				Premiered:   parseTmdbDate(seasonValue.Get("air_date").String()),
				Network:     StrPtr(""),
				EpisodeOrder: nil,
			}

			if seasonValue.Get("episode_count").Exists() {
				count := uint16(seasonValue.Get("episode_count").Int())
				season.EpisodeOrder = &count
			}

			episodes, err := tc.getSeasonEpisodes(tmdbId, seasonNumber)
			if err == nil {
				season.Episodes = episodes
			}

			show.Seasons = append(show.Seasons, *season)
		}
	}

	return show, nil
}

func (tc TmdbClient) getSeasonEpisodes(tmdbId string, seasonNumber uint8) ([]modelsdb.Episode, error) {
	body, err := tc.get(fmt.Sprintf("tv/%s/season/%d", tmdbId, seasonNumber), nil)
	if err != nil {
		return nil, err
	}

	episodes := []modelsdb.Episode{}
	json := gjson.Parse(body)

	json.Get("episodes").ForEach(func(_, value gjson.Result) bool {
		number := uint16(value.Get("episode_number").Int())

		episode := &modelsdb.Episode{
			ProviderID:   StrPtr(fmt.Sprintf("%s:s%d:e%d", tmdbId, seasonNumber, number)),
			Name:         StrPtr(value.Get("name").String()),
			Number:       &number,
			Type:         StrPtr("regular"),
			Runtime:      Uint16Ptr(value.Get("runtime").Uint()),
			Summary:      StrPtr(strip.StripTags(value.Get("overview").String())),
			SeasonNumber: &seasonNumber,
		}

		if value.Get("air_date").String() != "" {
			if t, err := time.Parse(time.DateOnly, value.Get("air_date").String()); err == nil {
				episode.Aired = &t
			}
		}

		episodes = append(episodes, *episode)
		return true
	})

	return episodes, nil
}

func parseMediaItems(body string, mediaType string) ([]TmdbMediaItem, error) {
	items := []TmdbMediaItem{}
	json := gjson.Parse(body)

	json.Get("results").ForEach(func(_, value gjson.Result) bool {
		name := value.Get("name").String()
		if name == "" {
			name = value.Get("title").String()
		}

		items = append(items, TmdbMediaItem{
			ID:           int(value.Get("id").Int()),
			MediaType:    mediaType,
			Name:         name,
			Overview:     value.Get("overview").String(),
			PosterPath:   value.Get("poster_path").String(),
			BackdropPath: value.Get("backdrop_path").String(),
			FirstAirDate: value.Get("first_air_date").String(),
			ReleaseDate:  value.Get("release_date").String(),
			VoteAverage:  value.Get("vote_average").Float(),
			OriginalName: value.Get("original_name").String(),
		})
		return true
	})

	return items, nil
}

func parseTmdbDate(date string) *time.Time {
	if date == "" {
		return nil
	}

	if t, err := time.Parse(time.DateOnly, date); err == nil {
		return &t
	}

	return nil
}

func (tc TmdbClient) get(uri string, params *url.Values) (string, error) {
	key, err := TmdbApiKey()
	if err != nil {
		return "", err
	}

	base := TmdbApiBase()
	apiURL := fmt.Sprintf("%s/%s?api_key=%s", base, uri, url.QueryEscape(key))

	request, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", err
	}

	if params != nil {
		request.URL.RawQuery = request.URL.RawQuery + "&" + params.Encode()
	}

	response, err := tmdbHTTPClient.Do(request)
	if err != nil {
		return "", err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(response.Body)
		statusMessage := gjson.Get(string(bodyBytes), "status_message").String()
		return "", fmt.Errorf("tmdb: received error from provider with code %d: %s", response.StatusCode, statusMessage)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
