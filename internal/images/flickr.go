package images

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	flickrAPIBase    = "https://api.flickr.com/services/rest/"
	flickrHardcodedSkip = "4892923285" // Hardcoded ID to always skip.
)

// FlickrClient fetches bird images from the Flickr API.
type FlickrClient struct {
	apiKey   string
	client   *http.Client

	// Cached license map: license ID → URL.
	licenses   map[string]string
	licensesMu sync.Mutex

	// Cached NSID resolution: email → NSID.
	nsidCache   map[string]string
	nsidCacheMu sync.Mutex
}

// NewFlickrClient creates a new Flickr API client.
// If apiKey is empty, all methods return nil gracefully.
func NewFlickrClient(apiKey string) *FlickrClient {
	return &FlickrClient{
		apiKey:    apiKey,
		client:    &http.Client{Timeout: 10 * time.Second},
		licenses:  make(map[string]string),
		nsidCache: make(map[string]string),
	}
}

// flickrResponse wraps the common Flickr JSON envelope.
type flickrResponse struct {
	Stat string `json:"stat"`
}

type flickrSearchResponse struct {
	Photos struct {
		Photo []struct {
			ID     string `json:"id"`
			Secret string `json:"secret"`
			Server string `json:"server"`
			Farm   int    `json:"farm"`
			Title  string `json:"title"`
			Owner  string `json:"owner"`
		} `json:"photo"`
	} `json:"photos"`
	Stat string `json:"stat"`
}

type flickrPhotoInfoResponse struct {
	Photo struct {
		License string `json:"license"`
		Owner   struct {
			NSID     string `json:"nsid"`
			Username string `json:"username"`
			PhotosURL string `json:"photosurl"`
		} `json:"owner"`
		URLs struct {
			URL []struct {
				Content string `json:"_content"`
			} `json:"url"`
		} `json:"urls"`
	} `json:"photo"`
	Stat string `json:"stat"`
}

type flickrLicensesResponse struct {
	Licenses struct {
		License []struct {
			ID  string `json:"id"`
			URL string `json:"url"`
		} `json:"license"`
	} `json:"licenses"`
	Stat string `json:"stat"`
}

type flickrFindByEmailResponse struct {
	User struct {
		NSID string `json:"nsid"`
	} `json:"user"`
	Stat string `json:"stat"`
}

// FetchImage searches Flickr for a bird image matching the given names.
// isBlacklisted is called to check if a photo ID should be skipped.
// filterEmail, if non-empty, restricts results to that user.
// Returns nil, nil if no API key or no suitable image found.
func (f *FlickrClient) FetchImage(sciName, comName string, isBlacklisted func(string) bool, filterEmail string) (*ImageResult, error) {
	if f.apiKey == "" {
		return nil, nil
	}

	// Resolve filter email to NSID if provided.
	var userID string
	if filterEmail != "" {
		nsid, err := f.findByEmail(filterEmail)
		if err == nil && nsid != "" {
			userID = nsid
		}
	}

	// Search for photos.
	params := url.Values{
		"method":         {"flickr.photos.search"},
		"api_key":        {f.apiKey},
		"text":           {comName + " bird"},
		"sort":           {"relevance"},
		"per_page":       {"5"},
		"format":         {"json"},
		"nojsoncallback": {"1"},
	}

	if userID != "" {
		params.Set("user_id", userID)
	} else {
		params.Set("license", "2,3,4,5,6,9")
		params.Set("orientation", "square,portrait")
	}

	var searchResp flickrSearchResponse
	if err := f.apiCall(params, &searchResp); err != nil {
		return nil, err
	}

	// Find the first non-blacklisted photo.
	for _, photo := range searchResp.Photos.Photo {
		if photo.ID == flickrHardcodedSkip {
			continue
		}
		if isBlacklisted(photo.ID) {
			continue
		}

		// Get photo info for license and owner details.
		info, err := f.getPhotoInfo(photo.ID)
		if err != nil {
			continue
		}

		// Resolve license URL.
		licenseURL, err := f.getLicenseURL(info.Photo.License)
		if err != nil {
			licenseURL = ""
		}

		// Build image URL.
		imageURL := fmt.Sprintf("https://farm%d.static.flickr.com/%s/%s_%s.jpg",
			photo.Farm, photo.Server, photo.ID, photo.Secret)

		// Build owner photos URL.
		photosURL := info.Photo.Owner.PhotosURL
		if photosURL == "" && len(info.Photo.URLs.URL) > 0 {
			photosURL = info.Photo.URLs.URL[0].Content
		}

		// Build author URL.
		authorURL := ""
		if info.Photo.Owner.NSID != "" {
			authorURL = fmt.Sprintf("https://www.flickr.com/people/%s", info.Photo.Owner.NSID)
		}

		now := time.Now().UTC().Format(time.RFC3339)
		return &ImageResult{
			SciName:    sciName,
			ComName:    comName,
			Provider:   ProviderFlickr,
			ImageURL:   imageURL,
			Title:      photo.Title,
			SourceID:   photo.ID,
			AuthorURL:  authorURL,
			LicenseURL: licenseURL,
			PhotosURL:  photosURL,
			CachedAt:   now,
		}, nil
	}

	return nil, nil
}

func (f *FlickrClient) getPhotoInfo(photoID string) (*flickrPhotoInfoResponse, error) {
	params := url.Values{
		"method":         {"flickr.photos.getInfo"},
		"api_key":        {f.apiKey},
		"photo_id":       {photoID},
		"format":         {"json"},
		"nojsoncallback": {"1"},
	}

	var resp flickrPhotoInfoResponse
	if err := f.apiCall(params, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (f *FlickrClient) getLicenseURL(licenseID string) (string, error) {
	f.licensesMu.Lock()
	defer f.licensesMu.Unlock()

	// Return from cache if available.
	if u, ok := f.licenses[licenseID]; ok {
		return u, nil
	}

	// Fetch all licenses.
	if len(f.licenses) == 0 {
		params := url.Values{
			"method":         {"flickr.photos.licenses.getInfo"},
			"api_key":        {f.apiKey},
			"format":         {"json"},
			"nojsoncallback": {"1"},
		}

		var resp flickrLicensesResponse
		if err := f.apiCall(params, &resp); err != nil {
			return "", err
		}

		for _, lic := range resp.Licenses.License {
			f.licenses[lic.ID] = lic.URL
		}
	}

	return f.licenses[licenseID], nil
}

func (f *FlickrClient) findByEmail(email string) (string, error) {
	f.nsidCacheMu.Lock()
	defer f.nsidCacheMu.Unlock()

	if nsid, ok := f.nsidCache[email]; ok {
		return nsid, nil
	}

	params := url.Values{
		"method":         {"flickr.people.findByEmail"},
		"api_key":        {f.apiKey},
		"find_email":     {email},
		"format":         {"json"},
		"nojsoncallback": {"1"},
	}

	var resp flickrFindByEmailResponse
	if err := f.apiCall(params, &resp); err != nil {
		return "", err
	}

	f.nsidCache[email] = resp.User.NSID
	return resp.User.NSID, nil
}

func (f *FlickrClient) apiCall(params url.Values, result interface{}) error {
	resp, err := f.client.Get(flickrAPIBase + "?" + params.Encode())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("flickr API returned status %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(result)
}
