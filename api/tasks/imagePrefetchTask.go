package tasks

import (
	"os"

	"github.com/go-co-op/gocron/v2"
	"github.com/rs/zerolog/log"

	"mishosapi/db"
	modelsdb "mishosapi/models/db"
	"mishosapi/services"
)

// ImagePrefetchTask downloads remote show images into the local cache so
// pages render without hitting unreachable (from mainland China) image CDNs.
type ImagePrefetchTask struct {
	scheduler gocron.Scheduler
}

func (ipt ImagePrefetchTask) InitTasks() error {
	cron, ok := os.LookupEnv("CRON_IMAGE_PREFETCH")
	if !ok || cron == "" {
		cron = "0 4 * * * "
	}

	log.Info().Msgf("starting image prefetch task with cron: %s", cron)

	_, err := ipt.scheduler.NewJob(
		gocron.CronJob(cron, false),
		gocron.NewTask(ipt.RunImagePrefetch),
		gocron.WithName("imagePrefetch"),
	)

	return err
}

func (ipt ImagePrefetchTask) RunImagePrefetch() {
	shows := []modelsdb.Show{}
	if err := db.DB.Where("image_medium <> '' OR image_original <> ''").Find(&shows).Error; err != nil {
		log.Error().Err(err).Msg("error loading shows for image prefetch")
		return
	}

	cached := 0
	for i := range shows {
		services.PrefetchShowImages(&shows[i])
		cached++
	}

	log.Info().Msgf("image prefetch finished: %d shows processed", cached)
}

func NewImagePrefetchTask(scheduler gocron.Scheduler) *ImagePrefetchTask {
	return &ImagePrefetchTask{scheduler: scheduler}
}
