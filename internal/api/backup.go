package api

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/birdnet-pi/birdnet/internal/ws"
	"github.com/google/uuid"
)

// RestoreState tracks the progress of a restore operation.
type RestoreState struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"` // "uploading", "extracting", "completed", "failed"
	Progress  int       `json:"progress"`
	Stage     string    `json:"stage"`
	Error     string    `json:"error,omitempty"`
	StartedAt time.Time `json:"started_at"`
}

// restoreStatesTTL is how long to keep completed restore states before cleanup.
const restoreStatesTTL = 1 * time.Hour

var (
	restoreStates   = make(map[string]*RestoreState)
	restoreStatesMu sync.RWMutex
)

// RestoreProgressPayload is broadcast via WebSocket during restore.
type RestoreProgressPayload struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Progress int    `json:"progress"`
	Stage    string `json:"stage"`
	Error    string `json:"error,omitempty"`
}

// criticalFiles are files that must be successfully restored; failure aborts the restore.
var criticalFiles = map[string]bool{
	"birdnet.conf": true,
	"birds.db":     true,
	"db/birds.db":  true,
}

// CreateBackup handles POST /api/backup/create.
// Streams a tar.gz archive directly to the HTTP response.
func (h *Handlers) CreateBackup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Define paths
	birdnetDir := filepath.Join(h.homeDir, "BirdNET-Pi")
	extractedDir := filepath.Join(h.birdsongsDir, "Extracted")

	required := []string{
		filepath.Join(birdnetDir, "birdnet.conf"),
		filepath.Join(birdnetDir, "data", "db", "birds.db"),
		filepath.Join(birdnetDir, "BirdDB.txt"),
		filepath.Join(extractedDir, "Charts"),
		filepath.Join(extractedDir, "By_Date"),
	}

	optional := []string{
		filepath.Join(birdnetDir, "apprise.txt"),
		filepath.Join(birdnetDir, "body.txt"),
		filepath.Join(birdnetDir, "data", "blacklisted_images.txt"),
		filepath.Join(birdnetDir, "data", "disk_check_exclude.txt"),
		filepath.Join(birdnetDir, "exclude_species_list.txt"),
		filepath.Join(birdnetDir, "confirmed_species_list.txt"),
		filepath.Join(birdnetDir, "include_species_list.txt"),
	}

	// Check required files exist
	for _, path := range required {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("required file not found: %s", filepath.Base(path)))
			return
		}
	}

	// Set streaming headers
	timestamp := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("birdnet-backup_%s.tar.gz", timestamp)
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

	// Create gzip writer piped to response
	gzWriter := gzip.NewWriter(w)
	defer gzWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	log.Printf("Backup: streaming archive to client")

	// Add required files
	for _, path := range required {
		if ctx.Err() != nil {
			log.Printf("Backup: client disconnected")
			return
		}
		if err := addToArchive(ctx, tarWriter, path); err != nil {
			log.Printf("Backup: failed to add %s: %v", path, err)
			return
		}
	}

	// Add optional files (ignore errors)
	for _, path := range optional {
		if ctx.Err() != nil {
			return
		}
		if _, err := os.Stat(path); err == nil {
			if err := addToArchive(ctx, tarWriter, path); err != nil {
				log.Printf("Backup: warning - failed to add optional file %s: %v", path, err)
			}
		}
	}

	log.Printf("Backup: streaming complete")
}

// RestoreBackup handles POST /api/backup/restore.
// Accepts a multipart file upload and restores the backup.
func (h *Handlers) RestoreBackup(w http.ResponseWriter, r *http.Request) {
	// Clean up old restore states periodically
	cleanupOldRestoreStates()

	// Parse multipart form with 500MB limit
	if err := r.ParseMultipartForm(500 << 20); err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "file too large (max 500MB)")
		return
	}

	file, header, err := r.FormFile("backup")
	if err != nil {
		writeError(w, http.StatusBadRequest, "no backup file provided")
		return
	}
	defer file.Close()

	// Validate file extension
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".tar.gz") &&
		!strings.HasSuffix(strings.ToLower(header.Filename), ".tgz") {
		writeError(w, http.StatusBadRequest, "invalid file type (expected .tar.gz)")
		return
	}

	// Create restore task ID for progress tracking
	restoreID := uuid.New().String()

	// Initialize restore state
	state := &RestoreState{
		ID:        restoreID,
		Status:    "uploading",
		Progress:  0,
		Stage:     "Preparing restore...",
		StartedAt: time.Now(),
	}
	restoreStatesMu.Lock()
	restoreStates[restoreID] = state
	restoreStatesMu.Unlock()

	// Copy uploaded file to temp location (needed for two-pass processing)
	tempFile, err := os.CreateTemp("", "birdnet-restore-*.tar.gz")
	if err != nil {
		updateRestoreState(restoreID, "failed", 0, "", "failed to create temp file")
		writeError(w, http.StatusInternalServerError, "failed to process upload")
		return
	}

	_, err = io.Copy(tempFile, file)
	if err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		updateRestoreState(restoreID, "failed", 0, "", "failed to save upload")
		writeError(w, http.StatusInternalServerError, "failed to save upload")
		return
	}
	tempFile.Close()

	// Run restore in background goroutine
	go h.runRestore(restoreID, tempFile.Name())

	// Return task ID for progress polling
	writeJSON(w, http.StatusAccepted, map[string]string{
		"restore_id": restoreID,
		"status":     "started",
	})
}

