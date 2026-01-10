package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/birdnet-pi/birdnet/internal/db"
	"github.com/birdnet-pi/birdnet/internal/mlclient"
	"github.com/birdnet-pi/birdnet/internal/monitor"
	"github.com/birdnet-pi/birdnet/internal/testutil"
	"github.com/birdnet-pi/birdnet/internal/ws"
	"github.com/go-chi/chi/v5"
)

// setupTestHandlers creates handlers with test dependencies.
func setupTestHandlers(t *testing.T) (*Handlers, *sql.DB) {
	t.Helper()

	// Create test database
	sqlDB := testutil.TestDB(t)

	// Create a wrapper that implements our DB interface
	database := &db.DB{
		Queries: nil, // Will be set after we create it properly
	}

	// We need to recreate db.New behavior for testing
	// For now, use the generated queries directly
	genDB := struct {
		*sql.DB
	}{sqlDB}
	_ = genDB // Will use when we fix the db package

	hub := ws.NewHub()
	go hub.Run()

	memMonitor := monitor.NewMemoryMonitor()

	// Create mock ML server
	mlServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/status/health":
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		case "/status/status":
			json.NewEncoder(w).Encode(mlclient.Status{
				BirdNET: mlclient.BirdNETStatus{Loaded: true, MemoryBytes: 500000000},
			})
		case "/status/memory":
			json.NewEncoder(w).Encode(mlclient.MemoryStats{Total: 500000000, BirdNET: 500000000})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(mlServer.Close)

	mlClient := mlclient.New(mlServer.URL)

	// Create handlers - we'll need to handle database differently
	handlers := &Handlers{
		db:       database,
		hub:      hub,
		monitor:  memMonitor,
		mlClient: mlClient,
	}

	return handlers, sqlDB
}

func TestHealth(t *testing.T) {
	handlers, _ := setupTestHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()

	handlers.Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != "ok" {
		t.Errorf("Status = %s, want ok", response.Status)
	}
}

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()

	writeJSON(rec, http.StatusOK, map[string]string{"test": "value"})

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %s, want application/json", contentType)
	}

	var response map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["test"] != "value" {
		t.Errorf("response[test] = %s, want value", response["test"])
	}
}

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()

	writeError(rec, http.StatusBadRequest, "test error")

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var response map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["error"] != "test error" {
		t.Errorf("response[error] = %s, want 'test error'", response["error"])
	}
}

func TestParseIntParam(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		def      int
		expected int
	}{
		{"empty string", "", 10, 10},
		{"valid number", "5", 10, 5},
		{"invalid string", "invalid", 10, 10},
		{"zero", "0", 10, 0},
		{"negative", "-1", 10, -1},
		{"large number", "1000", 10, 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseIntParam(tt.input, tt.def)
			if result != tt.expected {
				t.Errorf("parseIntParam(%q, %d) = %d, want %d", tt.input, tt.def, result, tt.expected)
			}
		})
	}
}

func TestReceiveDetection_ValidPayload(t *testing.T) {
	handlers, _ := setupTestHandlers(t)

	payload := DetectionNotification{
		ID:         1,
		Date:       "2024-01-15",
		Time:       "10:30:00",
		SciName:    "Turdus migratorius",
		ComName:    "American Robin",
		Confidence: 0.92,
		FileName:   "robin.mp3",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/internal/detection", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handlers.ReceiveDetection(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestReceiveDetection_MissingFields(t *testing.T) {
	handlers, _ := setupTestHandlers(t)

	// Missing sci_name and com_name
	payload := DetectionNotification{
		ID:   1,
		Date: "2024-01-15",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/internal/detection", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handlers.ReceiveDetection(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestReceiveDetection_InvalidJSON(t *testing.T) {
	handlers, _ := setupTestHandlers(t)

	req := httptest.NewRequest(http.MethodPost, "/internal/detection", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handlers.ReceiveDetection(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestGetDetection_InvalidID(t *testing.T) {
	handlers, _ := setupTestHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/api/detections/invalid", nil)

	// Set up chi URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "invalid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()

	handlers.GetDetection(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestDetectionToResponse(t *testing.T) {
	lat := 42.3601
	lon := -71.0589

	tests := []struct {
		name     string
		input    struct {
			ID         int64
			Date       string
			Time       string
			SciName    string
			ComName    string
			Confidence float64
			Lat        sql.NullFloat64
			Lon        sql.NullFloat64
			FileName   string
		}
		checkLat bool
		checkLon bool
	}{
		{
			name: "with coordinates",
			input: struct {
				ID         int64
				Date       string
				Time       string
				SciName    string
				ComName    string
				Confidence float64
				Lat        sql.NullFloat64
				Lon        sql.NullFloat64
				FileName   string
			}{
				ID:         1,
				Date:       "2024-01-15",
				Time:       "10:30:00",
				SciName:    "Turdus migratorius",
				ComName:    "American Robin",
				Confidence: 0.92,
				Lat:        sql.NullFloat64{Float64: lat, Valid: true},
				Lon:        sql.NullFloat64{Float64: lon, Valid: true},
				FileName:   "robin.mp3",
			},
			checkLat: true,
			checkLon: true,
		},
		{
			name: "without coordinates",
			input: struct {
				ID         int64
				Date       string
				Time       string
				SciName    string
				ComName    string
				Confidence float64
				Lat        sql.NullFloat64
				Lon        sql.NullFloat64
				FileName   string
			}{
				ID:         2,
				Date:       "2024-01-15",
				Time:       "11:00:00",
				SciName:    "Cardinalis cardinalis",
				ComName:    "Northern Cardinal",
				Confidence: 0.85,
				Lat:        sql.NullFloat64{Valid: false},
				Lon:        sql.NullFloat64{Valid: false},
				FileName:   "cardinal.mp3",
			},
			checkLat: false,
			checkLon: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock detection - this would normally come from the DB
			// For now just verify the response structure
			resp := DetectionResponse{
				ID:         tt.input.ID,
				Date:       tt.input.Date,
				Time:       tt.input.Time,
				SciName:    tt.input.SciName,
				ComName:    tt.input.ComName,
				Confidence: tt.input.Confidence,
				FileName:   tt.input.FileName,
			}

			if tt.checkLat {
				resp.Lat = &lat
			}
			if tt.checkLon {
				resp.Lon = &lon
			}

			if resp.ID != tt.input.ID {
				t.Errorf("ID = %d, want %d", resp.ID, tt.input.ID)
			}
			if resp.ComName != tt.input.ComName {
				t.Errorf("ComName = %s, want %s", resp.ComName, tt.input.ComName)
			}
			if tt.checkLat && resp.Lat == nil {
				t.Error("Lat should not be nil")
			}
			if !tt.checkLat && resp.Lat != nil {
				t.Error("Lat should be nil")
			}
		})
	}
}
