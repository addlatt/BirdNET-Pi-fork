package api

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// ReclassifyRequest represents a request to change a detection's identification.
type ReclassifyRequest struct {
	FileName       string `json:"file_name"`        // Original filename (e.g., "Mésange_charbonnière-78-2024-05-02-birdnet-RTSP_1-18:14:08.mp3")
	NewIdentity    string `json:"new_identity"`     // New species in format "Sci_Name_Common Name" (e.g., "Parus major_Great Tit")
}

// ReclassifyResponse represents the response from a reclassification.
type ReclassifyResponse struct {
	Success        bool   `json:"success"`
	Message        string `json:"message"`
	OldCommonName  string `json:"old_common_name,omitempty"`
	NewCommonName  string `json:"new_common_name,omitempty"`
	NewFileName    string `json:"new_file_name,omitempty"`
}

// ReclassifyDetection handles POST /api/detections/reclassify requests.
// This replaces the birdnet_changeidentification.sh script.
func (h *Handlers) ReclassifyDetection(w http.ResponseWriter, r *http.Request) {
	var req ReclassifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Validate inputs
	if req.FileName == "" {
		writeError(w, http.StatusBadRequest, "file_name is required")
		return
	}
	if req.NewIdentity == "" {
		writeError(w, http.StatusBadRequest, "new_identity is required")
		return
	}
	if !strings.Contains(req.NewIdentity, "_") {
		writeError(w, http.StatusBadRequest, "new_identity must be in format 'Scientific Name_Common Name'")
		return
	}

	// Parse new identity
	parts := strings.SplitN(req.NewIdentity, "_", 2)
	newSciName := parts[0]
	newComName := parts[1]

	// Validate new species exists in labels
	cfg := h.configMgr.Get()
	labelsPath := filepath.Join(h.dataDir, "model", "labels.txt")
	if !h.isValidSpecies(labelsPath, req.NewIdentity) {
		writeError(w, http.StatusBadRequest, "Species not found in labels file: "+req.NewIdentity)
		return
	}

	// Get original detection info from database
	row := h.db.Conn().QueryRow(`
		SELECT Sci_Name, Com_Name, Date FROM detections WHERE File_Name = ? LIMIT 1
	`, req.FileName)

	var oldSciName, oldComName, date string
	if err := row.Scan(&oldSciName, &oldComName, &date); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "Detection not found: "+req.FileName)
			return
		}
		writeError(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	// Check if new name is the same as old
	if oldComName == newComName {
		writeJSON(w, http.StatusBadRequest, ReclassifyResponse{
			Success: false,
			Message: "New identification is the same as current identification",
		})
		return
	}

	// Sanitize names for filesystem (replace spaces with underscores, remove quotes)
	oldComNameSafe := sanitizeForFilesystem(oldComName)
	newComNameSafe := sanitizeForFilesystem(newComName)

	// Generate new filename
	newFileName := strings.Replace(req.FileName, oldComNameSafe, newComNameSafe, 1)

	// Move files
	extractedDir := cfg.Extracted
	if extractedDir == "" {
		extractedDir = filepath.Join(h.birdsongsDir, "Extracted")
	}

	oldPath := filepath.Join(extractedDir, "By_Date", date, oldComNameSafe, req.FileName)
	newDir := filepath.Join(extractedDir, "By_Date", date, newComNameSafe)
	newPath := filepath.Join(newDir, newFileName)

	// Check if source file exists
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		writeError(w, http.StatusNotFound, "Audio file not found: "+oldPath)
		return
	}

	// Create new directory if needed
	if err := os.MkdirAll(newDir, 0755); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create directory: "+err.Error())
		return
	}

	// Move audio file
	if err := os.Rename(oldPath, newPath); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to move audio file: "+err.Error())
		return
	}

	// Move spectrogram PNG if it exists
	oldPngPath := oldPath + ".png"
	newPngPath := newPath + ".png"
	if _, err := os.Stat(oldPngPath); err == nil {
		os.Rename(oldPngPath, newPngPath) // Ignore errors for PNG
	}

	// Update database
	_, err := h.db.Conn().Exec(`
		UPDATE detections
		SET Sci_Name = ?, Com_Name = ?, Confidence = 0, File_Name = ?
		WHERE File_Name = ?
	`, newSciName, newComName, newFileName, req.FileName)

	if err != nil {
		// Try to rollback file move
		os.Rename(newPath, oldPath)
		os.Rename(newPngPath, oldPngPath)
		writeError(w, http.StatusInternalServerError, "Failed to update database: "+err.Error())
		return
	}

	// Clean up old directory if empty
	oldDir := filepath.Dir(oldPath)
	entries, _ := os.ReadDir(oldDir)
	if len(entries) == 0 {
		os.Remove(oldDir)
	}

	writeJSON(w, http.StatusOK, ReclassifyResponse{
		Success:       true,
		Message:       "Detection reclassified successfully",
		OldCommonName: oldComName,
		NewCommonName: newComName,
		NewFileName:   newFileName,
	})
}

// GetModelLabels handles GET /api/labels/model requests.
// Returns all available species from the model's labels.txt file.
func (h *Handlers) GetModelLabels(w http.ResponseWriter, r *http.Request) {
	labelsPath := filepath.Join(h.dataDir, "model", "labels.txt")

	file, err := os.Open(labelsPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to open labels file: "+err.Error())
		return
	}
	defer file.Close()

	var labels []map[string]string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Format: "Scientific Name_Common Name"
		parts := strings.SplitN(line, "_", 2)
		if len(parts) == 2 {
			labels = append(labels, map[string]string{
				"scientific_name": parts[0],
				"common_name":     parts[1],
				"full":            line,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to read labels file: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"labels": labels,
		"count":  len(labels),
	})
}

// isValidSpecies checks if a species exists in the labels file.
func (h *Handlers) isValidSpecies(labelsPath, species string) bool {
	file, err := os.Open(labelsPath)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == species {
			return true
		}
	}
	return false
}

// sanitizeForFilesystem converts a species name for filesystem use.
func sanitizeForFilesystem(name string) string {
	// Replace spaces with underscores and remove quotes
	result := strings.ReplaceAll(name, " ", "_")
	result = strings.ReplaceAll(result, "'", "")
	return result
}
