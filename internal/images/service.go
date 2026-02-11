package images

import (
	"fmt"
	"log"
	"time"

	"github.com/birdnet-pi/birdnet/internal/config"
)

// Service orchestrates image fetching, caching, and blacklist management.
type Service struct {
	cache     *Cache
	wikipedia *WikipediaClient
	flickr    *FlickrClient
	configMgr *config.Manager
}

// NewService creates a new image service with cache and provider clients.
func NewService(dataDir string, configMgr *config.Manager) (*Service, error) {
	cache, err := NewCache(dataDir)
	if err != nil {
		return nil, fmt.Errorf("init image cache: %w", err)
	}

	cfg := configMgr.Get()
	flickrClient := NewFlickrClient(cfg.FlickrAPIKey)

	return &Service{
		cache:     cache,
		wikipedia: NewWikipediaClient(),
		flickr:    flickrClient,
		configMgr: configMgr,
	}, nil
}

// GetImage returns an image for the species, checking cache first then fetching.
// providerHint can be "flickr", "wikipedia", or "auto" (uses config default).
func (s *Service) GetImage(sciName, comName, providerHint string) (*ImageResult, error) {
	provider := s.resolveProvider(providerHint)

	// Check cache.
	cached, err := s.cache.Get(sciName, provider)
	if err != nil {
		return nil, fmt.Errorf("cache lookup: %w", err)
	}
	if cached != nil {
		return cached, nil
	}

	// Fetch from source.
	result, err := s.fetchFromProvider(sciName, comName, provider)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}

	// Cache the result.
	if err := s.cache.Set(result); err != nil {
		log.Printf("Warning: failed to cache image for %s: %v", sciName, err)
	}

	return result, nil
}

// BlacklistAndRefresh blacklists the current image and fetches a replacement.
func (s *Service) BlacklistAndRefresh(sciName, comName, provider string) (*ImageResult, error) {
	provider = s.resolveProvider(provider)

	// Get current cached image to find its source ID.
	cached, err := s.cache.Get(sciName, provider)
	if err != nil {
		return nil, fmt.Errorf("cache lookup: %w", err)
	}

	if cached != nil {
		if err := s.cache.AddToBlacklist(cached.SourceID); err != nil {
			return nil, fmt.Errorf("add to blacklist: %w", err)
		}
	}

	// Delete from cache.
	if err := s.cache.Delete(sciName, provider); err != nil {
		return nil, fmt.Errorf("delete from cache: %w", err)
	}

	// Fetch a new image.
	result, err := s.fetchFromProvider(sciName, comName, provider)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}

	// Cache the new result.
	if err := s.cache.Set(result); err != nil {
		log.Printf("Warning: failed to cache replacement image for %s: %v", sciName, err)
	}

	return result, nil
}

// GetStats returns cache statistics.
func (s *Service) GetStats() (*CacheStats, error) {
	return s.cache.GetStats()
}

// RefreshExpired iterates expired cache entries and re-fetches each.
func (s *Service) RefreshExpired() {
	expired, err := s.cache.ListExpired()
	if err != nil {
		log.Printf("Error listing expired images: %v", err)
		return
	}

	log.Printf("Refreshing %d expired image cache entries", len(expired))
	for _, entry := range expired {
		result, err := s.fetchFromProvider(entry.SciName, entry.ComName, entry.Provider)
		if err != nil {
			log.Printf("Error refreshing image for %s (%s): %v", entry.SciName, entry.Provider, err)
			continue
		}
		if result != nil {
			if err := s.cache.Set(result); err != nil {
				log.Printf("Error caching refreshed image for %s: %v", entry.SciName, err)
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	log.Printf("Image cache refresh complete")
}

// Close closes the underlying cache.
func (s *Service) Close() error {
	return s.cache.Close()
}

// resolveProvider maps a provider hint to a concrete provider name.
func (s *Service) resolveProvider(hint string) string {
	switch hint {
	case ProviderFlickr, ProviderWikipedia:
		return hint
	default:
		// Use config default.
		cfg := s.configMgr.Get()
		switch cfg.ImageProvider {
		case ProviderFlickr:
			return ProviderFlickr
		default:
			return ProviderWikipedia
		}
	}
}

// fetchFromProvider calls the appropriate provider client.
func (s *Service) fetchFromProvider(sciName, comName, provider string) (*ImageResult, error) {
	switch provider {
	case ProviderFlickr:
		cfg := s.configMgr.Get()
		return s.flickr.FetchImage(sciName, comName, s.cache.IsBlacklisted, cfg.FlickrFilterEmail)
	default:
		return s.wikipedia.FetchImage(sciName, comName)
	}
}
