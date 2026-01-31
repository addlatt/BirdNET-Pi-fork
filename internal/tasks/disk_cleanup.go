// Package tasks provides concrete task implementations for the scheduler.
package tasks

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/birdnet-pi/birdnet/internal/config"
)

// DiskCleanupTask monitors disk usage and purges old recordings when threshold is exceeded.
// This replaces the disk_check.sh cron script.
type DiskCleanupTask struct {
	configMgr    *config.Manager
	extractedDir string
	processedDir string
	dataDir      string
}

// NewDiskCleanupTask creates a new disk cleanup task.
func NewDiskCleanupTask(configMgr *config.Manager, extractedDir, processedDir, dataDir string) *DiskCleanupTask {
	return &DiskCleanupTask{
		configMgr:    configMgr,
		extractedDir: extractedDir,
		processedDir: processedDir,
		dataDir:      dataDir,
	}
}

func (t *DiskCleanupTask) Name() string {
	return "disk_cleanup"
}

func (t *DiskCleanupTask) Description() string {
	return "Monitors disk usage and purges old recordings when threshold is exceeded"
}

func (t *DiskCleanupTask) DefaultSchedule() string {
	return "*/5 * * * *" // Every 5 minutes
}

func (t *DiskCleanupTask) Timeout() time.Duration {
	return 30 * time.Minute
}

func (t *DiskCleanupTask) Run(ctx context.Context) error {
	cfg := t.configMgr.Get()

	// Get threshold from config, default to 95%
	purgeThreshold := cfg.PurgeThreshold
	if purgeThreshold == 0 {
		purgeThreshold = 95
	}

	// Get disk action from config
	fullDiskAction := strings.ToLower(cfg.FullDisk)
	if fullDiskAction == "" {
		fullDiskAction = "purge"
	}

	// Check disk usage
	usedPercent, err := t.getDiskUsage(t.extractedDir)
	if err != nil {
		return fmt.Errorf("failed to get disk usage: %w", err)
	}

	log.Printf("Disk cleanup: usage is %d%%, threshold is %d%%", usedPercent, purgeThreshold)

	if usedPercent < purgeThreshold {
		log.Printf("Disk cleanup: no action needed")
		return nil
	}

	// Disk is above threshold
	switch fullDiskAction {
	case "purge":
		if err := t.purgeOldRecordings(ctx); err != nil {
			return fmt.Errorf("failed to purge recordings: %w", err)
		}
	case "keep":
		log.Printf("Disk cleanup: disk full but action is 'keep', stopping core services")
		if err := t.stopCoreServices(ctx); err != nil {
			return fmt.Errorf("failed to stop core services: %w", err)
		}
	default:
		return fmt.Errorf("unknown full_disk action: %s", fullDiskAction)
	}

	// Check again after first pass
	usedPercent, err = t.getDiskUsage(t.extractedDir)
	if err != nil {
		return fmt.Errorf("failed to get disk usage after cleanup: %w", err)
	}

	if usedPercent >= purgeThreshold && fullDiskAction == "purge" {
		log.Printf("Disk cleanup: still at %d%%, removing processed files", usedPercent)
		if err := t.removeProcessedFiles(ctx); err != nil {
			return fmt.Errorf("failed to remove processed files: %w", err)
		}
	}

	return nil
}

// getDiskUsage returns the disk usage percentage for the given path.
func (t *DiskCleanupTask) getDiskUsage(path string) (int, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bfree * uint64(stat.Bsize)
	used := total - free

	if total == 0 {
		return 0, nil
	}

	return int((used * 100) / total), nil
}

// purgeOldRecordings removes old recordings to free disk space.
func (t *DiskCleanupTask) purgeOldRecordings(ctx context.Context) error {
	byDateDir := filepath.Join(t.extractedDir, "By_Date")

	// Load exclude list
	excludeList := t.loadExcludeList()

	// Check if exclude list is properly set up (contains ##start marker)
	excludeListPath := filepath.Join(t.dataDir, "disk_check_exclude.txt")
	if !t.hasStartMarker(excludeListPath) {
		log.Printf("Disk cleanup: exclude list missing ##start marker, skipping purge")
		return nil
	}

	// Get all files
	files, err := t.collectFiles(byDateDir)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		log.Printf("Disk cleanup: no files to purge")
		return nil
	}

	// Count directories (species) to calculate target
	dirs, err := t.countSpeciesDirs(byDateDir)
	if err != nil {
		return err
	}

	if dirs == 0 {
		return nil
	}

	// Calculate how many files to delete
	filesToDelete := len(files) / dirs

	log.Printf("Disk cleanup: found %d files across %d species, deleting ~%d files",
		len(files), dirs, filesToDelete)

	// Sort files by path (which includes date)
	sort.Strings(files)

	deleted := 0
	for _, file := range files {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if deleted >= filesToDelete {
			break
		}

		// Check if file is excluded
		relPath := strings.TrimPrefix(file, byDateDir+"/")
		if excludeList[relPath] {
			continue
		}

		if err := os.Remove(file); err != nil {
			log.Printf("Disk cleanup: failed to remove %s: %v", file, err)
			continue
		}
		deleted++
	}

	log.Printf("Disk cleanup: deleted %d files", deleted)

	// Clean up empty directories older than 90 days
	t.removeEmptyDirs(ctx, filepath.Dir(t.extractedDir))
	t.removeEmptyDirs(ctx, byDateDir)

	return nil
}

// collectFiles returns all audio files in the By_Date directory.
func (t *DiskCleanupTask) collectFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if info.IsDir() {
			return nil
		}
		// Include audio files
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".mp3" || ext == ".wav" || ext == ".flac" || ext == ".ogg" || ext == ".opus" {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}

// countSpeciesDirs counts the number of species directories.
func (t *DiskCleanupTask) countSpeciesDirs(byDateDir string) (int, error) {
	speciesDirs := make(map[string]bool)

	err := filepath.Walk(byDateDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			return nil
		}

		// Species dirs are one level below date dirs
		rel, _ := filepath.Rel(byDateDir, path)
		parts := strings.Split(rel, string(os.PathSeparator))
		if len(parts) == 2 {
			speciesDirs[parts[1]] = true
		}
		return nil
	})

	return len(speciesDirs), err
}

// loadExcludeList loads the file exclusion list.
func (t *DiskCleanupTask) loadExcludeList() map[string]bool {
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

// hasStartMarker checks if the exclude list has the ##start marker.
func (t *DiskCleanupTask) hasStartMarker(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == "##start" {
			return true
		}
	}
	return false
}

// removeEmptyDirs removes empty directories.
func (t *DiskCleanupTask) removeEmptyDirs(ctx context.Context, dir string) {
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil || !info.IsDir() {
			return nil
		}
		// Check if directory is empty
		entries, err := os.ReadDir(path)
		if err == nil && len(entries) == 0 {
			// Check if older than 90 days
			if time.Since(info.ModTime()) > 90*24*time.Hour {
				os.Remove(path)
			}
		}
		return nil
	})
}

// removeProcessedFiles removes all files in the processed directory.
func (t *DiskCleanupTask) removeProcessedFiles(ctx context.Context) error {
	if t.processedDir == "" {
		return nil
	}

	entries, err := os.ReadDir(t.processedDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		path := filepath.Join(t.processedDir, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			log.Printf("Disk cleanup: failed to remove %s: %v", path, err)
		}
	}

	return nil
}

// stopCoreServices stops the BirdNET core services.
func (t *DiskCleanupTask) stopCoreServices(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "/usr/local/bin/stop_core_services.sh")
	return cmd.Run()
}
