package api

import (
	"net/http"
	"time"
)

// HeatmapResponse represents the species-hourly heatmap data.
type HeatmapResponse struct {
	Date            string     `json:"date"`
	Species         []string   `json:"species"`
	Hours           []int      `json:"hours"`
	Data            [][]int64  `json:"data"`
	TotalDetections int64      `json:"total_detections"`
}

// GetHeatmapToday handles GET /api/heatmap/today requests.
// Returns a matrix of detection counts per species per hour for today.
func (h *Handlers) GetHeatmapToday(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get today's species-hourly distribution
	rows, err := h.db.Queries.GetSpeciesHourlyDistributionToday(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch heatmap data")
		return
	}

	// Build response
	today := time.Now().Format("2006-01-02")
	hours := make([]int, 24)
	for i := 0; i < 24; i++ {
		hours[i] = i
	}

	// If no data, return empty response
	if len(rows) == 0 {
		writeJSON(w, http.StatusOK, HeatmapResponse{
			Date:            today,
			Species:         []string{},
			Hours:           hours,
			Data:            [][]int64{},
			TotalDetections: 0,
		})
		return
	}

	// Build species list and data matrix
	// First pass: collect unique species in order
	speciesMap := make(map[string]int) // species com_name -> index
	speciesList := []string{}

	for _, row := range rows {
		if _, exists := speciesMap[row.ComName]; !exists {
			speciesMap[row.ComName] = len(speciesList)
			speciesList = append(speciesList, row.ComName)
		}
	}

	// Initialize data matrix: [species][hour] = 0
	data := make([][]int64, len(speciesList))
	for i := range data {
		data[i] = make([]int64, 24)
	}

	// Fill in the data
	var totalDetections int64
	for _, row := range rows {
		speciesIdx := speciesMap[row.ComName]
		hour := int(row.Hour)
		if hour >= 0 && hour < 24 {
			data[speciesIdx][hour] = row.DetectionCount
			totalDetections += row.DetectionCount
		}
	}

	response := HeatmapResponse{
		Date:            today,
		Species:         speciesList,
		Hours:           hours,
		Data:            data,
		TotalDetections: totalDetections,
	}

	writeJSON(w, http.StatusOK, response)
}
