package services

import (
	"os"
	"strconv"
	"time"

	"mishosapi/db"
	modelsdb "mishosapi/models/db"
	modelsdto "mishosapi/models/dto"
)

type WatchListService struct{}

func (ws WatchListService) GetUnwatched(userId uint) (*[]modelsdto.WatchlistEpisodeDto, error) {
	// get a list of the user's followed shows and their last watched episode date
	type ShowFollowed struct {
		ID          uint64
		Name        string
		ImageMedium string
		Network     string
		LastWatched string
	}

	var followed []ShowFollowed

	err := db.DB.
		Model(&modelsdb.Show{}).
		Select(`
      shows.id as ID,
      shows.name as Name,
      shows.image_medium as ImageMedium,
      shows.network as Network,
      max(watched_episodes.watched_at) as LastWatched
		`).
		Joins(`
      inner join seasons on seasons.show_id = shows.id
      inner join episodes on episodes.season_id = seasons.id
      inner join followed_shows on followed_shows.show_id = shows.id
      left join watched_episodes on watched_episodes.episode_id = episodes.id
    `).
		Where(`
			followed_shows.user_id = ?
		`, userId).
		Group("shows.id").
		Order("LastWatched DESC").
		Scan(&followed).
		Error

	if err != nil {
		return nil, err
	}

	unwatched := []modelsdto.WatchlistEpisodeDto{}

	// build a list of the next unwatched episodes per show
	for _, fshow := range followed {
		var eplist []modelsdto.WatchlistEpisodeDto

		err := db.DB.
			Model(&modelsdb.Episode{}).
			Select(`
		  	episodes.id as ID,
		  	episodes.name as Name,
		  	episodes.summary as Summary,
		  	episodes.number as Number,
		  	episodes.aired as Aired,
		  	episodes.runtime as Runtime,
		  	seasons.number as SeasonNumber,
		  	shows.id as ShowID,
		  	shows.name as ShowName,
		  	shows.image_medium as ImageMedium,
		  	shows.network as Network
		  `).
			Joins(`
        inner join seasons on seasons.id = episodes.season_id
        inner join shows on shows.id = seasons.show_id
        inner join followed_shows on followed_shows.show_id = shows.id
        left join watched_episodes on watched_episodes.episode_id = episodes.id and watched_episodes.user_id = ?
			`, userId).
			Where(`
				shows.id = ?
				and followed_shows.user_id = ?
				and watched_episodes.id is null
				and episodes.aired <= ?
			`, fshow.ID, userId, time.Now()).
			Order("seasons.number ASC, episodes.number ASC").
			Scan(&eplist).
			Error

		if err != nil {
			return nil, err
		}

		if len(eplist) > 0 {
			episode := eplist[0]
			episode.UnwatchedCount = uint(len(eplist))

			unwatched = append(unwatched, episode)
		}
	}

	// rewrite remote image urls to local cached urls
	for i := range unwatched {
		unwatched[i].ImageMedium = RewriteImageUrl(unwatched[i].ImageMedium)
	}

	// include movies: followed movies that have not been watched yet.
	// Movies have no seasons/episodes, so they are rendered as single entries.
	movies, err := ws.getUnwatchedMovies(userId)
	if err != nil {
		return nil, err
	}
	unwatched = append(unwatched, movies...)

	return &unwatched, nil
}

// getUnwatchedMovies returns the user's followed, unwatched movies.
func (ws WatchListService) getUnwatchedMovies(userId uint) ([]modelsdto.WatchlistEpisodeDto, error) {
	var movies = []modelsdto.WatchlistEpisodeDto{}
	err := db.DB.
		Model(&modelsdb.Show{}).
		Select(`
      shows.id as ID,
      shows.name as Name,
      shows.summary as Summary,
      shows.premiered as Aired,
      shows.runtime as Runtime,
      shows.id as ShowID,
      shows.name as ShowName,
      shows.image_medium as ImageMedium,
      shows.network as Network
    `).
		Joins(`
      inner join followed_shows on followed_shows.show_id = shows.id
      left join watched_shows on watched_shows.show_id = shows.id and watched_shows.user_id = ?
    `, userId).
		Where(`
      followed_shows.user_id = ?
      and shows.media_type = ?
      and watched_shows.id is null
    `, userId, "movie").
		Order("shows.name asc").
		Scan(&movies).
		Error

	if err != nil {
		return nil, err
	}

	for i := range movies {
		movies[i].ShowMediaType = "movie"
		movies[i].ImageMedium = RewriteImageUrl(movies[i].ImageMedium)
	}

	return movies, nil
}

