package tasks

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/birdnet-pi/birdnet/internal/config"
)

// SpeciesCleanupTask maintains a maximum number of files per species.
// This replaces the disk_species_clean.sh cron script.
type SpeciesCleanupTask struct {
	configMgr    *config.Manager
	db           *sql.DB
	extractedDir string
	dataDir      string
}

// NewSpeciesCleanupTask creates a new species cleanup task.
func NewSpeciesCleanupTask(configMgr *config.Manager, db *sql.DB, extractedDir, dataDir string) *SpeciesCleanupTask {
	return &SpeciesCleanupTask{
		configMgr:    configMgr,
		db:           db,
		extractedDir: extractedDir,
		dataDir:      dataDir,
	}
}

func (t *SpeciesCleanupTask) Name() string {
	return "species_cleanup"
}

func (t *SpeciesCleanupTask) Description() string {
	return "Maintains maximum files per species by removing old, low-confidence recordings"
}

func (t *SpeciesCleanupTask) DefaultSchedule() string {
	return "0 2 * * *" // Daily at 2 AM
}

func (t *SpeciesCleanupTask) Timeout() time.Duration {
	return 2 * time.Hour
}

func (t *SpeciesCleanupTask) Run(ctx context.Context) error {
	cfg := t.configMgr.Get()

	maxFilesPerSpecies := cfg.MaxFilesSpecies
	if maxFilesPerSpecies < 1 {
		log.Printf("Species cleanup: max_files_species is %d, skipping", maxFilesPerSpecies)
		return nil
	}

	// Get list of species from database
	species, err := t.getSpeciesFromDB()
	if err != nil {
		return fmt.Errorf("failed to get species from database: %w", err)
	}

	if len(species) == 0 {
		log.Printf("Species cleanup: no species found")
		return nil
	}

	log.Printf("Species cleanup: processing %d species with max %d files each",
		len(species), maxFilesPerSpecies)

	// Load exclude list
	excludeList := t.loadExcludeList()

	// Calculate dates to protect (last 7 days)
	protectedDates := t.getProtectedDates()

	byDateDir := filepath.Join(t.extractedDir, "By_Date")
	totalDeleted := 0

	for _, speciesName := range species {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		deleted, remaining, err := t.cleanupSpecies(ctx, byDateDir, speciesName, maxFilesPerSpecies, excludeList, protectedDates)
		if err != nil {
			log.Printf("Species cleanup: %s failed: %v", speciesName, err)
			continue
		}

		if deleted > 0 {
			log.Printf("Species cleanup: %s - deleted %d files, %d remaining", speciesName, deleted, remaining)
			totalDeleted += deleted
		}
	}

	log.Printf("Species cleanup: completed, deleted %d files total", totalDeleted)
	return nil
}

// getSpeciesFromDB returns the list of unique species common names from the database.
func (t *SpeciesCleanupTask) getSpeciesFromDB() ([]string, error) {
	query := "SELECT DISTINCT Com_Name FROM detections"
	rows, err := t.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var species []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		species = append(species, name)
	}

	return species, rows.Err()
}

// getProtectedDates returns a set of dates to protect (last 7 days).
func (t *SpeciesCleanupTask) getProtectedDates() map[string]bool {
	protected := make(map[string]bool)
	now := time.Now()

	for i := 0; i < 8; i++ {
		date := now.AddDate(0, 0, -i).Format("2006-01-02")
		protected[date] = true
	}

	return protected
}