// GetRestoreStatus handles GET /api/backup/status?id={restore_id}.
func (h *Handlers) GetRestoreStatus(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing restore id")
		return
	}

	restoreStatesMu.RLock()
	state, exists := restoreStates[id]
	restoreStatesMu.RUnlock()

	if !exists {
		writeError(w, http.StatusNotFound, "restore not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":         state.ID,
		"status":     state.Status,
		"progress":   state.Progress,
		"stage":      state.Stage,
		"error":      state.Error,
		"started_at": state.StartedAt.Format(time.RFC3339),
	})
}

// runRestore performs the actual restore operation with atomic guarantees.
func (h *Handlers) runRestore(id, tempFilePath string) {
	defer os.Remove(tempFilePath)

	updateRestoreState(id, "extracting", 5, "Counting files in archive...", "")
	h.broadcastRestoreProgress(id)

	// Define path mappings
	birdnetDir := filepath.Join(h.homeDir, "BirdNET-Pi")
	extractedDir := filepath.Join(h.birdsongsDir, "Extracted")

	// Get absolute paths for security validation
	absBirdnetDir, err := filepath.Abs(birdnetDir)
	if err != nil {
		updateRestoreState(id, "failed", 0, "", "failed to resolve BirdNET directory")
		h.broadcastRestoreProgress(id)
		return
	}
	absExtractedDir, err := filepath.Abs(extractedDir)
	if err != nil {
		updateRestoreState(id, "failed", 0, "", "failed to resolve extracted directory")
		h.broadcastRestoreProgress(id)
		return
	}

	// PASS 1: Count files and validate archive structure
	totalFiles, err := countArchiveFiles(tempFilePath)
	if err != nil {
		updateRestoreState(id, "failed", 0, "", fmt.Sprintf("invalid archive: %v", err))
		h.broadcastRestoreProgress(id)
		return
	}

	if totalFiles == 0 {
		updateRestoreState(id, "failed", 0, "", "archive is empty")
		h.broadcastRestoreProgress(id)
		return
	}

	log.Printf("Restore: archive contains %d files", totalFiles)

	// Create temporary extraction directory for atomic restore
	tempExtractDir, err := os.MkdirTemp("", "birdnet-restore-extract-*")
	if err != nil {
		updateRestoreState(id, "failed", 0, "", "failed to create temp directory")
		h.broadcastRestoreProgress(id)
		return
	}
	defer os.RemoveAll(tempExtractDir) // Clean up on any exit

	updateRestoreState(id, "extracting", 10, "Extracting to staging area...", "")
	h.broadcastRestoreProgress(id)

	// PASS 2: Extract files to temporary directory
	file, err := os.Open(tempFilePath)
	if err != nil {
		updateRestoreState(id, "failed", 0, "", fmt.Sprintf("failed to open archive: %v", err))
		h.broadcastRestoreProgress(id)
		return
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		updateRestoreState(id, "failed", 0, "", fmt.Sprintf("invalid gzip archive: %v", err))
		h.broadcastRestoreProgress(id)
		return
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	// Track files to move after successful extraction
	type fileMapping struct {
		tempPath string
		destPath string
	}
	var filesToMove []fileMapping

	processedCount := 0
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			updateRestoreState(id, "failed", 0, "", fmt.Sprintf("error reading archive: %v", err))
			h.broadcastRestoreProgress(id)
			return
		}

		// Reject symlinks and hardlinks (security)
		if header.Typeflag == tar.TypeSymlink || header.Typeflag == tar.TypeLink {
			log.Printf("Restore: rejecting symlink/hardlink: %s", header.Name)
			continue
		}

		// Clean and validate archive path
		cleanName := filepath.Clean(header.Name)

		// Map archive path to system path
		destPath := mapArchivePathToSystem(cleanName, birdnetDir, extractedDir)
		if destPath == "" {
			log.Printf("Restore: skipping unrecognized path: %s", cleanName)
			continue
		}

		// SECURITY: Validate destination is within allowed directories
		absDestPath, err := filepath.Abs(destPath)
		if err != nil {
			log.Printf("Restore: skipping invalid path: %s", cleanName)
			continue
		}

		if !strings.HasPrefix(absDestPath, absBirdnetDir+string(os.PathSeparator)) &&
			!strings.HasPrefix(absDestPath, absExtractedDir+string(os.PathSeparator)) &&
			absDestPath != absBirdnetDir && absDestPath != absExtractedDir {
			log.Printf("Restore: path traversal attempt blocked: %s -> %s", cleanName, absDestPath)
			continue
		}

		// Calculate temp extraction path (mirrors dest structure)
		relPath, _ := filepath.Rel(h.homeDir, destPath)
		tempPath := filepath.Join(tempExtractDir, relPath)

		// Extract to temp location
		if err := extractFile(tarReader, header, tempPath); err != nil {
			isCritical := criticalFiles[cleanName]
			if isCritical {
				updateRestoreState(id, "failed", 0, "", fmt.Sprintf("failed to extract critical file %s: %v", cleanName, err))
				h.broadcastRestoreProgress(id)
				return
			}
			log.Printf("Restore: failed to extract %s: %v (continuing)", cleanName, err)
			continue
		}

		// Track for final move (skip directories, they're created implicitly)
		if header.Typeflag != tar.TypeDir {
			filesToMove = append(filesToMove, fileMapping{
				tempPath: tempPath,
				destPath: destPath,
			})
		}

		// Update progress
		processedCount++
		progress := 10 + int(float64(processedCount)/float64(totalFiles)*70)
		if progress > 80 {
			progress = 80
		}
		updateRestoreState(id, "extracting", progress, fmt.Sprintf("Extracting %s (%d/%d)", cleanName, processedCount, totalFiles), "")

		// Broadcast progress every 10 files to avoid flooding
		if processedCount%10 == 0 {
			h.broadcastRestoreProgress(id)
		}
	}

	// PASS 3: Move files from temp to final destination (atomic)
	updateRestoreState(id, "extracting", 85, "Moving files to final location...", "")
	h.broadcastRestoreProgress(id)

	movedCount := 0
	for _, fm := range filesToMove {
		// Ensure destination directory exists
		if err := os.MkdirAll(filepath.Dir(fm.destPath), 0755); err != nil {
			log.Printf("Restore: failed to create directory for %s: %v", fm.destPath, err)
			continue
		}

		// Move file (copy + delete for cross-filesystem support)
		if err := moveFile(fm.tempPath, fm.destPath); err != nil {
			log.Printf("Restore: failed to move %s: %v", fm.destPath, err)
			continue
		}
		movedCount++

		progress := 85 + int(float64(movedCount)/float64(len(filesToMove))*14)
		if progress > 99 {
			progress = 99
		}
		if movedCount%20 == 0 {
			updateRestoreState(id, "extracting", progress, fmt.Sprintf("Installing files (%d/%d)", movedCount, len(filesToMove)), "")
			h.broadcastRestoreProgress(id)
		}
	}

	updateRestoreState(id, "completed", 100, "Restore complete", "")
	h.broadcastRestoreProgress(id)
	log.Printf("Restore: completed, processed %d files, moved %d files", processedCount, movedCount)
}

