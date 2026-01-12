package api

import (
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// SpectrogramInfoResponse represents spectrogram metadata.
type SpectrogramInfoResponse struct {
	ImageURL       string `json:"image_url"`
	LastModified   string `json:"last_modified,omitempty"`
	Available      bool   `json:"available"`
	LivestreamURL  string `json:"livestream_url"`
	RefreshSeconds int    `json:"refresh_seconds"`
}

// RecentDetection represents a recent detection for the spectrogram page.
type RecentDetection struct {
	Time       string  `json:"time"`
	ComName    string  `json:"com_name"`
	SciName    string  `json:"sci_name"`
	Confidence float64 `json:"confidence"`
	FileName   string  `json:"file_name"`
}

// RecentDetectionsResponse represents the recent detections response.
type RecentDetectionsResponse struct {
	Detections []RecentDetection `json:"detections"`
	Total      int               `json:"total"`
}

// GetSpectrogramInfo handles GET /api/spectrogram/info requests.
// Returns metadata about the current spectrogram image.
func (h *Handlers) GetSpectrogramInfo(w http.ResponseWriter, r *http.Request) {
	// Build the path to spectrogram.png
	// In BirdNET-Pi, this is at ~/BirdSongs/Extracted/spectrogram.png
	spectrogramPath := filepath.Join(h.birdsongsDir, "Extracted", "spectrogram.png")

	// Use the proxied stream URL (Icecast binds to localhost only)
	livestreamURL := "/api/stream"

	response := SpectrogramInfoResponse{
		ImageURL:       "/api/spectrogram/image",
		Available:      false,
		LivestreamURL:  livestreamURL,
		RefreshSeconds: 3, // Default refresh interval
	}

	// Check if spectrogram file exists
	if info, err := os.Stat(spectrogramPath); err == nil {
		response.Available = true
		response.LastModified = info.ModTime().Format(time.RFC3339)
	}

	writeJSON(w, http.StatusOK, response)
}

// GetSpectrogramImage handles GET /api/spectrogram/image requests.
// Serves the spectrogram image file with cache control headers.
func (h *Handlers) GetSpectrogramImage(w http.ResponseWriter, r *http.Request) {
	spectrogramPath := filepath.Join(h.birdsongsDir, "Extracted", "spectrogram.png")

	// Check if file exists
	info, err := os.Stat(spectrogramPath)
	if err != nil {
		writeError(w, http.StatusNotFound, "Spectrogram image not found")
		return
	}

	// Set cache control headers
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("Last-Modified", info.ModTime().Format(http.TimeFormat))

	// Serve the file
	http.ServeFile(w, r, spectrogramPath)
}

// ProxyLivestream handles GET /api/stream requests.
// Proxies the Icecast audio stream to clients (since Icecast binds to localhost).
func (h *Handlers) ProxyLivestream(w http.ResponseWriter, r *http.Request) {
	// Connect to local Icecast
	resp, err := http.Get("http://127.0.0.1:8000/stream")
	if err != nil {
		writeError(w, http.StatusBadGateway, "Failed to connect to audio stream")
		return
	}
	defer resp.Body.Close()

	// Copy headers
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.Header().Set("Cache-Control", "no-cache, no-store")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Stream the audio
	w.WriteHeader(resp.StatusCode)

	// Use a buffer for efficient copying
	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				return // Client disconnected
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
		if err != nil {
			return
		}
	}
}

// GetRecentDetections handles GET /api/spectrogram/detections requests.
// Returns the most recent detections for display on the spectrogram page.
// Query parameters:
//   - limit: Maximum number of detections to return (default: 10, max: 50)
func (h *Handlers) GetRecentDetections(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	limit := parseIntParam(r.URL.Query().Get("limit"), 10)
	if limit > 50 {
		limit = 50
	}
	if limit < 1 {
		limit = 10
	}

	// Get today's date
	today := time.Now().Format("2006-01-02")

	// Query recent detections from today
	detections, err := h.db.Queries.ListDetectionsByDate(ctx, today)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch recent detections")
		return
	}

	// Limit results and convert to response format
	response := RecentDetectionsResponse{
		Detections: make([]RecentDetection, 0, limit),
		Total:      len(detections),
	}

	for i, d := range detections {
		if i >= limit {
			break
		}
		response.Detections = append(response.Detections, RecentDetection{
			Time:       d.Time,
			ComName:    d.ComName,
			SciName:    d.SciName,
			Confidence: d.Confidence.Float64,
			FileName:   d.FileName,
		})
	}

	writeJSON(w, http.StatusOK, response)
}
