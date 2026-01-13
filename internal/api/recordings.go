package api

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	db "github.com/birdnet-pi/birdnet/internal/db/generated"
	"github.com/go-chi/chi/v5"
)

// =============================================================================
// Response Types
// =============================================================================

// RecordingDateResponse represents a date with recordings.
type RecordingDateResponse struct {
	Date string `json:"date"`
}

// ListRecordingDatesResponse is the response for listing recording dates.
type ListRecordingDatesResponse struct {
	Dates []string `json:"dates"`
	Total int      `json:"total"`
}

// RecordingSpeciesResponse represents a species with recording info.
type RecordingSpeciesResponse struct {
	SciName        string  `json:"sci_name"`
	ComName        string  `json:"com_name"`
	DetectionCount int64   `json:"detection_count"`
	MaxConfidence  float64 `json:"max_confidence"`
	LastSeen       string  `json:"last_seen,omitempty"`
}

// ListRecordingSpeciesResponse is the response for listing species with recordings.
type ListRecordingSpeciesResponse struct {
	Species []RecordingSpeciesResponse `json:"species"`
	Total   int                        `json:"total"`
}

// RecordingFile represents a single audio recording file.
type RecordingFile struct {
	Date         string  `json:"date"`
	Time         string  `json:"time"`
	SciName      string  `json:"sci_name"`
	ComName      string  `json:"com_name"`
	Confidence   float64 `json:"confidence"`
	FileName     string  `json:"file_name"`
	AudioURL     string  `json:"audio_url"`
	SpectroURL   string  `json:"spectrogram_url"`
	IsLocked     bool    `json:"is_locked"`
	IsShifted    bool    `json:"is_shifted"`
	ShiftedURL   string  `json:"shifted_url,omitempty"`
}

// ListRecordingFilesResponse is the response for listing recording files.
type ListRecordingFilesResponse struct {
	Files []RecordingFile `json:"files"`
	Total int             `json:"total"`
	Page  int             `json:"page"`
	Limit int             `json:"limit"`
}

// ChangeIdentificationRequest is the body for changing a detection's species.
type ChangeIdentificationRequest struct {
	NewSpecies string `json:"new_species"`
}

// ExclusionListResponse is the response for the exclusion list.
type ExclusionListResponse struct {
	Exclusions []string `json:"exclusions"`
	Total      int      `json:"total"`
}

// =============================================================================
// Handler Functions
// =============================================================================

// ListRecordingDates handles GET /api/recordings/dates requests.
// Returns a list of dates that have recordings (sorted DESC).
func (h *Handlers) ListRecordingDates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	limit := parseIntParam(r.URL.Query().Get("limit"), 365)
	if limit > 1000 {
		limit = 1000
	}

	dates, err := h.db.Queries.GetDetectionDates(ctx, db.GetDetectionDatesParams{
		Limit:  int64(limit),
		Offset: 0,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch recording dates")
		return
	}

	// Filter to only dates that have actual files on disk
	validDates := make([]string, 0, len(dates))
	extractedDir := filepath.Join(h.birdsongsDir, "Extracted", "By_Date")
	for _, date := range dates {
		datePath := filepath.Join(extractedDir, date)
		if info, err := os.Stat(datePath); err == nil && info.IsDir() {
			validDates = append(validDates, date)
		}
	}

	if validDates == nil {
		validDates = []string{}
	}

	writeJSON(w, http.StatusOK, ListRecordingDatesResponse{
		Dates: validDates,
		Total: len(validDates),
	})
}

