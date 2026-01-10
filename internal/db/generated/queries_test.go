package db_test

import (
	"context"
	"testing"
	"time"

	db "github.com/birdnet-pi/birdnet/internal/db/generated"
	"github.com/birdnet-pi/birdnet/internal/testutil"
)

func TestGetDetection(t *testing.T) {
	sqlDB := testutil.TestDB(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	// Seed test data
	testutil.SeedDetections(t, sqlDB, testutil.SampleDetections())

	tests := []struct {
		name    string
		id      int64
		wantErr bool
	}{
		{
			name:    "existing detection",
			id:      1,
			wantErr: false,
		},
		{
			name:    "non-existent detection",
			id:      999,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detection, err := queries.GetDetection(ctx, tt.id)
			if tt.wantErr {
				testutil.AssertError(t, err)
				return
			}
			testutil.AssertNoError(t, err)
			testutil.AssertEqual(t, detection.ID, tt.id, "detection ID")
		})
	}
}

func TestListDetections(t *testing.T) {
	sqlDB := testutil.TestDB(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	// Seed test data
	testutil.SeedDetections(t, sqlDB, testutil.SampleDetections())

	tests := []struct {
		name      string
		limit     int64
		offset    int64
		wantCount int
	}{
		{
			name:      "first page",
			limit:     3,
			offset:    0,
			wantCount: 3,
		},
		{
			name:      "second page",
			limit:     3,
			offset:    3,
			wantCount: 3,
		},
		{
			name:      "beyond data",
			limit:     10,
			offset:    100,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detections, err := queries.ListDetections(ctx, db.ListDetectionsParams{
				Limit:  tt.limit,
				Offset: tt.offset,
			})
			testutil.AssertNoError(t, err)
			testutil.AssertEqual(t, len(detections), tt.wantCount, "detection count")
		})
	}
}

func TestListDetectionsByDate(t *testing.T) {
	sqlDB := testutil.TestDB(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	// Seed test data
	testutil.SeedDetections(t, sqlDB, testutil.SampleDetections())

	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	tests := []struct {
		name      string
		date      string
		wantCount int
	}{
		{
			name:      "today's detections",
			date:      today,
			wantCount: 4, // 4 detections today in sample data
		},
		{
			name:      "yesterday's detections",
			date:      yesterday,
			wantCount: 2, // 2 detections yesterday in sample data
		},
		{
			name:      "no detections",
			date:      "2020-01-01",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detections, err := queries.ListDetectionsByDate(ctx, tt.date)
			testutil.AssertNoError(t, err)
			testutil.AssertEqual(t, len(detections), tt.wantCount, "detection count")
		})
	}
}

func TestListSpecies(t *testing.T) {
	sqlDB := testutil.TestDB(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	// Seed test data
	testutil.SeedDetections(t, sqlDB, testutil.SampleDetections())

	species, err := queries.ListSpecies(ctx)
	testutil.AssertNoError(t, err)

	// Should have 4 unique species in sample data
	testutil.AssertEqual(t, len(species), 4, "species count")

	// First species should be American Robin (most detections)
	if len(species) > 0 {
		testutil.AssertEqual(t, species[0].ComName, "American Robin", "top species")
		testutil.AssertEqual(t, species[0].DetectionCount, int64(3), "detection count")
	}
}

func TestCountDetections(t *testing.T) {
	sqlDB := testutil.TestDB(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	// Empty database
	count, err := queries.CountDetections(ctx)
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, count, int64(0), "empty count")

	// Seed test data
	testutil.SeedDetections(t, sqlDB, testutil.SampleDetections())

	count, err = queries.CountDetections(ctx)
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, count, int64(6), "total count")
}

func TestGetTotalSpeciesCount(t *testing.T) {
	sqlDB := testutil.TestDB(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	// Seed test data
	testutil.SeedDetections(t, sqlDB, testutil.SampleDetections())

	count, err := queries.GetTotalSpeciesCount(ctx)
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, count, int64(4), "species count")
}

func TestGetRecentDetections(t *testing.T) {
	sqlDB := testutil.TestDB(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	// Seed test data
	testutil.SeedDetections(t, sqlDB, testutil.SampleDetections())

	tests := []struct {
		name      string
		limit     int64
		wantCount int
	}{
		{
			name:      "get 3 recent",
			limit:     3,
			wantCount: 3,
		},
		{
			name:      "get all",
			limit:     100,
			wantCount: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detections, err := queries.GetRecentDetections(ctx, tt.limit)
			testutil.AssertNoError(t, err)
			testutil.AssertEqual(t, len(detections), tt.wantCount, "detection count")
		})
	}
}

func TestListDetectionsBySpecies(t *testing.T) {
	sqlDB := testutil.TestDB(t)
	queries := db.New(sqlDB)
	ctx := context.Background()

	// Seed test data
	testutil.SeedDetections(t, sqlDB, testutil.SampleDetections())

	tests := []struct {
		name      string
		sciName   string
		comName   string
		wantCount int
	}{
		{
			name:      "by scientific name",
			sciName:   "Turdus migratorius",
			comName:   "",
			wantCount: 3,
		},
		{
			name:      "by common name",
			sciName:   "",
			comName:   "Blue Jay",
			wantCount: 1,
		},
		{
			name:      "no match",
			sciName:   "Unknown species",
			comName:   "Unknown",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detections, err := queries.ListDetectionsBySpecies(ctx, db.ListDetectionsBySpeciesParams{
				SciName: tt.sciName,
				ComName: tt.comName,
				Limit:   100,
				Offset:  0,
			})
			testutil.AssertNoError(t, err)
			testutil.AssertEqual(t, len(detections), tt.wantCount, "detection count")
		})
	}
}
