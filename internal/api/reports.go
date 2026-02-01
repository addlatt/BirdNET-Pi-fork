package api

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// WeeklyReportResponse represents the weekly report data.
type WeeklyReportResponse struct {
	Week            string                `json:"week"`
	StartDate       string                `json:"start_date"`
	EndDate         string                `json:"end_date"`
	TotalDetections int64                 `json:"total_detections"`
	UniqueSpecies   int                   `json:"unique_species"`
	Comparison      WeekComparison        `json:"comparison"`
	TopSpecies      []SpeciesWeekSummary  `json:"top_species"`
	NewSpecies      []NewSpeciesSummary   `json:"new_species"`
}

// WeekComparison shows how this week compares to the previous week.
type WeekComparison struct {
	PrevTotal  int64   `json:"prev_total"`
	ChangePct  float64 `json:"change_pct"`
}

// SpeciesWeekSummary represents a species summary for the week.
type SpeciesWeekSummary struct {
	SciName   string  `json:"sci_name"`
	ComName   string  `json:"com_name"`
	Count     int64   `json:"count"`
	ChangePct float64 `json:"change_pct"`
}

// NewSpeciesSummary represents a newly detected species.
type NewSpeciesSummary struct {
	SciName       string `json:"sci_name"`
	ComName       string `json:"com_name"`
	Count         int64  `json:"count"`
	FirstDetected string `json:"first_detected"`
}

// parseWeekString parses a week string like "2024-W05" and returns start/end dates.
// Week starts on Sunday and ends on Saturday.
func parseWeekString(weekStr string) (time.Time, time.Time, error) {
	parts := strings.Split(weekStr, "-W")
	if len(parts) != 2 {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid week format: %s (expected YYYY-Www)", weekStr)
	}

	year, err := strconv.Atoi(parts[0])
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid year in week string: %s", weekStr)
	}

	weekNum, err := strconv.Atoi(parts[1])
	if err != nil || weekNum < 1 || weekNum > 53 {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid week number in week string: %s", weekStr)
	}

	// Find the first Sunday of the year
	jan1 := time.Date(year, time.January, 1, 0, 0, 0, 0, time.Local)
	daysTillSunday := (7 - int(jan1.Weekday())) % 7
	firstSunday := jan1.AddDate(0, 0, daysTillSunday)

	// If Jan 1 is a Sunday, that's week 1
	if jan1.Weekday() == time.Sunday {
		firstSunday = jan1
	}

	// Calculate the start of the requested week
	weekStart := firstSunday.AddDate(0, 0, (weekNum-1)*7)
	weekEnd := weekStart.AddDate(0, 0, 6)

	return weekStart, weekEnd, nil
}

// getCurrentWeek returns the current week string in ISO format (YYYY-Www).
func getCurrentWeek() string {
	now := time.Now()
	year, week := now.ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, week)
}

// getWeekBoundaries returns the start (Sunday) and end (Saturday) dates for the current week.
func getWeekBoundaries(t time.Time) (time.Time, time.Time) {
	// Find the most recent Sunday
	daysUntilSunday := int(t.Weekday())
	weekStart := t.AddDate(0, 0, -daysUntilSunday)
	weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, time.Local)
	weekEnd := weekStart.AddDate(0, 0, 6)
	return weekStart, weekEnd
}