// ListRecordingSpecies handles GET /api/recordings/species requests.
// Returns species list with sorting options.
// Query params:
//   - sort: alphabetical|occurrences|confidence|date (default: alphabetical)
//   - date: optional date filter (YYYY-MM-DD)
func (h *Handlers) ListRecordingSpecies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sort := r.URL.Query().Get("sort")
	if sort == "" {
		sort = "alphabetical"
	}
	date := r.URL.Query().Get("date")

	response := ListRecordingSpeciesResponse{
		Species: []RecordingSpeciesResponse{},
		Total:   0,
	}

	if date != "" {
		// Get species for a specific date
		species, err := h.db.Queries.ListSpeciesWithStatsByDate(ctx, date)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to fetch species")
			return
		}
		for _, s := range species {
			response.Species = append(response.Species, RecordingSpeciesResponse{
				SciName:        s.SciName,
				ComName:        s.ComName,
				DetectionCount: s.DetectionCount,
				MaxConfidence:  toFloat64(s.MaxConfidence),
				LastSeen:       s.LastSeen,
			})
		}
	} else {
		// Get all species based on sort order
		switch sort {
		case "occurrences":
			species, err := h.db.Queries.ListSpeciesWithStatsByOccurrences(ctx)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "Failed to fetch species")
				return
			}
			for _, s := range species {
				response.Species = append(response.Species, RecordingSpeciesResponse{
					SciName:        s.SciName,
					ComName:        s.ComName,
					DetectionCount: s.DetectionCount,
					MaxConfidence:  toFloat64(s.MaxConfidence),
					LastSeen:       toStringFromInterface(s.LastSeen),
				})
			}
		case "confidence":
			species, err := h.db.Queries.ListSpeciesWithStatsByConfidence(ctx)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "Failed to fetch species")
				return
			}
			for _, s := range species {
				response.Species = append(response.Species, RecordingSpeciesResponse{
					SciName:        s.SciName,
					ComName:        s.ComName,
					DetectionCount: s.DetectionCount,
					MaxConfidence:  toFloat64(s.MaxConfidence),
					LastSeen:       toStringFromInterface(s.LastSeen),
				})
			}
		case "date":
			species, err := h.db.Queries.ListSpeciesWithStatsByDate2(ctx)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "Failed to fetch species")
				return
			}
			for _, s := range species {
				response.Species = append(response.Species, RecordingSpeciesResponse{
					SciName:        s.SciName,
					ComName:        s.ComName,
					DetectionCount: s.DetectionCount,
					MaxConfidence:  toFloat64(s.MaxConfidence),
					LastSeen:       toStringFromInterface(s.LastSeen),
				})
			}
		default: // alphabetical
			species, err := h.db.Queries.ListSpeciesWithStats(ctx)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "Failed to fetch species")
				return
			}
			for _, s := range species {
				response.Species = append(response.Species, RecordingSpeciesResponse{
					SciName:        s.SciName,
					ComName:        s.ComName,
					DetectionCount: s.DetectionCount,
					MaxConfidence:  toFloat64(s.MaxConfidence),
					LastSeen:       toStringFromInterface(s.LastSeen),
				})
			}
		}
	}

	response.Total = len(response.Species)
	writeJSON(w, http.StatusOK, response)
}

// ListRecordingsByDate handles GET /api/recordings/by-date/{date} requests.
// Returns species detected on a specific date.
func (h *Handlers) ListRecordingsByDate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	date := chi.URLParam(r, "date")
	if date == "" {
		writeError(w, http.StatusBadRequest, "Date is required")
		return
	}

	species, err := h.db.Queries.ListSpeciesWithStatsByDate(ctx, date)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch species for date")
		return
	}

	// Filter to species that have files on disk
	extractedDir := filepath.Join(h.birdsongsDir, "Extracted", "By_Date", date)
	validSpecies := make([]RecordingSpeciesResponse, 0, len(species))

	for _, s := range species {
		dirName := strings.ReplaceAll(s.ComName, " ", "_")
		dirName = strings.ReplaceAll(dirName, "'", "")
		speciesPath := filepath.Join(extractedDir, dirName)
		if info, err := os.Stat(speciesPath); err == nil && info.IsDir() {
			validSpecies = append(validSpecies, RecordingSpeciesResponse{
				SciName:        s.SciName,
				ComName:        s.ComName,
				DetectionCount: s.DetectionCount,
				MaxConfidence:  toFloat64(s.MaxConfidence),
				LastSeen:       s.LastSeen,
			})
		}
	}

	writeJSON(w, http.StatusOK, ListRecordingSpeciesResponse{
		Species: validSpecies,
		Total:   len(validSpecies),
	})
}

