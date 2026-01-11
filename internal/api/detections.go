package api

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	db "github.com/birdnet-pi/birdnet/internal/db/generated"
	"github.com/go-chi/chi/v5"
)

// DetectionResponse represents a single detection in API responses.
type DetectionResponse struct {
	Date       string   `json:"date"`
	Time       string   `json:"time"`
	SciName    string   `json:"sci_name"`
	ComName    string   `json:"com_name"`
	Confidence float64  `json:"confidence"`
	Lat        *float64 `json:"lat,omitempty"`
	Lon        *float64 `json:"lon,omitempty"`
	FileName   string   `json:"file_name"`
}

// ListDetectionsResponse represents the list detections response.
type ListDetectionsResponse struct {
	Detections []DetectionResponse `json:"detections"`
	Total      int64               `json:"total"`
	Page       int                 `json:"page"`
	PerPage    int                 `json:"per_page"`
}

// SpeciesHistoryResponse represents daily detection counts for a species.
type SpeciesHistoryResponse struct {
	Species string                   `json:"species"`
	Days    int                      `json:"days"`
	History []SpeciesHistoryDayEntry `json:"history"`
}

// SpeciesHistoryDayEntry represents a single day's detection count.
type SpeciesHistoryDayEntry struct {
	Date           string `json:"date"`
	DetectionCount int64  `json:"detection_count"`
}

// ListDetections handles GET /api/detections requests.
// Query parameters:
//   - page: Page number (default: 1)
//   - per_page: Results per page (default: 20, max: 100)
//   - date: Filter by specific date (YYYY-MM-DD)
//   - start_date, end_date: Filter by date range
//   - species: Filter by species (scientific or common name)
//   - search: Text search across com_name, sci_name, file_name, time
//   - min_confidence: Minimum confidence threshold (0.0-1.0)
func (h *Handlers) ListDetections(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	page := parseIntParam(r.URL.Query().Get("page"), 1)
	perPage := parseIntParam(r.URL.Query().Get("per_page"), 20)
	date := r.URL.Query().Get("date")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	species := r.URL.Query().Get("species")
	search := r.URL.Query().Get("search")
	minConfidence := parseFloatParam(r.URL.Query().Get("min_confidence"), 0.0)

	// Ensure reasonable limits
	if perPage > 100 {
		perPage = 100
	}
	if perPage < 1 {
		perPage = 20
	}
	if page < 1 {
		page = 1
	}
	if minConfidence < 0 {
		minConfidence = 0
	}
	if minConfidence > 1 {
		minConfidence = 1
	}

	offset := int64((page - 1) * perPage)
	limit := int64(perPage)

	var detections []db.Detection
	var total int64
	var err error

	// Convert minConfidence to sql.NullFloat64 for sqlc queries
	minConfidenceSQL := sql.NullFloat64{Float64: minConfidence, Valid: true}

	// Handle search queries (most specific case)
	if date != "" && search != "" {
		// Text search within a specific date
		searchPattern := "%" + search + "%"
		detections, err = h.db.Queries.SearchDetectionsByDate(ctx, db.SearchDetectionsByDateParams{
			Date:       date,
			ComName:    searchPattern,
			SciName:    searchPattern,
			FileName:   searchPattern,
			Time:       searchPattern,
			Confidence: minConfidenceSQL,
			Limit:      limit,
			Offset:     offset,
		})
		if err == nil {
			total, _ = h.db.Queries.CountSearchDetectionsByDate(ctx, db.CountSearchDetectionsByDateParams{
				Date:       date,
				ComName:    searchPattern,
				SciName:    searchPattern,
				FileName:   searchPattern,
				Time:       searchPattern,
				Confidence: minConfidenceSQL,
			})
		}
	} else if date != "" && minConfidence > 0 {
		// Date with confidence filter
		detections, err = h.db.Queries.ListDetectionsByDateWithConfidence(ctx, db.ListDetectionsByDateWithConfidenceParams{
			Date:       date,
			Confidence: minConfidenceSQL,
			Limit:      limit,
			Offset:     offset,
		})
		if err == nil {
			total, _ = h.db.Queries.CountDetectionsByDateWithConfidence(ctx, db.CountDetectionsByDateWithConfidenceParams{
				Date:       date,
				Confidence: minConfidenceSQL,
			})
		}
	} else if date != "" {
		// Simple date filter (no pagination in original query, so we handle it)
		allDetections, queryErr := h.db.Queries.ListDetectionsByDate(ctx, date)
		err = queryErr
		if err == nil {
			total = int64(len(allDetections))
			// Apply pagination manually
			start := int(offset)
			end := start + int(limit)
			if start > len(allDetections) {
				detections = []db.Detection{}
			} else {
				if end > len(allDetections) {
					end = len(allDetections)
				}
				detections = allDetections[start:end]
			}
		}
	} else if startDate != "" && endDate != "" {
		detections, err = h.db.Queries.ListDetectionsByDateRange(ctx, db.ListDetectionsByDateRangeParams{
			Date:   startDate,
			Date_2: endDate,
			Limit:  limit,
			Offset: offset,
		})
		if err == nil {
			// For date range, we don't have a count query, estimate from results
			total = int64(len(detections))
			if len(detections) == int(limit) {
				total = offset + limit + 1 // Indicate there might be more
			}
		}
	} else if species != "" {
		detections, err = h.db.Queries.ListDetectionsBySpecies(ctx, db.ListDetectionsBySpeciesParams{
			SciName: species,
			ComName: species,
			Limit:   limit,
			Offset:  offset,
		})
		if err == nil {
			// Estimate total from results
			total = int64(len(detections))
			if len(detections) == int(limit) {
				total = offset + limit + 1
			}
		}
	} else {
		detections, err = h.db.Queries.ListDetections(ctx, db.ListDetectionsParams{
			Limit:  limit,
			Offset: offset,
		})
		if err == nil {
			total, _ = h.db.Queries.CountDetections(ctx)
		}
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch detections")
		return
	}

	// Convert to response format
	response := ListDetectionsResponse{
		Detections: make([]DetectionResponse, 0, len(detections)),
		Total:      total,
		Page:       page,
		PerPage:    perPage,
	}

	for _, d := range detections {
		response.Detections = append(response.Detections, detectionToResponse(d))
	}

	writeJSON(w, http.StatusOK, response)
}

