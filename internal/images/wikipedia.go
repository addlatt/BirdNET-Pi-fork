package images

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"
)

const (
	wikipediaMaxWidth = 1024
	userAgent         = "BirdNET-Pi"
)

// WikipediaClient fetches bird images from Wikipedia and Wikimedia Commons.
type WikipediaClient struct {
	client *http.Client
}

// NewWikipediaClient creates a new Wikipedia API client.
func NewWikipediaClient() *WikipediaClient {
	return &WikipediaClient{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// wikipediaSummaryResponse is the response from the Wikipedia REST page summary API.
type wikipediaSummaryResponse struct {
	OriginalImage *struct {
		Source string `json:"source"`
	} `json:"originalimage"`
}

// commonsQueryResponse is the response from the Wikimedia Commons API.
type commonsQueryResponse struct {
	Query *struct {
		Pages map[string]struct {
			ImageInfo []struct {
				ExtMetadata map[string]struct {
					Value interface{} `json:"value"`
				} `json:"extmetadata"`
			} `json:"imageinfo"`
		} `json:"pages"`
	} `json:"query"`
}

// FetchImage fetches a bird image from Wikipedia for the given scientific name.
// Returns nil, nil if no image is found.
func (w *WikipediaClient) FetchImage(sciName, comName string) (*ImageResult, error) {
	// Step 1: Get original image URL from Wikipedia page summary.
	imageURL, err := w.getPageImage(sciName)
	if err != nil {
		return nil, err
	}
	if imageURL == "" {
		return nil, nil
	}

	// Step 2: Extract filename and get metadata from Commons.
	filename := extractFilename(imageURL)
	if filename == "" {
		return nil, nil
	}

	authorURL, licenseURL := w.getCommonsMetadata(filename)

	// Step 3: Resize if needed.
	imageURL = resizeURL(imageURL, filename)

	now := time.Now().UTC().Format(time.RFC3339)
	return &ImageResult{
		SciName:    sciName,
		ComName:    comName,
		Provider:   ProviderWikipedia,
		ImageURL:   imageURL,
		Title:      filename,
		SourceID:   "wiki_" + filename,
		AuthorURL:  authorURL,
		LicenseURL: licenseURL,
		PhotosURL:  "https://en.wikipedia.org/wiki/" + url.PathEscape(sciName),
		CachedAt:   now,
	}, nil
}

// getPageImage calls the Wikipedia REST API to get the original image for a page.
func (w *WikipediaClient) getPageImage(title string) (string, error) {
	apiURL := fmt.Sprintf("https://en.wikipedia.org/api/rest_v1/page/summary/%s", url.PathEscape(title))

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := w.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil // Not found is not an error.
	}

	var summary wikipediaSummaryResponse
	if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
		return "", err
	}

	if summary.OriginalImage == nil {
		return "", nil
	}
	return summary.OriginalImage.Source, nil
}

// getCommonsMetadata fetches artist and license info from the Wikimedia Commons API.
func (w *WikipediaClient) getCommonsMetadata(filename string) (authorURL, licenseURL string) {
	params := url.Values{
		"action":  {"query"},
		"titles":  {"File:" + filename},
		"prop":    {"imageinfo"},
		"iiprop":  {"extmetadata"},
		"format":  {"json"},
	}
	apiURL := "https://commons.wikimedia.org/w/api.php?" + params.Encode()

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", ""
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := w.client.Do(req)
	if err != nil {
		return "", ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", ""
	}

	var result commonsQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", ""
	}

	if result.Query == nil {
		return "", ""
	}

	for _, page := range result.Query.Pages {
		if len(page.ImageInfo) == 0 {
			continue
		}
		meta := page.ImageInfo[0].ExtMetadata

		if artist, ok := meta["Artist"]; ok {
			authorURL = extractURL(fmt.Sprint(artist.Value))
		}
		if license, ok := meta["LicenseUrl"]; ok {
			licenseURL = fmt.Sprint(license.Value)
		}
	}
	return authorURL, licenseURL
}

// extractFilename extracts the file name from a Wikimedia image URL.
// e.g., ".../commons/a/ab/Bird_photo.jpg" → "Bird_photo.jpg"
func extractFilename(imageURL string) string {
	u, err := url.Parse(imageURL)
	if err != nil {
		return ""
	}
	return path.Base(u.Path)
}

// hrefRegex matches href="..." in HTML snippets from Commons metadata.
var hrefRegex = regexp.MustCompile(`href="([^"]+)"`)

// extractURL extracts the first href URL from an HTML string.
// Commons returns artist info as HTML like: <a href="https://...">Author Name</a>
func extractURL(html string) string {
	matches := hrefRegex.FindStringSubmatch(html)
	if len(matches) >= 2 {
		return matches[1]
	}
	// If no href found, the value itself might be a plain URL or text.
	html = strings.TrimSpace(html)
	if strings.HasPrefix(html, "http") {
		return html
	}
	return ""
}

// resizeURL rewrites a Wikimedia image URL to a thumbnail if width > maxWidth.
// Original: /commons/a/ab/Photo.jpg
// Thumb:    /commons/thumb/a/ab/Photo.jpg/1024px-Photo.jpg
func resizeURL(imageURL, filename string) string {
	if !strings.Contains(imageURL, "/commons/") {
		return imageURL
	}
	// Already a thumbnail.
	if strings.Contains(imageURL, "/commons/thumb/") {
		return imageURL
	}
	thumbURL := strings.Replace(imageURL, "/commons/", "/commons/thumb/", 1)
	thumbURL = fmt.Sprintf("%s/%dpx-%s", thumbURL, wikipediaMaxWidth, filename)
	return thumbURL
}
