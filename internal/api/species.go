package api

import (
	"net/http"
)

// SpeciesResponse represents a species in API responses.
type SpeciesResponse struct {
	SciName        string  `json:"sci_name"`
	ComName        string  `json:"com_name"`
	DetectionCount int64   `json:"detection_count"`
	MaxConfidence  float64 `json:"max_confidence"`
}

// ListSpeciesResponse represents the list species response.
type ListSpeciesResponse struct {
	Species []SpeciesResponse `json:"species"`
	Total   int               `json:"total"`
}

// ListSpecies handles GET /api/species requests.
func (h *Handlers) ListSpecies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check if we want today's species only
	todayOnly := r.URL.Query().Get("today") == "true"

	var speciesList []SpeciesResponse
	var err error

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
				MaxConfidence:  row.MaxConfidence,
			})
		}
	} else {
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
				MaxConfidence:  row.MaxConfidence,
			})
		}
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch species")
		return
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
