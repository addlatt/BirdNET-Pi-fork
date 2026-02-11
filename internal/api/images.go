package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// GetSpeciesImage returns a cached or freshly-fetched image for a species.
// GET /api/species/{name}/image?provider=&com_name=
func (h *Handlers) GetSpeciesImage(w http.ResponseWriter, r *http.Request) {
	if h.imageService == nil {
		writeError(w, http.StatusServiceUnavailable, "image service not initialized")
		return
	}

	sciName := chi.URLParam(r, "name")
	if sciName == "" {
		writeError(w, http.StatusBadRequest, "species name is required")
		return
	}

	provider := r.URL.Query().Get("provider")
	comName := r.URL.Query().Get("com_name")

	result, err := h.imageService.GetImage(sciName, comName, provider)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get image: "+err.Error())
		return
	}

	if result == nil {
		writeError(w, http.StatusNotFound, "no image found for species")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// BlacklistSpeciesImage blacklists the current image and returns a replacement.
// POST /api/species/{name}/image/blacklist
func (h *Handlers) BlacklistSpeciesImage(w http.ResponseWriter, r *http.Request) {
	if h.imageService == nil {
		writeError(w, http.StatusServiceUnavailable, "image service not initialized")
		return
	}

	sciName := chi.URLParam(r, "name")
	if sciName == "" {
		writeError(w, http.StatusBadRequest, "species name is required")
		return
	}

	var body struct {
		Provider string `json:"provider"`
		ComName  string `json:"com_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.imageService.BlacklistAndRefresh(sciName, body.ComName, body.Provider)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to blacklist image: "+err.Error())
		return
	}

	if result == nil {
		writeJSON(w, http.StatusOK, map[string]string{
			"status":  "blacklisted",
			"message": "image blacklisted but no replacement found",
		})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GetImageCacheStats returns aggregate counts for the image cache.
// GET /api/images/cache/stats
func (h *Handlers) GetImageCacheStats(w http.ResponseWriter, r *http.Request) {
	if h.imageService == nil {
		writeError(w, http.StatusServiceUnavailable, "image service not initialized")
		return
	}

	stats, err := h.imageService.GetStats()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get cache stats: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// RefreshImageCache launches a background refresh of expired cache entries.
// POST /api/images/cache/refresh
func (h *Handlers) RefreshImageCache(w http.ResponseWriter, r *http.Request) {
	if h.imageService == nil {
		writeError(w, http.StatusServiceUnavailable, "image service not initialized")
		return
	}

	go h.imageService.RefreshExpired()

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "refresh started",
	})
}