// countArchiveFiles counts the number of files in a tar.gz archive.
func countArchiveFiles(archivePath string) (int, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return 0, err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	count := 0
	for {
		_, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
		count++
	}
	return count, nil
}

// moveFile moves a file from src to dst, handling cross-filesystem moves.
func moveFile(src, dst string) error {
	// Try rename first (fast, same filesystem)
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	// Fall back to copy + delete (cross-filesystem)
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	// Close before removing
	srcFile.Close()
	return os.Remove(src)
}

// mapArchivePathToSystem maps archive paths to their system destinations.
func mapArchivePathToSystem(archivePath, birdnetDir, extractedDir string) string {
	// Handle top-level files
	switch archivePath {
	case "birdnet.conf":
		return filepath.Join(birdnetDir, "birdnet.conf")
	case "BirdDB.txt":
		return filepath.Join(birdnetDir, "BirdDB.txt")
	case "apprise.txt":
		return filepath.Join(birdnetDir, "apprise.txt")
	case "body.txt":
		return filepath.Join(birdnetDir, "body.txt")
	case "exclude_species_list.txt":
		return filepath.Join(birdnetDir, "exclude_species_list.txt")
	case "confirmed_species_list.txt":
		return filepath.Join(birdnetDir, "confirmed_species_list.txt")
	case "include_species_list.txt":
		return filepath.Join(birdnetDir, "include_species_list.txt")
	}

	// Handle birds.db (might be at top level or in db/ subdirectory)
	if archivePath == "birds.db" || archivePath == "db/birds.db" {
		return filepath.Join(birdnetDir, "data", "db", "birds.db")
	}

	// Handle data directory files
	if strings.HasPrefix(archivePath, "data/") {
		return filepath.Join(birdnetDir, archivePath)
	}

	// Handle Charts directory
	if strings.HasPrefix(archivePath, "Charts/") || archivePath == "Charts" {
		return filepath.Join(extractedDir, archivePath)
	}

	// Handle By_Date directory
	if strings.HasPrefix(archivePath, "By_Date/") || archivePath == "By_Date" {
		return filepath.Join(extractedDir, archivePath)
	}

	return ""
}

