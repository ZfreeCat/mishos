package services

import (
	"errors"
	"sort"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"

	"mishosapi/db"
	modelsdb "mishosapi/models/db"
	modelsdto "mishosapi/models/dto"
	"mishosapi/tvproviders"
)

const (
	TmdbProviderType = "tmdb"

	RecommendationUpcoming = "upcoming"
	RecommendationNew      = "new"
	RecommendationTrending = "trending"
)

var (
	errTmdbConfigMissing = errors.New("user config could not be found")
	errTmdbNotConnected  = errors.New("tmdb account is not connected")
)

type TmdbService struct{}

// ---------- Auth ----------

// CreateRequestToken starts the TMDB v3 authentication flow and returns the
// request token plus the URL the user must open in a browser to approve it.
func (ts TmdbService) CreateRequestToken() (requestToken string, authUrl string, err error) {
	client := tvproviders.TmdbClient{}

	token, err := client.CreateRequestToken()
	if err != nil {
		return "", "", err
	}

	return token, client.AuthUrl(token), nil
}

// CompleteAuth exchanges an authorized request token for a TMDB session,
// loads the account info and stores the binding on the user's config.
func (ts TmdbService) CompleteAuth(userId uint, requestToken string) error {
	client := tvproviders.TmdbClient{}

	sessionId, err := client.CreateSession(requestToken)
	if err != nil {
		return err
	}

	accountId, username, err := client.GetAccountInfo(sessionId)
	if err != nil {
		return err
	}

	var user modelsdb.User
	if err := db.DB.Preload("UserConfig").First(&user, userId).Error; err != nil {
		return err
	}

	if user.UserConfig.ID == 0 {
		return errTmdbConfigMissing
	}

	user.UserConfig.TmdbSessionId = sessionId
	user.UserConfig.TmdbAccountId = accountId
	user.UserConfig.TmdbUsername = username
	user.UserConfig.TmdbSyncEnabled = true

	if err := db.DB.Save(user.UserConfig).Error; err != nil {
		return err
	}

	return nil
}

func (ts TmdbService) Disconnect(userId uint) error {
	var user modelsdb.User
	if err := db.DB.Preload("UserConfig").First(&user, userId).Error; err != nil {
		return err
	}

	if user.UserConfig.ID == 0 {
		return errTmdbConfigMissing
	}

	user.UserConfig.TmdbSessionId = ""
	user.UserConfig.TmdbAccountId = ""
	user.UserConfig.TmdbUsername = ""
	user.UserConfig.TmdbSyncEnabled = false
	user.UserConfig.TmdbLastSyncAt = nil

	if err := db.DB.Save(user.UserConfig).Error; err != nil {
		return err
	}

	return nil
}

// GetTmdbStatus returns the TMDB binding status for a user.
func (ts TmdbService) GetTmdbStatus(userId uint) (connected bool, username string, lastSyncAt *time.Time, err error) {
	var user modelsdb.User
	if err := db.DB.Preload("UserConfig").First(&user, userId).Error; err != nil {
		return false, "", nil, err
	}

	cfg := user.UserConfig
	if cfg.ID == 0 {
		return false, "", nil, nil
	}

	if cfg.TmdbSessionId != "" && cfg.TmdbAccountId != "" {
		return true, cfg.TmdbUsername, cfg.TmdbLastSyncAt, nil
	}

	return false, "", nil, nil
}

// ---------- Watchlist sync ----------

