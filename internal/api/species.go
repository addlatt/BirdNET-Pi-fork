package api

import (
	"net/http"

	db "github.com/birdnet-pi/birdnet/internal/db/generated"
	"github.com/go-chi/chi/v5"
)

// toFloat64 converts an interface{} to float64, returning 0 if conversion fails.
func toFloat64(v interface{}) float64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int64:
		return float64(val)
	case int:
		return float64(val)
	default:
		return 0
	}
}

// toString converts an interface{} to string, returning empty string if conversion fails.
func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// SpeciesResponse represents a species in API responses.
type SpeciesResponse struct {
	SciName        string  `json:"sci_name"`
	ComName        string  `json:"com_name"`
	DetectionCount int64   `json:"detection_count"`
	MaxConfidence  float64 `json:"max_confidence"`
	LastSeen       string  `json:"last_seen,omitempty"`
}

// ListSpeciesResponse represents the list species response.
type ListSpeciesResponse struct {
	Species []SpeciesResponse `json:"species"`
	Total   int               `json:"total"`
}

// SpeciesDetailResponse represents detailed species information.
type SpeciesDetailResponse struct {
	SciName        string  `json:"sci_name"`
	ComName        string  `json:"com_name"`
	DetectionCount int64   `json:"detection_count"`
	MaxConfidence  float64 `json:"max_confidence"`
	BestDate       string  `json:"best_date"`
	BestTime       string  `json:"best_time"`
	BestFileName   string  `json:"best_file_name"`
	AudioURL       string  `json:"audio_url"`
	SpectrogramURL string  `json:"spectrogram_url"`
}

// ListSpecies handles GET /api/species requests.
// Query parameters:
//   - today: "true" to get only today's species
//   - sort: "alphabetical", "occurrences" (default), "confidence", "date"
func (h *Handlers) ListSpecies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check if we want today's species only
	todayOnly := r.URL.Query().Get("today") == "true"
	sortBy := r.URL.Query().Get("sort")
	if sortBy == "" {
		sortBy = "occurrences"
	}

	var speciesList []SpeciesResponse

	if todayOnly {
		rows, err := h.db.Queries.ListSpeciesToday(ctx)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to fetch species")
			return
		}
		for _, row := range rows {
			speciesList = append(speciesList, SpeciesResponse{
				SciName:        row.SciName,
				ComName:        row.ComName,
				DetectionCount: row.DetectionCount,
				MaxConfidence:  toFloat64(row.MaxConfidence),
			})
		}
	} else {
		// Fetch based on sort order
		switch sortBy {
		case "alphabetical":
			rows, err := h.db.Queries.ListSpeciesSortedByAlphabetical(ctx)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "Failed to fetch species")
				return
			}
			for _, row := range rows {
				speciesList = append(speciesList, SpeciesResponse{
					SciName:        row.SciName,
					ComName:        row.ComName,
					DetectionCount: row.DetectionCount,
					MaxConfidence:  toFloat64(row.MaxConfidence),
				})
			}
		case "confidence":
			rows, err := h.db.Queries.ListSpeciesSortedByConfidence(ctx)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "Failed to fetch species")
				return
			}
			for _, row := range rows {
				speciesList = append(speciesList, SpeciesResponse{
					SciName:        row.SciName,
					ComName:        row.ComName,
					DetectionCount: row.DetectionCount,
					MaxConfidence:  toFloat64(row.MaxConfidence),
				})
			}
		case "date":
			rows, err := h.db.Queries.ListSpeciesSortedByDate(ctx)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "Failed to fetch species")
				return
			}
			for _, row := range rows {
				speciesList = append(speciesList, SpeciesResponse{
					SciName:        row.SciName,
					ComName:        row.ComName,
					DetectionCount: row.DetectionCount,
					MaxConfidence:  toFloat64(row.MaxConfidence),
					LastSeen:       toString(row.LastSeen),
				})
			}
		default: // "occurrences" - default sort
			rows, err := h.db.Queries.ListSpecies(ctx)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "Failed to fetch species")
				return
			}
			for _, row := range rows {
				speciesList = append(speciesList, SpeciesResponse{
					SciName:        row.SciName,
					ComName:        row.ComName,
					DetectionCount: row.DetectionCount,
					MaxConfidence:  toFloat64(row.MaxConfidence),
				})
			}
		}
	}

	// Ensure we return empty array instead of null
	if speciesList == nil {
		speciesList = []SpeciesResponse{}
	}

	response := ListSpeciesResponse{
		Species: speciesList,
		Total:   len(speciesList),
	}

	writeJSON(w, http.StatusOK, response)
}

// GetSpeciesDetail handles GET /api/species/{name} requests.
// Returns detailed information about a species including best detection.
func (h *Handlers) GetSpeciesDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := chi.URLParam(r, "name")

	if name == "" {
		writeError(w, http.StatusBadRequest, "Species name is required")
		return
	}

	// Get best detection for this species
	best, err := h.db.Queries.GetBestDetectionForSpecies(ctx, db.GetBestDetectionForSpeciesParams{
		SciName: name,
		ComName: name,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "Species not found")
		return
	}

	// Get species stats
	stats, err := h.db.Queries.GetSpeciesStats(ctx, name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch species stats")
		return
	}

	// Build audio URL path
	comNamePath := sanitizePathComponent(best.ComName)
	audioURL := "/By_Date/" + best.Date + "/" + comNamePath + "/" + best.FileName
	spectrogramURL := audioURL + ".png"

	response := SpeciesDetailResponse{
		SciName:        best.SciName,
		ComName:        best.ComName,
		DetectionCount: stats.TotalDetections,
		MaxConfidence:  toFloat64(stats.MaxConfidence),
		BestDate:       best.Date,
		BestTime:       best.Time,
		BestFileName:   best.FileName,
		AudioURL:       audioURL,
		SpectrogramURL: spectrogramURL,
	}

	writeJSON(w, http.StatusOK, response)
}

// sanitizePathComponent replaces spaces with underscores and removes apostrophes
// to match the file path format used by the system.
func sanitizePathComponent(s string) string {
	result := ""
	for _, c := range s {
		switch c {
		case ' ':
			result += "_"
		case '\'':
			// skip apostrophes
		default:
			result += string(c)
		}
	}
	return result
}
