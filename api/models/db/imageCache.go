package modelsdb

// ImageCache maps a locally served image hash back to its remote source URL.
// The local file itself lives on disk under the configured image cache dir.
type ImageCache struct {
	Model
	Hash        string `gorm:"uniqueIndex" json:"hash"`
	SourceURL   string `gorm:"type:text" json:"sourceUrl"`
	ContentType string `json:"contentType"`
}