// ListRecordingsBySpecies handles GET /api/recordings/by-species/{name} requests.
// Returns audio files for a specific species.
// Query params:
//   - date: optional date filter (YYYY-MM-DD)
//   - sort: date|confidence (default: date)
//   - only_locked: true|false - filter to only locked files
//   - limit: max files to return (default: 40)
//   - page: page number (default: 1)
func (h *Handlers) ListRecordingsBySpecies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	name := chi.URLParam(r, "name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "Species name is required")
		return
	}

	date := r.URL.Query().Get("date")
	sort := r.URL.Query().Get("sort")
	if sort == "" {
		sort = "date"
	}
	onlyLocked := r.URL.Query().Get("only_locked") == "true"
	limit := parseIntParam(r.URL.Query().Get("limit"), 40)
	page := parseIntParam(r.URL.Query().Get("page"), 1)

	if limit > 200 {
		limit = 200
	}
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit

	// Get detections from database
	var detections []db.Detection
	var err error

	if date != "" {
		// Filtered by date and species
		detections, err = h.db.Queries.ListDetectionsBySpeciesAndDate(ctx, db.ListDetectionsBySpeciesAndDateParams{
			SciName: name,
			ComName: name,
			Date:    date,
			Limit:   int64(limit + offset + 100), // Get extra for filtering
			Offset:  0,
		})
	} else {
		// All detections for species
		detections, err = h.db.Queries.ListDetectionsBySpecies(ctx, db.ListDetectionsBySpeciesParams{
			SciName: name,
			ComName: name,
			Limit:   int64(limit + offset + 100), // Get extra for filtering
			Offset:  0,
		})
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch recordings")
		return
	}

	// Load exclusion list
	exclusions := h.loadExclusionList()

	// Build response
	files := make([]RecordingFile, 0, len(detections))
	extractedDir := filepath.Join(h.birdsongsDir, "Extracted", "By_Date")
	shiftedDir := filepath.Join(extractedDir, "shifted")

	for _, d := range detections {
		comNameDir := strings.ReplaceAll(d.ComName, " ", "_")
		comNameDir = strings.ReplaceAll(comNameDir, "'", "")

		// Check if audio file exists on disk
		audioPath := filepath.Join(extractedDir, d.Date, comNameDir, d.FileName)
		if _, err := os.Stat(audioPath); os.IsNotExist(err) {
			continue // Skip if file doesn't exist
		}

		// Build file path for exclusion check
		filePath := d.Date + "/" + comNameDir + "/" + d.FileName
		isLocked := exclusions[filePath]

		if onlyLocked && !isLocked {
			continue // Skip if filtering by locked only
		}

		// Check if shifted version exists
		shiftedPath := filepath.Join(shiftedDir, d.Date, comNameDir, d.FileName)
		isShifted := fileExists(shiftedPath)

		// Build URLs
		audioURL := "/By_Date/" + d.Date + "/" + comNameDir + "/" + url.PathEscape(d.FileName)
		spectroURL := audioURL + ".png"
		shiftedURL := ""
		if isShifted {
			shiftedURL = "/By_Date/shifted/" + d.Date + "/" + comNameDir + "/" + url.PathEscape(d.FileName)
		}

		files = append(files, RecordingFile{
			Date:         d.Date,
			Time:         d.Time,
			SciName:      d.SciName,
			ComName:      d.ComName,
			Confidence:   d.Confidence.Float64,
			FileName:     d.FileName,
			AudioURL:     audioURL,
			SpectroURL:   spectroURL,
			IsLocked:     isLocked,
			IsShifted:    isShifted,
			ShiftedURL:   shiftedURL,
		})
	}

	// Apply pagination
	total := len(files)
	if offset > len(files) {
		files = []RecordingFile{}
	} else {
		end := offset + limit
		if end > len(files) {
			end = len(files)
		}
		files = files[offset:end]
	}

	writeJSON(w, http.StatusOK, ListRecordingFilesResponse{
		Files: files,
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

// DeleteRecording handles POST /api/recordings/{date}/{species}/{filename}/delete requests.
// Deletes a recording file and its database entry.
func (h *Handlers) DeleteRecording(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	date := chi.URLParam(r, "date")
	species := chi.URLParam(r, "species")
	filename := chi.URLParam(r, "filename")

	if date == "" || species == "" || filename == "" {
		writeError(w, http.StatusBadRequest, "Missing date, species, or filename")
		return
	}

	// Decode URL-encoded filename
	decodedFilename, err := url.QueryUnescape(filename)
	if err != nil {
		decodedFilename = filename
	}

	// Build file path
	comNameDir := strings.ReplaceAll(species, " ", "_")
	comNameDir = strings.ReplaceAll(comNameDir, "'", "")
	extractedDir := filepath.Join(h.birdsongsDir, "Extracted", "By_Date")
	audioPath := filepath.Join(extractedDir, date, comNameDir, decodedFilename)
	spectroPath := audioPath + ".png"

	// Delete audio file
	if err := os.Remove(audioPath); err != nil && !os.IsNotExist(err) {
		writeError(w, http.StatusInternalServerError, "Failed to delete audio file: "+err.Error())
		return
	}

	// Delete spectrogram file
	_ = os.Remove(spectroPath)

	// Delete shifted version if it exists
	shiftedDir := filepath.Join(extractedDir, "shifted")
	shiftedPath := filepath.Join(shiftedDir, date, comNameDir, decodedFilename)
	_ = os.Remove(shiftedPath)

	// Delete from database
	err = h.db.Queries.DeleteDetectionByFileName(ctx, decodedFilename)
	if err != nil {
		// Don't fail if DB delete fails - file is already deleted
		// Log the error in production
	}

	// Remove from exclusion list if present
	h.removeFromExclusionList(date + "/" + comNameDir + "/" + decodedFilename)

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ChangeRecordingIdentification handles POST /api/recordings/{date}/{species}/{filename}/change requests.
// Changes the species identification of a recording.
func (h *Handlers) ChangeRecordingIdentification(w http.ResponseWriter, r *http.Request) {
	date := chi.URLParam(r, "date")
	species := chi.URLParam(r, "species")
	filename := chi.URLParam(r, "filename")

	if date == "" || species == "" || filename == "" {
		writeError(w, http.StatusBadRequest, "Missing date, species, or filename")
		return
	}

	var req ChangeIdentificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.NewSpecies == "" {
		writeError(w, http.StatusBadRequest, "new_species is required")
		return
	}

	// Decode URL-encoded filename
	decodedFilename, err := url.QueryUnescape(filename)
	if err != nil {
		decodedFilename = filename
	}

	// Build the full filename path
	comNameDir := strings.ReplaceAll(species, " ", "_")
	comNameDir = strings.ReplaceAll(comNameDir, "'", "")
	oldFilename := date + "/" + comNameDir + "/" + decodedFilename

	// Call the birdnet_changeidentification.sh script
	scriptPath := filepath.Join(h.scriptsDir, "birdnet_changeidentification.sh")
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		// Try alternate location
		scriptPath = filepath.Join(h.scriptsDir, "tools", "birdnet_changeidentification.sh")
	}

	cmd := exec.Command(scriptPath, decodedFilename, req.NewSpecies, "log_errors")
	output, err := cmd.CombinedOutput()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to change identification: "+string(output))
		return
	}

	// Update exclusion list if needed (move entry to new species)
	h.updateExclusionListForRename(oldFilename, req.NewSpecies)

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ToggleRecordingLock handles POST /api/recordings/{date}/{species}/{filename}/lock requests.
// Toggles the purge exclusion status of a recording.
func (h *Handlers) ToggleRecordingLock(w http.ResponseWriter, r *http.Request) {
	date := chi.URLParam(r, "date")
	species := chi.URLParam(r, "species")
	filename := chi.URLParam(r, "filename")

	if date == "" || species == "" || filename == "" {
		writeError(w, http.StatusBadRequest, "Missing date, species, or filename")
		return
	}

	// Decode URL-encoded filename
	decodedFilename, err := url.QueryUnescape(filename)
	if err != nil {
		decodedFilename = filename
	}

	// Build file path
	comNameDir := strings.ReplaceAll(species, " ", "_")
	comNameDir = strings.ReplaceAll(comNameDir, "'", "")
	filePath := date + "/" + comNameDir + "/" + decodedFilename

	// Check current lock status and toggle
	exclusions := h.loadExclusionList()
	isLocked := exclusions[filePath]

	if isLocked {
		// Remove from exclusion list
		h.removeFromExclusionList(filePath)
	} else {
		// Add to exclusion list
		h.addToExclusionList(filePath)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "ok",
		"is_locked": !isLocked,
	})
}

// ToggleRecordingShift handles POST /api/recordings/{date}/{species}/{filename}/shift requests.
// Creates or removes a frequency-shifted version of the recording.
func (h *Handlers) ToggleRecordingShift(w http.ResponseWriter, r *http.Request) {
	date := chi.URLParam(r, "date")
	species := chi.URLParam(r, "species")
	filename := chi.URLParam(r, "filename")

	if date == "" || species == "" || filename == "" {
		writeError(w, http.StatusBadRequest, "Missing date, species, or filename")
		return
	}

	// Decode URL-encoded filename
	decodedFilename, err := url.QueryUnescape(filename)
	if err != nil {
		decodedFilename = filename
	}

	// Build paths
	comNameDir := strings.ReplaceAll(species, " ", "_")
	comNameDir = strings.ReplaceAll(comNameDir, "'", "")
	extractedDir := filepath.Join(h.birdsongsDir, "Extracted", "By_Date")
	sourcePath := filepath.Join(extractedDir, date, comNameDir, decodedFilename)
	shiftedDir := filepath.Join(extractedDir, "shifted", date, comNameDir)
	shiftedPath := filepath.Join(shiftedDir, decodedFilename)

	// Check if shifted version exists
	isShifted := fileExists(shiftedPath)

	if isShifted {
		// Remove shifted version
		if err := os.Remove(shiftedPath); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to remove shifted file: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":     "ok",
			"is_shifted": false,
		})
		return
	}

	// Create shifted version
	// First, ensure directory exists
	if err := os.MkdirAll(shiftedDir, 0755); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create shifted directory: "+err.Error())
		return
	}

	// Load config to get shift settings
	cfg := h.configMgr.Get()
	freqshiftTool := cfg.FreqshiftTool
	if freqshiftTool == "" {
		freqshiftTool = "sox"
	}

	var cmd *exec.Cmd
	if freqshiftTool == "ffmpeg" {
		// Use ffmpeg for frequency shifting
		lo := cfg.FreqshiftLo
		hi := cfg.FreqshiftHi
		if lo == 0 {
			lo = 1
		}
		if hi == 0 {
			hi = 2
		}
		filter := "rubberband=pitch=" + strconv.Itoa(lo) + "/" + strconv.Itoa(hi)
		cmd = exec.Command("ffmpeg", "-y", "-i", sourcePath, "-af", filter, shiftedPath)
	} else {
		// Use sox for frequency shifting
		pitch := cfg.FreqshiftPitch
		if pitch == 0 {
			pitch = -500
		}
		cmd = exec.Command("sox", sourcePath, shiftedPath, "pitch", "-q", strconv.Itoa(pitch))
	}

	if output, err := cmd.CombinedOutput(); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create shifted file: "+string(output))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":      "ok",
		"is_shifted":  true,
		"shifted_url": "/By_Date/shifted/" + date + "/" + comNameDir + "/" + url.PathEscape(decodedFilename),
	})
}

