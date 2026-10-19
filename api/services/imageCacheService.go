package services

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"mishosapi/db"
	modelsdb "mishosapi/models/db"
)

var imageHTTPClient = &http.Client{Timeout: 30 * time.Second}

// per-hash locks so concurrent requests for the same image download once
var imageDownloadLocks sync.Map // map[string]*sync.Mutex

// ImageCacheDir returns the local directory images are cached into.
func ImageCacheDir() string {
	dir := strings.TrimSpace(os.Getenv("IMAGE_CACHE_DIR"))
	if dir == "" {
		dir = "./data/images"
	}
	return dir
}

// allowedImageDomains returns the domain whitelist used when downloading
// remote images. Extra domains can be added with IMAGE_ALLOWED_DOMAINS.
func allowedImageDomains() []string {
	domains := []string{"tvmaze.com", "tmdb.org"}
	extra := strings.TrimSpace(os.Getenv("IMAGE_ALLOWED_DOMAINS"))
	if extra != "" {
		for _, d := range strings.Split(extra, ",") {
			d = strings.TrimSpace(d)
			if d != "" {
				domains = append(domains, d)
			}
		}
	}
	return domains
}

func imageDomainAllowed(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return false
	}

	host := strings.ToLower(parsed.Host)
	for _, domain := range allowedImageDomains() {
		domain = strings.ToLower(domain)
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return true
		}
	}

	return false
}

// HashImageUrl returns the sha1 hash of a source image URL.
func HashImageUrl(sourceURL string) string {
	sum := sha1.Sum([]byte(sourceURL))
	return hex.EncodeToString(sum[:])
}

// imageExt guesses a file extension from an image URL.
func imageExt(sourceURL string) string {
	parsed, err := url.Parse(sourceURL)
	if err != nil {
		return ".jpg"
	}

	ext := strings.ToLower(filepath.Ext(parsed.Path))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif":
		return ext
	}

	return ".jpg"
}

// cachedFilePath returns the on-disk path for a source URL.
func cachedFilePath(sourceURL string) string {
	return filepath.Join(ImageCacheDir(), HashImageUrl(sourceURL)+imageExt(sourceURL))
}

// RewriteImageUrl turns a remote image URL into a local /api/images/{hash}{ext}
// URL and registers the hash -> source mapping so the image can be served and
// downloaded lazily. Returns "" for empty input.
func RewriteImageUrl(sourceURL string) string {
	sourceURL = strings.TrimSpace(sourceURL)
	if sourceURL == "" {
		return ""
	}

	hash := HashImageUrl(sourceURL)
	ext := imageExt(sourceURL)

	// register the mapping (used both for serving and for the prefetch task)
	registerImageMapping(hash, sourceURL)

	return fmt.Sprintf("/api/images/%s%s", hash, ext)
}

func registerImageMapping(hash string, sourceURL string) {
	var count int64
	db.DB.Model(&modelsdb.ImageCache{}).Where("hash = ?", hash).Count(&count)
	if count == 0 {
		record := modelsdb.ImageCache{
			Hash:      hash,
			SourceURL: sourceURL,
		}
		if err := db.DB.Create(&record).Error; err != nil {
			log.Warn().Err(err).Msgf("could not register image mapping %s", sourceURL)
		}
	}
}

// EnsureCached downloads a remote image to the local cache dir if it is not
// already present. It refuses to download from domains outside the whitelist.
func EnsureCached(sourceURL string) error {
	sourceURL = strings.TrimSpace(sourceURL)
	if sourceURL == "" {
		return errors.New("empty image url")
	}

	if !imageDomainAllowed(sourceURL) {
		return fmt.Errorf("image domain not allowed: %s", sourceURL)
	}

	hash := HashImageUrl(sourceURL)
	path := cachedFilePath(sourceURL)

	lock := getImageLock(hash)
	lock.Lock()
	defer lock.Unlock()

	// already cached?
	if _, err := os.Stat(path); err == nil {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	request, err := http.NewRequest("GET", sourceURL, nil)
	if err != nil {
		return err
	}

	request.Header.Set("User-Agent", "mishos/1.0 (+self-hosted)")

	response, err := imageHTTPClient.Do(request)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download image %s: status %d", sourceURL, response.StatusCode)
	}

	// write to a temp file first to avoid serving partially downloaded files
	tmpPath := path + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, response.Body); err != nil {
		out.Close()
		os.Remove(tmpPath)
		return err
	}

	if err := out.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}

	// record content type for later reference
	contentType := response.Header.Get("Content-Type")
	db.DB.Model(&modelsdb.ImageCache{}).Where("hash = ?", hash).Update("content_type", contentType)

	log.Info().Msgf("cached image %s -> %s", sourceURL, path)
	return nil
}

func getImageLock(hash string) *sync.Mutex {
	value, _ := imageDownloadLocks.LoadOrStore(hash, &sync.Mutex{})
	return value.(*sync.Mutex)
}

// PrefetchShowImages downloads the medium and original images of a show.
func PrefetchShowImages(show *modelsdb.Show) {
	for _, img := range []*string{show.ImageMedium, show.ImageOriginal} {
		if img == nil || strings.TrimSpace(*img) == "" {
			continue
		}
		if err := EnsureCached(*img); err != nil {
			log.Warn().Err(err).Msgf("could not prefetch image for show %s", show.Name)
		}
	}
}