// SyncWatchlist pulls the user's TMDB watchlist (movies + tv) and makes sure
// every entry exists as a followed show in Mishos. It also performs the
// reverse direction: local TMDB shows that are no longer on the TMDB
// watchlist are unfollowed (two-way sync).
func (ts TmdbService) SyncWatchlist(userId uint) (added int, followed int, removed int, err error) {
	var user modelsdb.User
	if err := db.DB.Preload("UserConfig").First(&user, userId).Error; err != nil {
		return 0, 0, 0, err
	}

	cfg := user.UserConfig
	if cfg.ID == 0 || cfg.TmdbSessionId == "" || cfg.TmdbAccountId == "" {
		return 0, 0, 0, errTmdbNotConnected
	}

	client := tvproviders.TmdbClient{}

	// provider ids (as strings) currently present on the TMDB watchlist
	tmdbWatchlist := map[string]bool{}

	for _, mediaType := range []string{tvproviders.TmdbMediaTypeMovie, tvproviders.TmdbMediaTypeTv} {
		items, err := client.GetWatchlist(cfg.TmdbAccountId, cfg.TmdbSessionId, mediaType)
		if err != nil {
			log.Error().Err(err).Msgf("error loading tmdb %s watchlist", mediaType)
			continue
		}

		for _, item := range items {
			tmdbWatchlist[strconv.Itoa(item.ID)] = true

			addedCount, followedCount, err := ts.ensureShowFollowed(userId, item.ID, item.MediaType)
			if err != nil {
				log.Error().Err(err).Msgf("error syncing tmdb watchlist item: %s (%d)", item.Name, item.ID)
				continue
			}

			added += addedCount
			followed += followedCount
		}
	}

	// two-way sync: unfollow local TMDB shows that are no longer on the TMDB watchlist
	var localTmdbFollows []struct {
		ShowID uint
	}
	if err := db.DB.
		Model(&modelsdb.FollowedShow{}).
		Select("followed_shows.show_id as ShowID").
		Joins("inner join shows on shows.id = followed_shows.show_id").
		Where("followed_shows.user_id = ? AND shows.provider_type = ?", userId, TmdbProviderType).
		Scan(&localTmdbFollows).Error; err != nil {
		log.Error().Err(err).Msg("error loading local tmdb follows for removal sync")
	}

	showService := ShowService{}
	for _, local := range localTmdbFollows {
		var show modelsdb.Show
		if err := db.DB.First(&show, local.ShowID).Error; err != nil {
			continue
		}
		if show.ProviderID == nil {
			continue
		}

		if !tmdbWatchlist[*show.ProviderID] {
			if err := showService.Unfollow(userId, show.ID); err != nil {
				log.Error().Err(err).Msgf("error unfollowing show %d no longer on tmdb watchlist", show.ID)
				continue
			}
			removed++
		}
	}

	now := time.Now()
	cfg.TmdbLastSyncAt = &now
	if err := db.DB.Save(cfg).Error; err != nil {
		return added, followed, removed, err
	}

	return added, followed, removed, nil
}

// SyncFollowToTmdb writes a local follow/unfollow back to the user's TMDB
// watchlist (two-way sync). Best-effort: remote failures are logged and never
// fail the local operation.
func (ts TmdbService) SyncFollowToTmdb(userId uint, showId uint, follow bool) {
	var user modelsdb.User
	if err := db.DB.Preload("UserConfig").First(&user, userId).Error; err != nil {
		return
	}

	cfg := user.UserConfig
	if cfg.ID == 0 || cfg.TmdbSessionId == "" || cfg.TmdbAccountId == "" {
		return
	}

	var show modelsdb.Show
	if err := db.DB.First(&show, showId).Error; err != nil {
		return
	}
	if show.ProviderType != TmdbProviderType || show.ProviderID == nil {
		return
	}

	mediaId, err := strconv.Atoi(*show.ProviderID)
	if err != nil {
		log.Error().Err(err).Msgf("tmdb: invalid provider id for show %d", showId)
		return
	}

	client := tvproviders.TmdbClient{}
	if err := client.SetWatchlistItem(cfg.TmdbAccountId, cfg.TmdbSessionId, show.MediaType, mediaId, follow); err != nil {
		log.Error().Err(err).Msgf("tmdb: error writing watchlist for user %d show %d (follow=%t)", userId, showId, follow)
	}
}

// ensureShowFollowed creates the show from TMDB if missing and follows it.
func (ts TmdbService) ensureShowFollowed(userId uint, tmdbId int, mediaType string) (added int, followed int, err error) {
	providerID := strconv.Itoa(tmdbId)

	var show modelsdb.Show
	result := db.DB.Where("provider_type = ? AND provider_id = ?", TmdbProviderType, providerID).First(&show)

	if result.Error != nil {
		// show not stored yet - fetch it from TMDB and create
		client := tvproviders.TmdbClient{}
		newShow, err := client.GetShow(providerID, mediaType)
		if err != nil {
			return 0, 0, err
		}

		if err := db.DB.Create(newShow).Error; err != nil {
			return 0, 0, err
		}

		show = *newShow
		added = 1
	}

	// follow if not followed yet
	count := int64(0)
	if err := db.DB.Model(&modelsdb.FollowedShow{}).Where("user_id = ? AND show_id = ?", userId, show.ID).Count(&count).Error; err != nil {
		return added, 0, err
	}

	if count == 0 {
		showService := ShowService{}
		if err := showService.Follow(userId, show.ID); err != nil {
			return added, 0, err
		}
		followed = 1
	}

	return added, followed, nil
}

// EnsureShowFollowed is the exported wrapper used by the API layer.
func (ts TmdbService) EnsureShowFollowed(userId uint, tmdbId int, mediaType string) (added int, followed int, err error) {
	return ts.ensureShowFollowed(userId, tmdbId, mediaType)
}

// ---------- Recommendations ----------

