package tasks

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/birdnet-pi/birdnet/internal/config"
)

// NotificationTask checks for new species and sends notifications.
// This replaces the species_notifier.sh script.
type NotificationTask struct {
	configMgr *config.Manager
	db        *sql.DB
	homeDir   string
}

// NewNotificationTask creates a new notification task.
func NewNotificationTask(configMgr *config.Manager, db *sql.DB, homeDir string) *NotificationTask {
	return &NotificationTask{
		configMgr: configMgr,
		db:        db,
		homeDir:   homeDir,
	}
}

func (t *NotificationTask) Name() string {
	return "species_notification"
}

func (t *NotificationTask) Description() string {
	return "Checks for new species detections and sends notifications via Apprise"
}

func (t *NotificationTask) DefaultSchedule() string {
	return "*/5 * * * *" // Every 5 minutes
}

func (t *NotificationTask) Timeout() time.Duration {
	return 5 * time.Minute
}

func (t *NotificationTask) Run(ctx context.Context) error {
	cfg := t.configMgr.Get()

	// Check if new species notifications are enabled
	if cfg.AppriseNotifyNewSpecies != 1 {
		log.Println("Species notification: new species notifications disabled")
		return nil
	}

	// Paths
	birdnetDir := filepath.Join(t.homeDir, "BirdNET-Pi")
	idFilePath := cfg.IDFile
	if idFilePath == "" {
		idFilePath = filepath.Join(birdnetDir, "detected_species.txt")
	}
	appriseConfigPath := filepath.Join(birdnetDir, "apprise.txt")

	// Check if apprise config exists and has content
	appriseContent, err := os.ReadFile(appriseConfigPath)
	if err != nil || len(strings.TrimSpace(string(appriseContent))) == 0 {
		log.Println("Species notification: no apprise configuration found, skipping")
		return nil
	}

	// Read current species list
	currentSpecies, err := t.readSpeciesList(idFilePath)
	if err != nil {
		// If file doesn't exist, create it
		currentSpecies = make(map[string]bool)
	}

	// Get species from database
	dbSpecies, err := t.getSpeciesFromDB(ctx)
	if err != nil {
		return fmt.Errorf("failed to get species from database: %w", err)
	}

	// Find new species
	var newSpecies []string
	for _, sp := range dbSpecies {
		if !currentSpecies[sp] {
			newSpecies = append(newSpecies, sp)
		}
	}

	if len(newSpecies) == 0 {
		log.Println("Species notification: no new species detected")
		return nil
	}

	// Send notification for new species
	log.Printf("Species notification: found %d new species: %v", len(newSpecies), newSpecies)

	title := cfg.AppriseNotificationTitle
	if title == "" {
		title = "New Species Detected"
	}

	body := fmt.Sprintf("New Species Detection: %s", strings.Join(newSpecies, ", "))

	if err := t.sendAppriseNotification(ctx, appriseConfigPath, title, body); err != nil {
		log.Printf("Species notification: failed to send notification: %v", err)
		// Don't return error - we still want to update the species list
	}

	// Update species list file
	if err := t.writeSpeciesList(idFilePath, dbSpecies); err != nil {
		return fmt.Errorf("failed to update species list: %w", err)
	}

	return nil
}

func (t *NotificationTask) readSpeciesList(path string) (map[string]bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	species := make(map[string]bool)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			species[line] = true
		}
	}

	return species, scanner.Err()
}

func (t *NotificationTask) getSpeciesFromDB(ctx context.Context) ([]string, error) {
	rows, err := t.db.QueryContext(ctx, "SELECT DISTINCT(Com_Name) FROM detections")
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

	sort.Strings(species)
	return species, rows.Err()
}

func (t *NotificationTask) writeSpeciesList(path string, species []string) error {
	content := strings.Join(species, "\n")
	if len(species) > 0 {
		content += "\n"
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return os.WriteFile(path, []byte(content), 0644)
}

func (t *NotificationTask) sendAppriseNotification(ctx context.Context, configPath, title, body string) error {
	// Find apprise executable
	apprisePath := filepath.Join(t.homeDir, "BirdNET-Pi", "birdnet", "bin", "apprise")

	// Check if apprise exists
	if _, err := os.Stat(apprisePath); os.IsNotExist(err) {
		// Try system apprise
		apprisePath = "apprise"
	}

	cmd := exec.CommandContext(ctx, apprisePath,
		"-vv",
		"-t", title,
		"-b", body,
		"--config="+configPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("apprise failed: %w, output: %s", err, string(output))
	}

	log.Printf("Species notification: sent successfully via apprise")
	return nil
}
