package tasks

import (
	"os"

	"github.com/go-co-op/gocron/v2"
	"github.com/rs/zerolog/log"

	"mishosapi/db"
	modelsdb "mishosapi/models/db"
	"mishosapi/services"
)

// TmdbSyncTask periodically syncs every connected user's TMDB watchlist.
type TmdbSyncTask struct {
	scheduler gocron.Scheduler
}

func (tst TmdbSyncTask) InitTasks() error {
	cron, ok := os.LookupEnv("CRON_TMDB_SYNC")
	if !ok || cron == "" {
		cron = "0 3 * * * "
	}

	log.Info().Msgf("starting tmdb watchlist sync task with cron: %s", cron)

	_, err := tst.scheduler.NewJob(
		gocron.CronJob(cron, false),
		gocron.NewTask(tst.RunTmdbSync),
		gocron.WithName("tmdbSync"),
	)

	return err
}

func (tst TmdbSyncTask) RunTmdbSync() {
	tmdbService := services.TmdbService{}

	var configs []modelsdb.UserConfig
	if err := db.DB.Where("tmdb_session_id <> '' AND tmdb_account_id <> '' AND tmdb_sync_enabled = ?", true).Find(&configs).Error; err != nil {
		log.Error().Err(err).Msg("error loading tmdb users for sync")
		return
	}

	for _, config := range configs {
		added, followed, removed, err := tmdbService.SyncWatchlist(config.UserID)
		if err != nil {
			log.Error().Err(err).Msgf("error syncing tmdb watchlist for user %d", config.UserID)
			continue
		}

		log.Info().Msgf("tmdb watchlist synced for user %d: added=%d followed=%d removed=%d", config.UserID, added, followed, removed)
	}
}

func NewTmdbSyncTask(scheduler gocron.Scheduler) *TmdbSyncTask {
	return &TmdbSyncTask{scheduler: scheduler}
}
