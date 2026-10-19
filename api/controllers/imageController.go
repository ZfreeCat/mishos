package controllers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"mishosapi/db"
	modelsdb "mishosapi/models/db"
	"mishosapi/services"
)

type ImageController struct{}

// GetImage serves a locally cached image, downloading it lazily from the
// source URL on first access. This route is intentionally unauthenticated so
// plain <img> tags can load images without headers.
func (ic ImageController) GetImage(context *gin.Context) {
	name := context.Param("name")
	if name == "" {
		context.Status(http.StatusNotFound)
		return
	}

	parts := strings.SplitN(name, ".", 2)
	hash := parts[0]
	if hash == "" {
		context.Status(http.StatusNotFound)
		return
	}

	var cache modelsdb.ImageCache
	if err := db.DB.Where("hash = ?", hash).First(&cache).Error; err != nil {
		context.Status(http.StatusNotFound)
		return
	}

	path := filepath.Join(services.ImageCacheDir(), name)

	if _, err := os.Stat(path); err != nil {
		if err := services.EnsureCached(cache.SourceURL); err != nil {
			context.Status(http.StatusBadGateway)
			return
		}
	}

	context.File(path)
}
