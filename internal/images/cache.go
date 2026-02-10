package images

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const cacheExpirationDays = 20

// Cache provides a SQLite-backed image cache with blacklist support.
type Cache struct {
	db        *sql.DB
	blacklist map[string]struct{}
	mu        sync.RWMutex
}

// NewCache opens (or creates) the images.db database and initializes the schema.
// If data/blacklisted_images.txt exists, its entries are imported on first run.
func NewCache(dataDir string) (*Cache, error) {
	dbDir := filepath.Join(dataDir, "db")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	dbPath := filepath.Join(dbDir, "images.db")
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open images db: %w", err)
	}

	if err := createSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	c := &Cache{
		db:        db,
		blacklist: make(map[string]struct{}),
	}

	// Import blacklist file on first run (if table is empty and file exists).
	if err := c.importBlacklistFile(dataDir); err != nil {
		// Non-fatal: log but continue.
		fmt.Printf("Warning: failed to import blacklist file: %v\n", err)
	}

	// Load blacklist into memory.
	if err := c.loadBlacklist(); err != nil {
		db.Close()
		return nil, fmt.Errorf("load blacklist: %w", err)
	}

	return c, nil
}

func createSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS images (
		sci_name    TEXT NOT NULL,
		provider    TEXT NOT NULL,
		com_name    TEXT NOT NULL,
		image_url   TEXT NOT NULL,
		title       TEXT NOT NULL,
		source_id   TEXT NOT NULL,
		author_url  TEXT NOT NULL,
		license_url TEXT NOT NULL,
		photos_url  TEXT NOT NULL DEFAULT '',
		cached_at   TEXT NOT NULL,
		PRIMARY KEY (sci_name, provider)
	);
	CREATE TABLE IF NOT EXISTS blacklist (
		source_id TEXT PRIMARY KEY,
		added_at  TEXT NOT NULL
	);`
	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("create schema: %w", err)
	}
	return nil
}

func (c *Cache) importBlacklistFile(dataDir string) error {
	// Only import if the blacklist table is empty.
	var count int
	if err := c.db.QueryRow("SELECT COUNT(*) FROM blacklist").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	filePath := filepath.Join(dataDir, "blacklisted_images.txt")
	f, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No file, nothing to import.
		}
		return err
	}
	defer f.Close()

	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT OR IGNORE INTO blacklist (source_id, added_at) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		id := strings.TrimSpace(scanner.Text())
		if id != "" {
			stmt.Exec(id, now)
		}
	}

	return tx.Commit()
}

func (c *Cache) loadBlacklist() error {
	rows, err := c.db.Query("SELECT source_id FROM blacklist")
	if err != nil {
		return err
	}
	defer rows.Close()

	c.mu.Lock()
	defer c.mu.Unlock()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		c.blacklist[id] = struct{}{}
	}
	return rows.Err()
}

// Get returns the cached image for a species+provider, or nil if not found or expired.
func (c *Cache) Get(sciName, provider string) (*ImageResult, error) {
	row := c.db.QueryRow(`
		SELECT sci_name, provider, com_name, image_url, title,
		       source_id, author_url, license_url, photos_url, cached_at
		FROM images WHERE sci_name = ? AND provider = ?`,
		sciName, provider)

	var r ImageResult
	err := row.Scan(&r.SciName, &r.Provider, &r.ComName, &r.ImageURL, &r.Title,
		&r.SourceID, &r.AuthorURL, &r.LicenseURL, &r.PhotosURL, &r.CachedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Check expiration.
	cachedAt, err := time.Parse(time.RFC3339, r.CachedAt)
	if err != nil {
		// Can't parse timestamp — treat as expired.
		return nil, nil
	}
	if time.Since(cachedAt) > time.Duration(cacheExpirationDays)*24*time.Hour {
		return nil, nil
	}

	return &r, nil
}

// Set inserts or replaces a cached image result.
func (c *Cache) Set(r *ImageResult) error {
	_, err := c.db.Exec(`
		INSERT OR REPLACE INTO images
			(sci_name, provider, com_name, image_url, title, source_id, author_url, license_url, photos_url, cached_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.SciName, r.Provider, r.ComName, r.ImageURL, r.Title,
		r.SourceID, r.AuthorURL, r.LicenseURL, r.PhotosURL, r.CachedAt)
	return err
}

// Delete removes a cached image for a species+provider.
func (c *Cache) Delete(sciName, provider string) error {
	_, err := c.db.Exec("DELETE FROM images WHERE sci_name = ? AND provider = ?", sciName, provider)
	return err
}

// IsBlacklisted checks if a source ID is in the blacklist (in-memory check).
func (c *Cache) IsBlacklisted(sourceID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.blacklist[sourceID]
	return ok
}

// AddToBlacklist adds a source ID to both the database and in-memory blacklist.
func (c *Cache) AddToBlacklist(sourceID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := c.db.Exec("INSERT OR IGNORE INTO blacklist (source_id, added_at) VALUES (?, ?)", sourceID, now)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.blacklist[sourceID] = struct{}{}
	c.mu.Unlock()

	return nil
}

// GetStats returns aggregate counts for the image cache.
func (c *Cache) GetStats() (*CacheStats, error) {
	stats := &CacheStats{}

	err := c.db.QueryRow("SELECT COUNT(*) FROM images WHERE provider = ?", ProviderFlickr).Scan(&stats.FlickrCount)
	if err != nil {
		return nil, err
	}

	err = c.db.QueryRow("SELECT COUNT(*) FROM images WHERE provider = ?", ProviderWikipedia).Scan(&stats.WikipediaCount)
	if err != nil {
		return nil, err
	}

	stats.TotalCount = stats.FlickrCount + stats.WikipediaCount

	cutoff := time.Now().UTC().Add(-time.Duration(cacheExpirationDays) * 24 * time.Hour).Format(time.RFC3339)
	err = c.db.QueryRow("SELECT COUNT(*) FROM images WHERE cached_at < ?", cutoff).Scan(&stats.ExpiredCount)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// ListExpired returns all cached images that have passed the expiration window.
func (c *Cache) ListExpired() ([]*ImageResult, error) {
	cutoff := time.Now().UTC().Add(-time.Duration(cacheExpirationDays) * 24 * time.Hour).Format(time.RFC3339)

	rows, err := c.db.Query(`
		SELECT sci_name, provider, com_name, image_url, title,
		       source_id, author_url, license_url, photos_url, cached_at
		FROM images WHERE cached_at < ?`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*ImageResult
	for rows.Next() {
		var r ImageResult
		if err := rows.Scan(&r.SciName, &r.Provider, &r.ComName, &r.ImageURL, &r.Title,
			&r.SourceID, &r.AuthorURL, &r.LicenseURL, &r.PhotosURL, &r.CachedAt); err != nil {
			return nil, err
		}
		results = append(results, &r)
	}
	return results, rows.Err()
}

// Close closes the underlying database connection.
func (c *Cache) Close() error {
	return c.db.Close()
}