// cleanupSpecies removes excess files for a single species.
func (t *SpeciesCleanupTask) cleanupSpecies(
	ctx context.Context,
	byDateDir string,
	speciesName string,
	maxFiles int,
	excludeList map[string]bool,
	protectedDates map[string]bool,
) (deleted, remaining int, err error) {
	// Sanitize species name for filesystem (replace spaces with underscores, remove quotes)
	sanitizedName := strings.ReplaceAll(speciesName, " ", "_")
	sanitizedName = strings.ReplaceAll(sanitizedName, "'", "")

	// Find all files for this species across all dates
	files, err := t.findSpeciesFiles(byDateDir, sanitizedName)
	if err != nil {
		return 0, 0, err
	}

	if len(files) <= maxFiles {
		return 0, len(files), nil
	}

	// Filter out protected files (recent dates and exclude list)
	var deletable []fileInfo
	var protectedCount int

	for _, f := range files {
		relPath := strings.TrimPrefix(f.path, byDateDir+"/")

		// Check exclude list
		if excludeList[relPath] {
			protectedCount++
			continue
		}

		// Check if file is from protected dates
		if t.isProtectedDate(f.path, protectedDates) {
			protectedCount++
			continue
		}

		deletable = append(deletable, f)
	}

	// Sort by confidence (descending) then date (oldest first)
	sort.Slice(deletable, func(i, j int) bool {
		// Higher confidence comes first (keep)
		if deletable[i].confidence != deletable[j].confidence {
			return deletable[i].confidence > deletable[j].confidence
		}
		// Newer files come first (keep)
		return deletable[i].date > deletable[j].date
	})

	// Calculate how many to keep from deletable list
	keepFromDeletable := maxFiles - protectedCount
	if keepFromDeletable < 0 {
		keepFromDeletable = 0
	}

	// Delete files beyond the keep count
	toDelete := deletable[keepFromDeletable:]

	for _, f := range toDelete {
		if ctx.Err() != nil {
			return deleted, len(files) - deleted, ctx.Err()
		}

		// Delete audio file
		if err := os.Remove(f.path); err != nil {
			log.Printf("Species cleanup: failed to remove %s: %v", f.path, err)
			continue
		}

		// Also delete associated PNG if it exists
		pngPath := f.path + ".png"
		if _, err := os.Stat(pngPath); err == nil {
			os.Remove(pngPath)
		}

		deleted++
	}

	return deleted, len(files) - deleted, nil
}

// fileInfo holds information about a recording file.
type fileInfo struct {
	path       string
	date       string
	confidence float64
}

// findSpeciesFiles finds all audio files for a species.
func (t *SpeciesCleanupTask) findSpeciesFiles(byDateDir, speciesName string) ([]fileInfo, error) {
	var files []fileInfo

	// Pattern: YYYY-MM-DD/Species_Name/*.mp3 (or other audio formats)
	// Filename format: Com_Name-Conf-YYYY-MM-DD-birdnet-HH:MM:SS.ext

	err := filepath.Walk(byDateDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// Check if this file belongs to the species
		dir := filepath.Dir(path)
		dirName := filepath.Base(dir)

		if dirName != speciesName {
			return nil
		}

		// Only process audio files
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".mp3" && ext != ".wav" && ext != ".flac" && ext != ".ogg" && ext != ".opus" {
			return nil
		}

		// Parse file info
		fi := t.parseFileInfo(path)
		files = append(files, fi)

		return nil
	})

	return files, err
}

// parseFileInfo extracts date and confidence from a filename.
func (t *SpeciesCleanupTask) parseFileInfo(path string) fileInfo {
	fi := fileInfo{path: path}

	filename := filepath.Base(path)

	// Extract date from filename (YYYY-MM-DD pattern)
	datePattern := regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
	if matches := datePattern.FindStringSubmatch(filename); len(matches) > 1 {
		fi.date = matches[1]
	}

	// Extract confidence from filename
	// Format: Species_Name-Confidence-Date-...
	// The confidence is typically the second field when split by "-"
	parts := strings.Split(filename, "-")
	for i, part := range parts {
		// Look for a number that could be confidence (typically 2-4 digits)
		if conf, err := strconv.ParseFloat(part, 64); err == nil && conf >= 0 && conf <= 100 {
			// Confidence values in filenames are often integers 0-100
			fi.confidence = conf
			break
		}
		// Also check for parts after species name
		if i > 0 {
			if conf, err := strconv.ParseFloat(part, 64); err == nil {
				fi.confidence = conf
				break
			}
		}
	}

	return fi
}

// isProtectedDate checks if a file's date is in the protected dates set.
func (t *SpeciesCleanupTask) isProtectedDate(path string, protectedDates map[string]bool) bool {
	// Try to extract date from path (dates are directory names like "2024-01-15")
	datePattern := regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
	matches := datePattern.FindStringSubmatch(path)

	if len(matches) > 1 {
		return protectedDates[matches[1]]
	}

	return false
}

// loadExcludeList loads the file exclusion list.
func (t *SpeciesCleanupTask) loadExcludeList() map[string]bool {
	excludeList := make(map[string]bool)

	excludePath := filepath.Join(t.dataDir, "disk_check_exclude.txt")
	file, err := os.Open(excludePath)
	if err != nil {
		return excludeList
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			excludeList[line] = true
		}
	}

	return excludeList
}