// GetDetection handles GET /api/detections/{date}/{time}/{species} requests.
// Since the database has no ID column, we use a composite key.
func (h *Handlers) GetDetection(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	date := chi.URLParam(r, "date")
	time := chi.URLParam(r, "time")
	species := chi.URLParam(r, "species")

	if date == "" || time == "" || species == "" {
		writeError(w, http.StatusBadRequest, "Missing date, time, or species parameter")
		return
	}

	detection, err := h.db.Queries.GetDetectionByCompositeKey(ctx, db.GetDetectionByCompositeKeyParams{
		Date:    date,
		Time:    time,
		SciName: species,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "Detection not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to fetch detection")
		return
	}

	writeJSON(w, http.StatusOK, detectionToResponse(detection))
}

// DeleteDetection handles DELETE /api/detections/{date}/{time}/{species} requests.
// Deletes a detection by its composite key.
func (h *Handlers) DeleteDetection(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	date := chi.URLParam(r, "date")
	timeParam := chi.URLParam(r, "time")
	species := chi.URLParam(r, "species")

	if date == "" || timeParam == "" || species == "" {
		writeError(w, http.StatusBadRequest, "Missing date, time, or species parameter")
		return
	}

	// First check if the detection exists
	_, err := h.db.Queries.GetDetectionByCompositeKey(ctx, db.GetDetectionByCompositeKeyParams{
		Date:    date,
		Time:    timeParam,
		SciName: species,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "Detection not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to verify detection")
		return
	}

	// Delete the detection
	err = h.db.Queries.DeleteDetectionByCompositeKey(ctx, db.DeleteDetectionByCompositeKeyParams{
		Date:    date,
		Time:    timeParam,
		SciName: species,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete detection")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// GetSpeciesHistory handles GET /api/species/{name}/history requests.
// Returns daily detection counts for a species over the specified number of days.
// Query parameters:
//   - days: Number of days to include (default: 30, max: 365)
func (h *Handlers) GetSpeciesHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	name := chi.URLParam(r, "name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "Missing species name")
		return
	}

	days := parseIntParam(r.URL.Query().Get("days"), 30)
	if days < 1 {
		days = 30
	}
	if days > 365 {
		days = 365
	}

	// Calculate start date (days ago from today)
	startDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	// Get species detection history
	history, err := h.db.Queries.GetSpeciesDetectionHistory(ctx, db.GetSpeciesDetectionHistoryParams{
		ComName: name,
		SciName: name,
		Date:    startDate,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch species history")
		return
	}

	// Build response
	response := SpeciesHistoryResponse{
		Species: name,
		Days:    days,
		History: make([]SpeciesHistoryDayEntry, 0, len(history)),
	}

	for _, entry := range history {
		response.History = append(response.History, SpeciesHistoryDayEntry{
			Date:           entry.Date,
			DetectionCount: entry.DetectionCount,
		})
	}

	writeJSON(w, http.StatusOK, response)
}

// detectionToResponse converts a database detection to API response format.
func detectionToResponse(d db.Detection) DetectionResponse {
	resp := DetectionResponse{
		Date:       d.Date,
		Time:       d.Time,
		SciName:    d.SciName,
		ComName:    d.ComName,
		Confidence: d.Confidence.Float64, // Extract float64 from sql.NullFloat64
		FileName:   d.FileName,
	}

	if d.Lat.Valid {
		lat := d.Lat.Float64
		resp.Lat = &lat
	}
	if d.Lon.Valid {
		lon := d.Lon.Float64
		resp.Lon = &lon
	}

	return resp
}

// parseIntParam parses an integer query parameter with a default value.
func parseIntParam(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}

// parseFloatParam parses a float query parameter with a default value.
func parseFloatParam(s string, defaultVal float64) float64 {
	if s == "" {
		return defaultVal
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return defaultVal
	}
	return v
}
