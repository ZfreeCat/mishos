package services

import (
	"errors"
	"time"

	"mishosapi/db"
	modelsdb "mishosapi/models/db"
)

var errShowNotFound = errors.New("show not found")

// WatchedShowService handles marking an entire show (movies, or shows
// without seasons/episodes) as watched.
type WatchedShowService struct{}

func (ws WatchedShowService) WatchShow(userId uint, showId uint, score float64) (watched bool, err error) {
	count := int64(0)
	if err := db.DB.Model(&modelsdb.WatchedShow{}).Where("user_id = ? AND show_id = ?", userId, showId).Count(&count).Error; err != nil {
		return false, err
	}

	if count > 0 {
		return false, nil
	}

	// make sure the show exists
	if err := db.DB.Model(&modelsdb.Show{}).Where("id = ?", showId).Count(&count).Error; err != nil {
		return false, err
	}

	if count == 0 {
		return false, errShowNotFound
	}

	now := time.Now()

	watchedShow := modelsdb.WatchedShow{
		UserID:    userId,
		ShowID:    showId,
		WatchedAt: &now,
	}

	if err := db.DB.Create(&watchedShow).Error; err != nil {
		return false, err
	}

	// mirror watched state back to TMDB with the user's score (best-effort)
	tmdbService := TmdbService{}
	tmdbService.SyncWatchedToTmdb(userId, showId, true, score)

	return true, nil
}

func (ws WatchedShowService) UnwatchShow(userId uint, showId uint) error {
	if err := db.DB.Where("user_id = ? AND show_id = ?", userId, showId).Unscoped().Delete(&modelsdb.WatchedShow{}).Error; err != nil {
		return err
	}

	// mirror unwatch back to TMDB (best-effort; front-end confirms first)
	tmdbService := TmdbService{}
	tmdbService.SyncWatchedToTmdb(userId, showId, false, 0)

	return nil
}

func (ws WatchedShowService) IsShowWatched(userId uint, showId uint) (bool, error) {
	count := int64(0)
	if err := db.DB.Model(&modelsdb.WatchedShow{}).Where("user_id = ? AND show_id = ?", userId, showId).Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}