func (ws WatchListService) GetRecent(userId uint) (*[]modelsdto.WatchlistEpisodeDto, error) {
	limit := 10

	limitString, ok := os.LookupEnv("WATCHLIST_RECENT_LIMIT")

	if ok {
		converted, err := strconv.ParseInt(limitString, 10, 32)
		if err == nil {
			limit = int(converted)
		}
	}

	var watched = []modelsdto.WatchlistEpisodeDto{}
	err := db.DB.
		Model(&modelsdb.Episode{}).
		Select(`
      episodes.id as ID,
      episodes.name as Name,
      episodes.number as Number,
      episodes.summary as Summary,
      episodes.aired as Aired,
      episodes.runtime as Runtime,
      seasons.number as SeasonNumber,
      watched_episodes.watched_at as WatchedAt,
      shows.id as ShowID,
      shows.name as ShowName,
      shows.image_medium as ImageMedium,
      shows.network as Network
    `).
		Joins(`
      inner join watched_episodes on watched_episodes.episode_id = episodes.id
      inner join seasons on seasons.id = episodes.season_id
      inner join shows on shows.id = seasons.show_id
    `).
		Where(`
		  watched_episodes.user_id = ?
    `, userId).
		Order("watched_episodes.watched_at desc").
		Limit(limit).
		Scan(&watched).
		Error

	if err != nil {
		return nil, err
	}

	// rewrite remote image urls to local cached urls
	for i := range watched {
		watched[i].ImageMedium = RewriteImageUrl(watched[i].ImageMedium)
	}

	// include movies in the recently watched list
	movies, err := ws.getRecentMovies(userId, limit)
	if err != nil {
		return nil, err
	}
	watched = append(watched, movies...)

	return &watched, nil
}

// getRecentMovies returns the user's recently watched movies.
func (ws WatchListService) getRecentMovies(userId uint, limit int) ([]modelsdto.WatchlistEpisodeDto, error) {
	var movies = []modelsdto.WatchlistEpisodeDto{}
	err := db.DB.
		Model(&modelsdb.WatchedShow{}).
		Select(`
      watched_shows.watched_at as WatchedAt,
      shows.id as ID,
      shows.id as ShowID,
      shows.name as Name,
      shows.name as ShowName,
      shows.image_medium as ImageMedium,
      shows.network as Network,
      shows.runtime as Runtime
    `).
		Joins(`
      inner join shows on shows.id = watched_shows.show_id
    `).
		Where(`
      watched_shows.user_id = ?
      and shows.media_type = ?
    `, userId, "movie").
		Order("watched_shows.watched_at desc").
		Limit(limit).
		Scan(&movies).
		Error

	if err != nil {
		return nil, err
	}

	for i := range movies {
		movies[i].ShowMediaType = "movie"
		movies[i].ImageMedium = RewriteImageUrl(movies[i].ImageMedium)
	}

	return movies, nil
}

func (ws WatchListService) GetUpcoming(userId uint) (*[]modelsdto.WatchlistEpisodeDto, error) {
	var upcoming = []modelsdto.WatchlistEpisodeDto{}
	err := db.DB.
		Model(&modelsdb.Episode{}).
		Select(`
      episodes.id as ID,
      episodes.name as Name,
      episodes.number as Number,
      episodes.summary as Summary,
      episodes.aired as Aired,
      episodes.runtime as Runtime,
      seasons.number as SeasonNumber,
      shows.id as ShowID,
      shows.name as ShowName,
      shows.image_medium as ImageMedium,
      shows.network as Network
    `).
		Joins(`
      inner join seasons on seasons.id = episodes.season_id
      inner join shows on shows.id = seasons.show_id
      inner join followed_shows on followed_shows.show_id = shows.id
    `).
		Where(`
		  followed_shows.user_id = ?
		  and episodes.aired >= ?
    `, userId, time.Now()).
		Order("episodes.aired asc").
		Scan(&upcoming).
		Error

	if err != nil {
		return nil, err
	}

	// rewrite remote image urls to local cached urls
	for i := range upcoming {
		upcoming[i].ImageMedium = RewriteImageUrl(upcoming[i].ImageMedium)
	}

	return &upcoming, nil
}
