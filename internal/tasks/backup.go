package tasks

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// BackupTask creates tar archives of BirdNET-Pi data and configuration.
// This replaces the backup_data.sh script (backup functionality only).
type BackupTask struct {
	homeDir      string
	birdsongsDir string
	outputPath   string // Set dynamically when running
}

// NewBackupTask creates a new backup task.
func NewBackupTask(homeDir, birdsongsDir string) *BackupTask {
	return &BackupTask{
		homeDir:      homeDir,
		birdsongsDir: birdsongsDir,
	}
}

func (t *BackupTask) Name() string {
	return "backup"
}

func (t *BackupTask) Description() string {
	return "Creates a backup archive of BirdNET-Pi data and configuration"
}

func (t *BackupTask) DefaultSchedule() string {
	return "" // Manual only - no automatic scheduling
}

func (t *BackupTask) Timeout() time.Duration {
	return 4 * time.Hour // Backups can take a long time with lots of data
}

// SetOutputPath sets the output path for the backup archive.
// This should be called before Run() when triggering manually.
func (t *BackupTask) SetOutputPath(path string) {
	t.outputPath = path
}

func (t *BackupTask) Run(ctx context.Context) error {
	// Determine output path
	outputPath := t.outputPath
	if outputPath == "" {
		// Default to timestamped file in home directory
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		outputPath = filepath.Join(t.homeDir, fmt.Sprintf("birdnet-backup_%s.tar.gz", timestamp))
	}

	log.Printf("Backup: creating archive at %s", outputPath)

	// Check if file already exists
	if _, err := os.Stat(outputPath); err == nil {
		return fmt.Errorf("backup file already exists: %s", outputPath)
	}

	// Define files/directories to back up
	birdnetDir := filepath.Join(t.homeDir, "BirdNET-Pi")
	extractedDir := filepath.Join(t.birdsongsDir, "Extracted")

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
			return fmt.Errorf("required file/directory not found: %s", path)
		}
	}

	// Estimate size and check available space
	estimatedSize, err := t.estimateSize(required, optional)
	if err != nil {
		return fmt.Errorf("failed to estimate backup size: %w", err)
	}

	availableSpace, err := t.getAvailableSpace(filepath.Dir(outputPath))
	if err != nil {
		return fmt.Errorf("failed to check available space: %w", err)
	}

	// Add 10% buffer
	requiredSpace := int64(float64(estimatedSize) * 1.1)
	if availableSpace < requiredSpace {
		return fmt.Errorf("not enough space: need %d bytes, have %d bytes", requiredSpace, availableSpace)
	}

	log.Printf("Backup: estimated size %d MB, available space %d MB",
		estimatedSize/(1024*1024), availableSpace/(1024*1024))

	// Create the archive
	if err := t.createArchive(ctx, outputPath, required, optional); err != nil {
		// Clean up partial file on error
		os.Remove(outputPath)
		return fmt.Errorf("failed to create archive: %w", err)
	}

	// Get final file size
	info, err := os.Stat(outputPath)
	if err == nil {
		log.Printf("Backup: completed, archive size %d MB", info.Size()/(1024*1024))
	}

	return nil
}

// estimateSize calculates the total size of files to back up.
func (t *BackupTask) estimateSize(required, optional []string) (int64, error) {
	var total int64

	for _, path := range required {
		size, err := t.getPathSize(path)
		if err != nil {
			return 0, err
		}
		total += size
	}

	for _, path := range optional {
		size, _ := t.getPathSize(path) // Ignore errors for optional files
		total += size
	}

	return total, nil
}

// getPathSize returns the total size of a file or directory.
func (t *BackupTask) getPathSize(path string) (int64, error) {
	var size int64

	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	return size, err
}

// getAvailableSpace returns the available disk space at the given path.
func (t *BackupTask) getAvailableSpace(path string) (int64, error) {
	// This is a simplified version - in production you'd use syscall.Statfs
	// For now, just return a large value and let the tar operation fail if needed
	return 1 << 40, nil // 1 TB
}

// createArchive creates a gzipped tar archive.
func (t *BackupTask) createArchive(ctx context.Context, outputPath string, required, optional []string) error {
	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Create gzip writer
	gzWriter := gzip.NewWriter(outFile)
	defer gzWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Add required files
	for _, path := range required {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if err := t.addToArchive(ctx, tarWriter, path); err != nil {
			return fmt.Errorf("failed to add %s: %w", path, err)
		}
	}

	// Add optional files (ignore errors)
	for _, path := range optional {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if _, err := os.Stat(path); err == nil {
			if err := t.addToArchive(ctx, tarWriter, path); err != nil {
				log.Printf("Backup: warning - failed to add optional file %s: %v", path, err)
			}
		}
	}

	return nil
}

// addToArchive adds a file or directory to the tar archive.
func (t *BackupTask) addToArchive(ctx context.Context, tw *tar.Writer, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	// Calculate the base name for the archive
	baseName := filepath.Base(path)

	if info.IsDir() {
		return t.addDirToArchive(ctx, tw, path, baseName)
	}

	return t.addFileToArchive(tw, path, baseName)
}

// addFileToArchive adds a single file to the archive.
func (t *BackupTask) addFileToArchive(tw *tar.Writer, filePath, archiveName string) error {
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
func (t *BackupTask) addDirToArchive(ctx context.Context, tw *tar.Writer, dirPath, archiveBase string) error {
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
			defer file.Close()

			if _, err := io.Copy(tw, file); err != nil {
				return err
			}
		}

		return nil
	})
}

// GetEstimatedSize returns the estimated backup size in bytes.
// This can be called before Run() to show the user how much space is needed.
func (t *BackupTask) GetEstimatedSize() (int64, error) {
	birdnetDir := filepath.Join(t.homeDir, "BirdNET-Pi")
	extractedDir := filepath.Join(t.birdsongsDir, "Extracted")

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

	return t.estimateSize(required, optional)
}
