package tasks

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/birdnet-pi/birdnet/internal/config"
)

// DataResetTask resets all collected data to start fresh.
// This replaces the clear_all_data.sh script.
type DataResetTask struct {
	configMgr    *config.Manager
	db           *sql.DB
	homeDir      string
	birdsongsDir string
	dataDir      string
}

// NewDataResetTask creates a new data reset task.
func NewDataResetTask(configMgr *config.Manager, db *sql.DB, homeDir, birdsongsDir, dataDir string) *DataResetTask {
	return &DataResetTask{
		configMgr:    configMgr,
		db:           db,
		homeDir:      homeDir,
		birdsongsDir: birdsongsDir,
		dataDir:      dataDir,
	}
}

func (t *DataResetTask) Name() string {
	return "data_reset"
}

func (t *DataResetTask) Description() string {
	return "Resets all collected data and starts fresh (WARNING: destructive)"
}

func (t *DataResetTask) DefaultSchedule() string {
	return "" // No automatic schedule - manual only
}

func (t *DataResetTask) Timeout() time.Duration {
	return 30 * time.Minute
}

func (t *DataResetTask) Run(ctx context.Context) error {
	log.Println("Data reset: starting full data reset")

	// Step 1: Stop services
	log.Println("Data reset: stopping services")
	if err := t.stopServices(ctx); err != nil {
		log.Printf("Data reset: warning - failed to stop some services: %v", err)
		// Continue anyway
	}

	// Step 2: Remove recordings directory
	log.Println("Data reset: removing recordings directory")
	if err := os.RemoveAll(t.birdsongsDir); err != nil {
		return fmt.Errorf("failed to remove recordings directory: %w", err)
	}

	// Step 3: Remove ID file
	cfg := t.configMgr.Get()
	if cfg.IDFile != "" {
		log.Println("Data reset: removing ID file")
		os.Remove(cfg.IDFile)
	}

	// Step 4: Remove BirdDB.txt
	birdDBPath := filepath.Join(t.homeDir, "BirdNET-Pi", "BirdDB.txt")
	log.Println("Data reset: removing BirdDB.txt")
	os.Remove(birdDBPath)

	// Step 5: Recreate necessary directories
	log.Println("Data reset: creating directories")
	if err := t.createDirectories(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Step 6: Create symlinks for species lists
	log.Println("Data reset: creating symlinks")
	if err := t.createSymlinks(); err != nil {
		log.Printf("Data reset: warning - failed to create some symlinks: %v", err)
	}

	// Step 7: Reset database
	log.Println("Data reset: resetting database")
	if err := t.resetDatabase(ctx); err != nil {
		return fmt.Errorf("failed to reset database: %w", err)
	}

	// Step 8: Recreate BirdDB.txt
	log.Println("Data reset: recreating BirdDB.txt")
	if err := t.createBirdDBFile(); err != nil {
		log.Printf("Data reset: warning - failed to create BirdDB.txt: %v", err)
	}

	// Step 9: Restart services
	log.Println("Data reset: restarting services")
	if err := t.restartServices(ctx); err != nil {
		log.Printf("Data reset: warning - failed to restart some services: %v", err)
	}

	log.Println("Data reset: completed successfully")
	return nil
}

func (t *DataResetTask) stopServices(ctx context.Context) error {
	services := []string{
		"birdnet_recording.service",
		"birdnet_analysis.service",
	}

	for _, svc := range services {
		cmd := exec.CommandContext(ctx, "sudo", "systemctl", "stop", svc)
		if err := cmd.Run(); err != nil {
			log.Printf("Data reset: failed to stop %s: %v", svc, err)
		}
	}
	return nil
}

func (t *DataResetTask) createDirectories() error {
	extractedDir := filepath.Join(t.birdsongsDir, "Extracted")
	processedDir := filepath.Join(t.birdsongsDir, "Processed")
	streamDir := filepath.Join(t.birdsongsDir, "StreamData")

	dirs := []string{
		extractedDir,
		filepath.Join(extractedDir, "By_Date"),
		filepath.Join(extractedDir, "Charts"),
		processedDir,
		streamDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create %s: %w", dir, err)
		}
	}

	return nil
}