// WeeklyReport handles GET /api/reports/weekly requests.
func (h *Handlers) WeeklyReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get week parameter or use current week
	weekStr := r.URL.Query().Get("week")
	var weekStart, weekEnd time.Time
	var err error

	if weekStr == "" {
		weekStart, weekEnd = getWeekBoundaries(time.Now())
		year, week := time.Now().ISOWeek()
		weekStr = fmt.Sprintf("%d-W%02d", year, week)
	} else {
		weekStart, weekEnd, err = parseWeekString(weekStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	startDate := weekStart.Format("2006-01-02")
	endDate := weekEnd.Format("2006-01-02")

	// Query detections for this week grouped by species
	thisWeekSpecies, totalDetections, err := h.getWeekSpeciesStats(ctx, startDate, endDate)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get weekly stats: "+err.Error())
		return
	}

	// Query previous week for comparison
	prevWeekStart := weekStart.AddDate(0, 0, -7)
	prevWeekEnd := weekEnd.AddDate(0, 0, -7)
	prevStartDate := prevWeekStart.Format("2006-01-02")
	prevEndDate := prevWeekEnd.Format("2006-01-02")

	prevWeekSpecies, prevTotalDetections, err := h.getWeekSpeciesStats(ctx, prevStartDate, prevEndDate)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get previous week stats: "+err.Error())
		return
	}

	// Calculate change percentage
	var changePct float64
	if prevTotalDetections > 0 {
		changePct = float64(totalDetections-prevTotalDetections) / float64(prevTotalDetections) * 100
	}

	// Build top species list with change percentages
	topSpecies := make([]SpeciesWeekSummary, 0, len(thisWeekSpecies))
	for sciName, data := range thisWeekSpecies {
		var speciesChangePct float64
		if prev, ok := prevWeekSpecies[sciName]; ok && prev.count > 0 {
			speciesChangePct = float64(data.count-prev.count) / float64(prev.count) * 100
		} else if data.count > 0 {
			speciesChangePct = 100 // New species this week
		}
		topSpecies = append(topSpecies, SpeciesWeekSummary{
			SciName:   sciName,
			ComName:   data.comName,
			Count:     data.count,
			ChangePct: speciesChangePct,
		})
	}

	// Sort by count descending (simple bubble sort for small lists)
	for i := 0; i < len(topSpecies); i++ {
		for j := i + 1; j < len(topSpecies); j++ {
			if topSpecies[j].Count > topSpecies[i].Count {
				topSpecies[i], topSpecies[j] = topSpecies[j], topSpecies[i]
			}
		}
	}

	// Limit to top 20
	if len(topSpecies) > 20 {
		topSpecies = topSpecies[:20]
	}

	// Find new species (detected this week but not before)
	newSpecies, err := h.getNewSpeciesInRange(ctx, startDate, endDate)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get new species: "+err.Error())
		return
	}

	response := WeeklyReportResponse{
		Week:            weekStr,
		StartDate:       startDate,
		EndDate:         endDate,
		TotalDetections: totalDetections,
		UniqueSpecies:   len(thisWeekSpecies),
		Comparison: WeekComparison{
			PrevTotal: prevTotalDetections,
			ChangePct: changePct,
		},
		TopSpecies: topSpecies,
		NewSpecies: newSpecies,
	}

	writeJSON(w, http.StatusOK, response)
}

type speciesData struct {
	comName string
	count   int64
}

// getWeekSpeciesStats queries detections in a date range and returns species stats.
func (h *Handlers) getWeekSpeciesStats(ctx context.Context, startDate, endDate string) (map[string]speciesData, int64, error) {
	query := `
		SELECT sci_name, com_name, COUNT(*) as count
		FROM detections
		WHERE date >= ? AND date <= ?
		GROUP BY sci_name, com_name
		ORDER BY count DESC
	`
	rows, err := h.db.Conn().QueryContext(ctx, query, startDate, endDate)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	species := make(map[string]speciesData)
	var total int64

	for rows.Next() {
		var sciName, comName string
		var count int64
		if err := rows.Scan(&sciName, &comName, &count); err != nil {
			return nil, 0, err
		}
		species[sciName] = speciesData{comName: comName, count: count}
		total += count
	}

	return species, total, rows.Err()
}

