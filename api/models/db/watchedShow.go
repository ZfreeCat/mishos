package modelsdb

import (
	"time"
)

// WatchedShow records a user marking an entire show (e.g. a movie or a
// show without seasons/episodes) as watched.
type WatchedShow struct {
	Model
	UserID    uint       `gorm:"index:idx_watched_show_uni,unique" json:"userId"`
	ShowID    uint       `gorm:"index:idx_watched_show_uni,unique" json:"showId"`
	Show      Show       `json:"-"`
	WatchedAt *time.Time `json:"watchedAt"`
}