// GetExclusionList handles GET /api/recordings/exclusions requests.
// Returns the list of files excluded from purge.
func (h *Handlers) GetExclusionList(w http.ResponseWriter, r *http.Request) {
	exclusions := h.loadExclusionListSlice()

	if exclusions == nil {
		exclusions = []string{}
	}

	writeJSON(w, http.StatusOK, ExclusionListResponse{
		Exclusions: exclusions,
		Total:      len(exclusions),
	})
}

// =============================================================================
// Helper Functions
// =============================================================================

// exclusionListPath returns the path to the disk check exclusion file.
func (h *Handlers) exclusionListPath() string {
	return filepath.Join(h.scriptsDir, "disk_check_exclude.txt")
}

// loadExclusionList loads the exclusion list as a map for fast lookup.
func (h *Handlers) loadExclusionList() map[string]bool {
	exclusions := make(map[string]bool)
	path := h.exclusionListPath()

	file, err := os.Open(path)
	if err != nil {
		return exclusions
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			// Remove .png suffix if present (we store without it)
			line = strings.TrimSuffix(line, ".png")
			exclusions[line] = true
		}
	}

	return exclusions
}

// loadExclusionListSlice loads the exclusion list as a slice.
func (h *Handlers) loadExclusionListSlice() []string {
	var exclusions []string
	path := h.exclusionListPath()

	file, err := os.Open(path)
	if err != nil {
		return exclusions
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") && !strings.HasSuffix(line, ".png") {
			exclusions = append(exclusions, line)
		}
	}

	return exclusions
}