// getNewSpeciesInRange returns species detected in the date range that were never detected before.
func (h *Handlers) getNewSpeciesInRange(ctx context.Context, startDate, endDate string) ([]NewSpeciesSummary, error) {
	query := `
		SELECT d.sci_name, d.com_name, MIN(d.date || ' ' || d.time) as first_detected, COUNT(*) as count
		FROM detections d
		WHERE d.date >= ? AND d.date <= ?
		  AND d.sci_name NOT IN (
			SELECT DISTINCT sci_name FROM detections WHERE date < ?
		  )
		GROUP BY d.sci_name, d.com_name
		ORDER BY first_detected ASC
	`
	rows, err := h.db.Conn().QueryContext(ctx, query, startDate, endDate, startDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var newSpecies []NewSpeciesSummary
	for rows.Next() {
		var sciName, comName, firstDetected string
		var count int64
		if err := rows.Scan(&sciName, &comName, &firstDetected, &count); err != nil {
			return nil, err
		}
		newSpecies = append(newSpecies, NewSpeciesSummary{
			SciName:       sciName,
			ComName:       comName,
			Count:         count,
			FirstDetected: firstDetected,
		})
	}

	return newSpecies, rows.Err()
}

// ExportWeeklyReport handles GET /api/reports/weekly/export requests.
func (h *Handlers) ExportWeeklyReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get week parameter or use current week
	weekStr := r.URL.Query().Get("week")
	var weekStart, weekEnd time.Time
	var err error

	if weekStr == "" {
		weekStart, weekEnd = getWeekBoundaries(time.Now())
		year, week := time.Now().ISOWeek()
		weekStr = fmt.Sprintf("%d-W%02d", year, week)
	} else {
		weekStart, weekEnd, err = parseWeekString(weekStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	startDate := weekStart.Format("2006-01-02")
	endDate := weekEnd.Format("2006-01-02")

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}

	// Get species stats for the week
	query := `
		SELECT sci_name, com_name, COUNT(*) as count, MAX(confidence) as max_confidence
		FROM detections
		WHERE date >= ? AND date <= ?
		GROUP BY sci_name, com_name
		ORDER BY count DESC
	`
	rows, err := h.db.Conn().QueryContext(ctx, query, startDate, endDate)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query detections: "+err.Error())
		return
	}
	defer rows.Close()

	type exportRow struct {
		SciName       string
		ComName       string
		Count         int64
		MaxConfidence sql.NullFloat64
	}

	var data []exportRow
	for rows.Next() {
		var row exportRow
		if err := rows.Scan(&row.SciName, &row.ComName, &row.Count, &row.MaxConfidence); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan row: "+err.Error())
			return
		}
		data = append(data, row)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to iterate rows: "+err.Error())
		return
	}

	// Load eBird codes if exporting in eBird format
	var ebirdCodes map[string]string
	if format == "ebird" {
		ebirdCodes, err = loadEbirdCodes(h.dataDir)
		if err != nil {
			// Non-fatal: just won't have eBird codes
			ebirdCodes = make(map[string]string)
		}
	}

	// Set response headers for CSV download
	filename := fmt.Sprintf("birdnet-weekly-%s.csv", weekStr)
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	writer := csv.NewWriter(w)

	// Write header
	if format == "ebird" {
		writer.Write([]string{"Species Code", "Scientific Name", "Common Name", "Count", "Max Confidence"})
	} else {
		writer.Write([]string{"Scientific Name", "Common Name", "Count", "Max Confidence"})
	}

	// Write data rows
	for _, row := range data {
		confidence := ""
		if row.MaxConfidence.Valid {
			confidence = fmt.Sprintf("%.2f", row.MaxConfidence.Float64)
		}

		if format == "ebird" {
			code := ""
			if ebirdCodes != nil {
				code = ebirdCodes[row.SciName]
			}
			writer.Write([]string{code, row.SciName, row.ComName, strconv.FormatInt(row.Count, 10), confidence})
		} else {
			writer.Write([]string{row.SciName, row.ComName, strconv.FormatInt(row.Count, 10), confidence})
		}
	}

	writer.Flush()
}

// ebirdCodes is a cache for the eBird species codes.
var ebirdCodesCache map[string]string

// loadEbirdCodes loads the eBird species codes from the JSON file.
func loadEbirdCodes(dataDir string) (map[string]string, error) {
	if ebirdCodesCache != nil {
		return ebirdCodesCache, nil
	}

	jsonPath := filepath.Join(dataDir, "ebird.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read ebird.json: %w", err)
	}

	codes := make(map[string]string)
	if err := json.Unmarshal(data, &codes); err != nil {
		return nil, fmt.Errorf("failed to parse ebird.json: %w", err)
	}

	ebirdCodesCache = codes
	return codes, nil
}

// GetEbirdCode returns the eBird species code for a given scientific name.
func GetEbirdCode(dataDir, sciName string) string {
	codes, err := loadEbirdCodes(dataDir)
	if err != nil {
		return ""
	}
	if code, ok := codes[sciName]; ok {
		return code
	}
	return ""
}
