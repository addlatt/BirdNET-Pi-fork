package api

import (
	"database/sql"
	"net/http"
	"strconv"

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
	Total      int64              `json:"total"`
	Page       int                `json:"page"`
	PerPage    int                `json:"per_page"`
}

// ListDetections handles GET /api/detections requests.
func (h *Handlers) ListDetections(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	page := parseIntParam(r.URL.Query().Get("page"), 1)
	perPage := parseIntParam(r.URL.Query().Get("per_page"), 20)
	date := r.URL.Query().Get("date")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	species := r.URL.Query().Get("species")

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

	offset := int64((page - 1) * perPage)
	limit := int64(perPage)

	var detections []db.Detection
	var err error

	// Query based on filters
	if date != "" {
		detections, err = h.db.Queries.ListDetectionsByDate(ctx, date)
	} else if startDate != "" && endDate != "" {
		detections, err = h.db.Queries.ListDetectionsByDateRange(ctx, db.ListDetectionsByDateRangeParams{
			StartDate: startDate,
			EndDate:   endDate,
			Limit:     limit,
			Offset:    offset,
		})
	} else if species != "" {
		detections, err = h.db.Queries.ListDetectionsBySpecies(ctx, db.ListDetectionsBySpeciesParams{
			SciName: species,
			ComName: species,
			Limit:   limit,
			Offset:  offset,
		})
	} else {
		detections, err = h.db.Queries.ListDetections(ctx, db.ListDetectionsParams{
			Limit:  limit,
			Offset: offset,
		})
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch detections")
		return
	}

	// Get total count
	total, err := h.db.Queries.CountDetections(ctx)
	if err != nil {
		total = 0 // Non-fatal
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

	detection, err := h.db.Queries.GetDetection(ctx, db.GetDetectionParams{
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

// detectionToResponse converts a database detection to API response format.
func detectionToResponse(d db.Detection) DetectionResponse {
	resp := DetectionResponse{
		Date:       d.Date,
		Time:       d.Time,
		SciName:    d.SciName,
		ComName:    d.ComName,
		Confidence: d.Confidence,
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
