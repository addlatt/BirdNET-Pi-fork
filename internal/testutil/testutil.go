// Package testutil provides testing utilities and helpers.
package testutil

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// TestDB creates an in-memory SQLite database for testing.
// It automatically closes the database when the test completes.
func TestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	// Create schema
	schema := `
		CREATE TABLE IF NOT EXISTS detections (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date DATE NOT NULL,
			time TIME NOT NULL,
			sci_name VARCHAR(100) NOT NULL,
			com_name VARCHAR(100) NOT NULL,
			confidence REAL NOT NULL,
			lat REAL,
			lon REAL,
			cutoff REAL,
			week INTEGER,
			sens REAL,
			overlap REAL,
			file_name VARCHAR(255) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_detections_date_time ON detections(date DESC, time DESC);
		CREATE INDEX IF NOT EXISTS idx_detections_sci_name ON detections(sci_name);
		CREATE INDEX IF NOT EXISTS idx_detections_com_name ON detections(com_name);
	`

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		t.Fatalf("failed to create schema: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

// SeedDetections inserts test detection records into the database.
func SeedDetections(t *testing.T, db *sql.DB, detections []Detection) {
	t.Helper()

	stmt, err := db.Prepare(`
		INSERT INTO detections (date, time, sci_name, com_name, confidence, lat, lon, file_name, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		t.Fatalf("failed to prepare insert statement: %v", err)
	}
	defer stmt.Close()

	for _, d := range detections {
		createdAt := d.CreatedAt
		if createdAt.IsZero() {
			createdAt = time.Now()
		}
		_, err := stmt.Exec(d.Date, d.Time, d.SciName, d.ComName, d.Confidence, d.Lat, d.Lon, d.FileName, createdAt)
		if err != nil {
			t.Fatalf("failed to insert detection: %v", err)
		}
	}
}

// Detection represents a test detection record.
type Detection struct {
	Date       string
	Time       string
	SciName    string
	ComName    string
	Confidence float64
	Lat        *float64
	Lon        *float64
	FileName   string
	CreatedAt  time.Time
}

// SampleDetections returns a slice of sample detection data for testing.
func SampleDetections() []Detection {
	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	lat := 42.3601
	lon := -71.0589

	return []Detection{
		{Date: today, Time: "08:30:00", SciName: "Turdus migratorius", ComName: "American Robin", Confidence: 0.92, Lat: &lat, Lon: &lon, FileName: "robin_001.mp3"},
		{Date: today, Time: "09:15:00", SciName: "Cardinalis cardinalis", ComName: "Northern Cardinal", Confidence: 0.88, Lat: &lat, Lon: &lon, FileName: "cardinal_001.mp3"},
		{Date: today, Time: "10:00:00", SciName: "Turdus migratorius", ComName: "American Robin", Confidence: 0.75, Lat: &lat, Lon: &lon, FileName: "robin_002.mp3"},
		{Date: today, Time: "11:30:00", SciName: "Cyanocitta cristata", ComName: "Blue Jay", Confidence: 0.95, Lat: &lat, Lon: &lon, FileName: "bluejay_001.mp3"},
		{Date: yesterday, Time: "07:00:00", SciName: "Zenaida macroura", ComName: "Mourning Dove", Confidence: 0.82, Lat: &lat, Lon: &lon, FileName: "dove_001.mp3"},
		{Date: yesterday, Time: "14:30:00", SciName: "Turdus migratorius", ComName: "American Robin", Confidence: 0.91, Lat: &lat, Lon: &lon, FileName: "robin_003.mp3"},
	}
}

// AssertEqual is a helper for comparing values in tests.
func AssertEqual[T comparable](t *testing.T, got, want T, msg string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %v, want %v", msg, got, want)
	}
}

// AssertNoError fails the test if err is not nil.
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// AssertError fails the test if err is nil.
func AssertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