// GetRecommendations returns new-release recommendations for a category.
// category: upcoming | new | trending
func (ts TmdbService) GetRecommendations(userId uint, category string) ([]modelsdto.RecommendationDto, error) {
	if category == "" {
		category = RecommendationUpcoming
	}

	client := tvproviders.TmdbClient{}
	results := []modelsdto.RecommendationDto{}

	var fetchFn func() ([]tvproviders.TmdbMediaItem, error)

	switch category {
	case RecommendationNew:
		fetchFn = func() ([]tvproviders.TmdbMediaItem, error) {
			movies, err := client.GetNowPlayingMovies()
			if err != nil {
				return nil, err
			}
			tv, err := client.GetTopRated(tvproviders.TmdbMediaTypeTv)
			if err != nil {
				return nil, err
			}
			return append(movies, tv...), nil
		}
	case RecommendationTrending:
		fetchFn = func() ([]tvproviders.TmdbMediaItem, error) {
			movieTrending, err := client.GetTrending(tvproviders.TmdbMediaTypeMovie, "week")
			if err != nil {
				return nil, err
			}
			tvTrending, err := client.GetTrending(tvproviders.TmdbMediaTypeTv, "week")
			if err != nil {
				return nil, err
			}
			return append(movieTrending, tvTrending...), nil
		}
	default: // upcoming
		fetchFn = func() ([]tvproviders.TmdbMediaItem, error) {
			movies, err := client.GetUpcomingMovies()
			if err != nil {
				return nil, err
			}
			tv, err := client.GetOnTheAirTv()
			if err != nil {
				return nil, err
			}
			return append(movies, tv...), nil
		}
	}

	items, err := fetchFn()
	if err != nil {
		return nil, err
	}

	for _, item := range items {
		posterUrl := client.TmdbImageUrl(item.PosterPath, tvproviders.TmdbImageSizeW342)

		date := item.FirstAirDate
		if date == "" {
			date = item.ReleaseDate
		}

		showId, followed := ts.getShowFollowState(userId, item.ID)

		results = append(results, modelsdto.RecommendationDto{
			TmdbId:      item.ID,
			MediaType:   item.MediaType,
			Name:        item.Name,
			Overview:    item.Overview,
			PosterUrl:   RewriteImageUrl(posterUrl),
			Date:        date,
			VoteAverage: item.VoteAverage,
			ShowId:      showId,
			Followed:    followed,
		})
	}

	return results, nil
}

// GetMyRatings returns the user's TMDB rated entries (My Ratings).
// mediaType: all | tv | movie (default all = tv + movie merged).
func (ts TmdbService) GetMyRatings(userId uint, mediaType string) ([]modelsdto.RatingDto, error) {
	var user modelsdb.User
	if err := db.DB.Preload("UserConfig").First(&user, userId).Error; err != nil {
		return nil, err
	}

	cfg := user.UserConfig
	if cfg.ID == 0 || cfg.TmdbSessionId == "" || cfg.TmdbAccountId == "" {
		return nil, errTmdbNotConnected
	}

	client := tvproviders.TmdbClient{}
	results := []modelsdto.RatingDto{}

	types := []string{}
	switch mediaType {
	case tvproviders.TmdbMediaTypeMovie:
		types = []string{tvproviders.TmdbMediaTypeMovie}
	case tvproviders.TmdbMediaTypeTv:
		types = []string{tvproviders.TmdbMediaTypeTv}
	default:
		types = []string{tvproviders.TmdbMediaTypeMovie, tvproviders.TmdbMediaTypeTv}
	}

	for _, mt := range types {
		items, err := client.GetRated(cfg.TmdbAccountId, cfg.TmdbSessionId, mt)
		if err != nil {
			log.Error().Err(err).Msgf("error loading tmdb %s ratings", mt)
			continue
		}

		for _, item := range items {
			posterUrl := client.TmdbImageUrl(item.PosterPath, tvproviders.TmdbImageSizeW342)

			date := item.FirstAirDate
			if date == "" {
				date = item.ReleaseDate
			}

			showId, followed := ts.getShowFollowState(userId, item.ID)

			results = append(results, modelsdto.RatingDto{
				TmdbId:      item.ID,
				MediaType:   item.MediaType,
				Name:        item.Name,
				Overview:    item.Overview,
				PosterUrl:   RewriteImageUrl(posterUrl),
				Date:        date,
				VoteAverage: item.VoteAverage,
				Rating:      item.Rating,
				ShowId:      showId,
				Followed:    followed,
			})
		}
	}

	// stable order: tv first then movies within the "all" view is fine,
	// but sort by name so the merged list reads predictably.
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	return results, nil
}

func (ts TmdbService) getShowFollowState(userId uint, tmdbId int) (*uint, bool) {
	providerID := strconv.Itoa(tmdbId)

	var show modelsdb.Show
	if err := db.DB.Where("provider_type = ? AND provider_id = ?", TmdbProviderType, providerID).First(&show).Error; err != nil {
		return nil, false
	}

	count := int64(0)
	if err := db.DB.Model(&modelsdb.FollowedShow{}).Where("user_id = ? AND show_id = ?", userId, show.ID).Count(&count).Error; err != nil {
		return &show.ID, false
	}

	return &show.ID, count > 0
}