// addToExclusionList adds a file path to the exclusion list.
func (h *Handlers) addToExclusionList(filePath string) error {
	path := h.exclusionListPath()

	// Ensure file exists with headers
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.WriteFile(path, []byte("##start\n##end\n"), 0644); err != nil {
			return err
		}
	}

	// Append to file
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	// Add both the audio file and its spectrogram
	_, err = f.WriteString(filePath + "\n" + filePath + ".png\n")
	return err
}

// removeFromExclusionList removes a file path from the exclusion list.
func (h *Handlers) removeFromExclusionList(filePath string) error {
	path := h.exclusionListPath()

	// Read current content
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		// Skip lines matching the file path or its spectrogram
		if trimmedLine != filePath && trimmedLine != filePath+".png" {
			newLines = append(newLines, line)
		}
	}

	return os.WriteFile(path, []byte(strings.Join(newLines, "\n")), 0644)
}

// updateExclusionListForRename updates the exclusion list when a file is renamed.
func (h *Handlers) updateExclusionListForRename(oldPath, newSpecies string) error {
	path := h.exclusionListPath()

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Parse old path to get components
	parts := strings.Split(oldPath, "/")
	if len(parts) < 3 {
		return nil
	}
	date := parts[0]
	filename := parts[2]

	// Build new path
	newComNameDir := strings.ReplaceAll(newSpecies, " ", "_")
	newComNameDir = strings.ReplaceAll(newComNameDir, "'", "")
	// Extract common name from the "Scientific Name_Common Name" format
	if idx := strings.Index(newSpecies, "_"); idx > 0 {
		newComNameDir = strings.ReplaceAll(newSpecies[idx+1:], " ", "_")
		newComNameDir = strings.ReplaceAll(newComNameDir, "'", "")
	}
	newPath := date + "/" + newComNameDir + "/" + filename

	// Replace old path with new path
	newContent := strings.ReplaceAll(string(content), oldPath, newPath)
	newContent = strings.ReplaceAll(newContent, oldPath+".png", newPath+".png")

	return os.WriteFile(path, []byte(newContent), 0644)
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// toStringFromInterface safely converts interface{} to string.
func toStringFromInterface(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	default:
		return ""
	}
}
