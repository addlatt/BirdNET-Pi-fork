package tasks

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/birdnet-pi/birdnet/internal/config"
)

// SpeciesListUpdateTask updates the species ID file with distinct species from database.
// This replaces the update_species.sh script.
type SpeciesListUpdateTask struct {
	configMgr *config.Manager
	db        *sql.DB
	homeDir   string
}

// NewSpeciesListUpdateTask creates a new species list update task.
func NewSpeciesListUpdateTask(configMgr *config.Manager, db *sql.DB, homeDir string) *SpeciesListUpdateTask {
	return &SpeciesListUpdateTask{
		configMgr: configMgr,
		db:        db,
		homeDir:   homeDir,
	}
}

func (t *SpeciesListUpdateTask) Name() string {
	return "species_list_update"
}

func (t *SpeciesListUpdateTask) Description() string {
	return "Updates the detected species list from database"
}

func (t *SpeciesListUpdateTask) DefaultSchedule() string {
	return "*/15 * * * *" // Every 15 minutes
}

func (t *SpeciesListUpdateTask) Timeout() time.Duration {
	return 5 * time.Minute
}

func (t *SpeciesListUpdateTask) Run(ctx context.Context) error {
	cfg := t.configMgr.Get()

	// Determine ID file path
	idFilePath := cfg.IDFile
	if idFilePath == "" {
		idFilePath = filepath.Join(t.homeDir, "BirdNET-Pi", "detected_species.txt")
	}

	// Query distinct species from database
	rows, err := t.db.QueryContext(ctx, "SELECT DISTINCT(Com_Name) FROM detections")
	if err != nil {
		return fmt.Errorf("failed to query species: %w", err)
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

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating species: %w", err)
	}

	// Sort species alphabetically
	sort.Strings(species)

	// Write to ID file
	content := strings.Join(species, "\n")
	if len(species) > 0 {
		content += "\n"
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(idFilePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(idFilePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write ID file: %w", err)
	}

	log.Printf("Species list update: wrote %d species to %s", len(species), idFilePath)
	return nil
}
