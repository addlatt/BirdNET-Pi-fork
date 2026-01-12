package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

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
	stats, err := h.db.Queries.GetSpeciesStats(ctx, db.GetSpeciesStatsParams{
		SciName: name,
		ComName: name,
	})
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

// DeleteSpeciesResponse represents the response for deleting a species.
type DeleteSpeciesResponse struct {
	DetectionsDeleted int64 `json:"detections_deleted"`
	FilesDeleted      int   `json:"files_deleted"`
}

// CountSpeciesResponse represents the count of detections and files for a species.
type CountSpeciesResponse struct {
	DetectionCount int64 `json:"detection_count"`
	FileCount      int   `json:"file_count"`
}

// GetSpeciesCount handles GET /api/species/{name}/count requests.
// Returns the count of detections and files for a species before deletion.
func (h *Handlers) GetSpeciesCount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := chi.URLParam(r, "name")

	if name == "" {
		writeError(w, http.StatusBadRequest, "Species name is required")
		return
	}

	// Get detection count
	count, err := h.db.Queries.CountDetectionsBySpecies(ctx, name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to count detections")
		return
	}

	// Get file paths to count files
	files, err := h.db.Queries.GetSpeciesFilePaths(ctx, name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get file paths")
		return
	}

	// Count actual files (audio + spectrogram)
	fileCount := len(files) * 2 // Each detection has an audio file and a spectrogram

	response := CountSpeciesResponse{
		DetectionCount: count,
		FileCount:      fileCount,
	}

	writeJSON(w, http.StatusOK, response)
}

// DeleteAllSpeciesDetections handles DELETE /api/species/{name}/all requests.
// Deletes ALL detections and files for a species (destructive!).
func (h *Handlers) DeleteAllSpeciesDetections(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := chi.URLParam(r, "name")

	if name == "" {
		writeError(w, http.StatusBadRequest, "Species name is required")
		return
	}

	// Get file paths before deleting from database
	files, err := h.db.Queries.GetSpeciesFilePaths(ctx, name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get file paths")
		return
	}

	// Delete files from disk
	filesDeleted := 0
	dirsToClean := make(map[string]bool)

	for _, file := range files {
		// Extract date-only from the date field (format: 2026-01-11 or 2026-01-11T00:00:00Z)
		dateOnly := file.Date
		if idx := strings.Index(dateOnly, "T"); idx != -1 {
			dateOnly = dateOnly[:idx]
		}

		// Build file paths
		comNamePath := sanitizePathComponent(file.ComName)
		basePath := filepath.Join(h.dataDir, "By_Date", dateOnly, comNamePath)
		audioPath := filepath.Join(basePath, file.FileName)
		spectrogramPath := audioPath + ".png"

		// Track directory for cleanup
		dirsToClean[basePath] = true

		// Delete audio file
		if err := os.Remove(audioPath); err == nil {
			filesDeleted++
		}

		// Delete spectrogram
		if err := os.Remove(spectrogramPath); err == nil {
			filesDeleted++
		}
	}

	// Try to clean up empty directories
	for dir := range dirsToClean {
		// Try to remove the species directory if empty
		os.Remove(dir)
		// Try to remove the date directory if empty
		os.Remove(filepath.Dir(dir))
	}

	// Delete detections from database
	result, err := h.db.Queries.DeleteAllDetectionsForSpecies(ctx, name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete detections from database")
		return
	}

	rowsAffected, _ := result.RowsAffected()

	// Also remove from confirmed species list if present
	confirmedPath := filepath.Join(h.scriptsDir, "confirmed_species_list.txt")
	if species, err := readSpeciesListFile(confirmedPath); err == nil {
		var newSpecies []string
		for _, s := range species {
			if s != name {
				newSpecies = append(newSpecies, s)
			}
		}
		if len(newSpecies) != len(species) {
			writeSpeciesListFile(confirmedPath, newSpecies)
		}
	}

	response := DeleteSpeciesResponse{
		DetectionsDeleted: rowsAffected,
		FilesDeleted:      filesDeleted,
	}

	writeJSON(w, http.StatusOK, response)
}

// ListAllSpecies handles GET /api/species/all requests.
// Returns all species with detection count, max confidence, and last seen date.
func (h *Handlers) ListAllSpecies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rows, err := h.db.Queries.ListAllSpeciesWithLastSeen(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch species")
		return
	}

	var speciesList []SpeciesResponse
	for _, row := range rows {
		speciesList = append(speciesList, SpeciesResponse{
			SciName:        row.SciName,
			ComName:        row.ComName,
			DetectionCount: row.DetectionCount,
			MaxConfidence:  toFloat64(row.MaxConfidence),
			LastSeen:       toString(row.LastSeen),
		})
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