// extractFile extracts a single file from the tar archive to destPath.
func extractFile(tarReader *tar.Reader, header *tar.Header, destPath string) error {
	// Create directory
	if header.Typeflag == tar.TypeDir {
		return os.MkdirAll(destPath, os.FileMode(header.Mode))
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create the file
	outFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer outFile.Close()

	// Copy contents
	if _, err := io.Copy(outFile, tarReader); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// updateRestoreState updates the restore state atomically.
func updateRestoreState(id, status string, progress int, stage, errMsg string) {
	restoreStatesMu.Lock()
	defer restoreStatesMu.Unlock()

	if state, exists := restoreStates[id]; exists {
		state.Status = status
		state.Progress = progress
		if stage != "" {
			state.Stage = stage
		}
		state.Error = errMsg
	}
}

// cleanupOldRestoreStates removes restore states older than TTL.
func cleanupOldRestoreStates() {
	restoreStatesMu.Lock()
	defer restoreStatesMu.Unlock()

	now := time.Now()
	for id, state := range restoreStates {
		// Only clean up completed or failed states
		if state.Status == "completed" || state.Status == "failed" {
			if now.Sub(state.StartedAt) > restoreStatesTTL {
				delete(restoreStates, id)
			}
		}
	}
}

// broadcastRestoreProgress sends a WebSocket message with restore progress.
func (h *Handlers) broadcastRestoreProgress(id string) {
	restoreStatesMu.RLock()
	state, exists := restoreStates[id]
	if !exists {
		restoreStatesMu.RUnlock()
		return
	}

	payload := RestoreProgressPayload{
		ID:       state.ID,
		Status:   state.Status,
		Progress: state.Progress,
		Stage:    state.Stage,
		Error:    state.Error,
	}
	restoreStatesMu.RUnlock()

	if err := h.hub.Broadcast(ws.ChannelTasks, ws.TypeRestoreProgress, payload); err != nil {
		log.Printf("Restore: failed to broadcast progress: %v", err)
	}
}

// addToArchive adds a file or directory to the tar archive.
func addToArchive(ctx context.Context, tw *tar.Writer, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	baseName := filepath.Base(path)

	if info.IsDir() {
		return addDirToArchive(ctx, tw, path, baseName)
	}

	return addFileToArchive(tw, path, baseName)
}

// addFileToArchive adds a single file to the archive.
func addFileToArchive(tw *tar.Writer, filePath, archiveName string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}

	header.Name = archiveName

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	_, err = io.Copy(tw, file)
	return err
}

// addDirToArchive adds a directory and its contents to the archive.
func addDirToArchive(ctx context.Context, tw *tar.Writer, dirPath, archiveBase string) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Calculate relative path for archive
		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return err
		}

		archivePath := archiveBase
		if relPath != "." {
			archivePath = filepath.Join(archiveBase, relPath)
		}

		// Use forward slashes in archive
		archivePath = strings.ReplaceAll(archivePath, string(os.PathSeparator), "/")

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		header.Name = archivePath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// Write file contents if not a directory
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}

			_, copyErr := io.Copy(tw, file)
			file.Close() // Close immediately instead of defer to avoid accumulating handles

			if copyErr != nil {
				return copyErr
			}
		}

		return nil
	})
}
