package images

// Provider constants for image sources.
const (
	ProviderFlickr    = "flickr"
	ProviderWikipedia = "wikipedia"
	ProviderAuto      = "auto"
)

// ImageResult holds a cached or freshly-fetched image with attribution metadata.
type ImageResult struct {
	SciName    string `json:"sci_name"`
	ComName    string `json:"com_name"`
	Provider   string `json:"provider"`
	ImageURL   string `json:"image_url"`
	Title      string `json:"title"`
	SourceID   string `json:"source_id"`
	AuthorURL  string `json:"author_url"`
	LicenseURL string `json:"license_url"`
	PhotosURL  string `json:"photos_url"`
	CachedAt   string `json:"cached_at"`
}

// CacheStats holds aggregate counts for the image cache.
type CacheStats struct {
	FlickrCount    int `json:"flickr_count"`
	WikipediaCount int `json:"wikipedia_count"`
	TotalCount     int `json:"total_count"`
	ExpiredCount   int `json:"expired_count"`
}
