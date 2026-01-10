package api

import (
	"net/http"
	"time"

	"github.com/birdnet-pi/birdnet/internal/db/generated"
)

// StatsResponse represents overall statistics.
type StatsResponse struct {
	TotalDetections      int64               `json:"total_detections"`
	TotalSpecies         int64               `json:"total_species"`
	DetectionsToday      int64               `json:"detections_today"`
	SpeciesToday         int64               `json:"species_today"`
	DailyStats           []DailyStatResponse `json:"daily_stats,omitempty"`
	HourlyDistribution   []HourlyStatResponse `json:"hourly_distribution,omitempty"`
	TopSpecies           []TopSpeciesResponse `json:"top_species,omitempty"`
}

// DailyStatResponse represents daily statistics.
type DailyStatResponse struct {
	Date           string  `json:"date"`
	DetectionCount int64   `json:"detection_count"`
	SpeciesCount   int64   `json:"species_count"`
	AvgConfidence  float64 `json:"avg_confidence"`
}

// HourlyStatResponse represents hourly distribution.
type HourlyStatResponse struct {
	Hour           int   `json:"hour"`
	DetectionCount int64 `json:"detection_count"`
}

// TopSpeciesResponse represents a top species entry.
type TopSpeciesResponse struct {
	SciName        string `json:"sci_name"`
	ComName        string `json:"com_name"`
	DetectionCount int64  `json:"detection_count"`
}

// GetStats handles GET /api/stats requests.
func (h *Handlers) GetStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters for date range
	days := parseIntParam(r.URL.Query().Get("days"), 7)
	if days > 365 {
		days = 365
	}
	if days < 1 {
		days = 7
	}

	startDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	response := StatsResponse{}

	// Get total counts
	total, err := h.db.Queries.CountDetections(ctx)
	if err == nil {
		response.TotalDetections = total
	}

	totalSpecies, err := h.db.Queries.GetTotalSpeciesCount(ctx)
	if err == nil {
		response.TotalSpecies = totalSpecies
	}

	// Get today's counts
	todayCount, err := h.db.Queries.CountDetectionsToday(ctx)
	if err == nil {
		response.DetectionsToday = todayCount
	}

	speciesToday, err := h.db.Queries.GetTotalSpeciesCountToday(ctx)
	if err == nil {
		response.SpeciesToday = speciesToday
	}

	// Get daily stats if requested
	if r.URL.Query().Get("include_daily") == "true" {
		dailyStats, err := h.db.Queries.GetDailyStats(ctx, startDate)
		if err == nil {
			response.DailyStats = make([]DailyStatResponse, 0, len(dailyStats))
			for _, s := range dailyStats {
				response.DailyStats = append(response.DailyStats, DailyStatResponse{
					Date:           s.Date,
					DetectionCount: s.DetectionCount,
					SpeciesCount:   s.SpeciesCount,
					AvgConfidence:  s.AvgConfidence,
				})
			}
		}
	}

	// Get hourly distribution if requested
	if r.URL.Query().Get("include_hourly") == "true" {
		hourlyStats, err := h.db.Queries.GetHourlyDistribution(ctx, startDate)
		if err == nil {
			response.HourlyDistribution = make([]HourlyStatResponse, 0, len(hourlyStats))
			for _, s := range hourlyStats {
				response.HourlyDistribution = append(response.HourlyDistribution, HourlyStatResponse{
					Hour:           int(s.Hour),
					DetectionCount: s.DetectionCount,
				})
			}
		}
	}

	// Get top species if requested
	if r.URL.Query().Get("include_top_species") == "true" {
		topLimit := parseIntParam(r.URL.Query().Get("top_limit"), 10)
		topSpecies, err := h.db.Queries.GetTopSpecies(ctx, db.GetTopSpeciesParams{
			StartDate: startDate,
			Limit:     int64(topLimit),
		})
		if err == nil {
			response.TopSpecies = make([]TopSpeciesResponse, 0, len(topSpecies))
			for _, s := range topSpecies {
				response.TopSpecies = append(response.TopSpecies, TopSpeciesResponse{
					SciName:        s.SciName,
					ComName:        s.ComName,
					DetectionCount: s.DetectionCount,
				})
			}
		}
	}

	writeJSON(w, http.StatusOK, response)
}