func (t *DataResetTask) createSymlinks() error {
	birdnetDir := filepath.Join(t.homeDir, "BirdNET-Pi")
	speciesListsDir := filepath.Join(birdnetDir, "data", "species_lists")

	// Ensure species lists directory exists
	os.MkdirAll(speciesListsDir, 0755)

	symlinks := map[string]string{
		filepath.Join(birdnetDir, "exclude_species_list.txt"):   filepath.Join(speciesListsDir, "exclude_species_list.txt"),
		filepath.Join(birdnetDir, "confirmed_species_list.txt"): filepath.Join(speciesListsDir, "confirmed_species_list.txt"),
		filepath.Join(birdnetDir, "include_species_list.txt"):   filepath.Join(speciesListsDir, "include_species_list.txt"),
		filepath.Join(birdnetDir, "whitelist_species_list.txt"): filepath.Join(speciesListsDir, "whitelist_species_list.txt"),
	}

	for src, dst := range symlinks {
		// Remove existing symlink if any
		os.Remove(dst)
		// Create new symlink
		if err := os.Symlink(src, dst); err != nil && !os.IsExist(err) {
			log.Printf("Data reset: failed to create symlink %s -> %s: %v", dst, src, err)
		}
	}

	return nil
}

func (t *DataResetTask) resetDatabase(ctx context.Context) error {
	// Drop and recreate the detections table
	_, err := t.db.ExecContext(ctx, `DROP TABLE IF EXISTS detections`)
	if err != nil {
		return fmt.Errorf("failed to drop detections table: %w", err)
	}

	// Schema matches migrations/000001_initial_schema.up.sql
	_, err = t.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS detections (
			Date DATE NOT NULL,
			Time TIME NOT NULL,
			Sci_Name VARCHAR(100) NOT NULL,
			Com_Name VARCHAR(100) NOT NULL,
			Confidence REAL,
			Lat REAL,
			Lon REAL,
			Cutoff REAL,
			Week INTEGER,
			Sens REAL,
			Overlap REAL,
			File_Name VARCHAR(100) NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create detections table: %w", err)
	}

	// Create indexes matching migration schema
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_detections_date_time ON detections(Date DESC, Time DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_detections_sci_name ON detections(Sci_Name)`,
		`CREATE INDEX IF NOT EXISTS idx_detections_com_name ON detections(Com_Name)`,
		`CREATE INDEX IF NOT EXISTS idx_detections_confidence ON detections(Confidence)`,
	}
	for _, idx := range indexes {
		if _, err := t.db.ExecContext(ctx, idx); err != nil {
			log.Printf("Data reset: warning - failed to create index: %v", err)
		}
	}

	return nil
}

func (t *DataResetTask) createBirdDBFile() error {
	birdDBPath := filepath.Join(t.homeDir, "BirdNET-Pi", "BirdDB.txt")
	header := "Date;Time;Sci_Name;Com_Name;Confidence;Lat;Lon;Cutoff;Week;Sens;Overlap\n"

	if err := os.WriteFile(birdDBPath, []byte(header), 0644); err != nil {
		return err
	}

	return nil
}

func (t *DataResetTask) restartServices(ctx context.Context) error {
	services := []string{
		"birdnet_recording.service",
		"birdnet_analysis.service",
		"chart_viewer.service",
		"spectrogram_viewer.service",
	}

	for _, svc := range services {
		cmd := exec.CommandContext(ctx, "sudo", "systemctl", "restart", svc)
		if err := cmd.Run(); err != nil {
			log.Printf("Data reset: failed to restart %s: %v", svc, err)
		}
	}

	return nil
}
