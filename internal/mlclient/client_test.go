package mlclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// mockMLServer creates a test server that mimics the Python ML service.
func mockMLServer(t *testing.T, handlers map[string]http.HandlerFunc) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	for path, handler := range handlers {
		mux.HandleFunc(path, handler)
	}

	return httptest.NewServer(mux)
}

func TestNew(t *testing.T) {
	client := New("http://localhost:8001")

	if client == nil {
		t.Fatal("New returned nil")
	}
	if client.baseURL != "http://localhost:8001" {
		t.Errorf("baseURL = %s, want http://localhost:8001", client.baseURL)
	}
}

func TestWithTimeout(t *testing.T) {
	client := New("http://localhost:8001")
	newClient := client.WithTimeout(5 * time.Second)

	if newClient == nil {
		t.Fatal("WithTimeout returned nil")
	}
	if newClient.baseURL != client.baseURL {
		t.Error("baseURL should be preserved")
	}
	if newClient.httpClient.Timeout != 5*time.Second {
		t.Errorf("Timeout = %v, want 5s", newClient.httpClient.Timeout)
	}
}

func TestGetHealth_Success(t *testing.T) {
	server := mockMLServer(t, map[string]http.HandlerFunc{
		"/status/health": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("Method = %s, want GET", r.Method)
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		},
	})
	defer server.Close()

	client := New(server.URL)
	err := client.GetHealth(context.Background())

	if err != nil {
		t.Errorf("GetHealth() error = %v", err)
	}
}

func TestGetHealth_ServiceDown(t *testing.T) {
	// Use a server that's immediately closed
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	client := New(server.URL)
	err := client.GetHealth(context.Background())

	if err == nil {
		t.Error("GetHealth() should error when service is down")
	}
}

func TestGetHealth_BadStatus(t *testing.T) {
	server := mockMLServer(t, map[string]http.HandlerFunc{
		"/status/health": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "internal error"}`))
		},
	})
	defer server.Close()

	client := New(server.URL)
	err := client.GetHealth(context.Background())

	if err == nil {
		t.Error("GetHealth() should error on non-200 status")
	}
}

func TestGetStatus_Success(t *testing.T) {
	server := mockMLServer(t, map[string]http.HandlerFunc{
		"/status/status": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(Status{
				BirdNET: BirdNETStatus{
					Loaded:      true,
					MemoryBytes: 500000000,
				},
				VAD: VADStatus{
					Enabled: false,
				},
				LLM: LLMStatus{
					Enabled: false,
					Loaded:  false,
				},
			})
		},
	})
	defer server.Close()

	client := New(server.URL)
	status, err := client.GetStatus(context.Background())

	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}
	if !status.BirdNET.Loaded {
		t.Error("BirdNET.Loaded should be true")
	}
	if status.BirdNET.MemoryBytes != 500000000 {
		t.Errorf("BirdNET.MemoryBytes = %d, want 500000000", status.BirdNET.MemoryBytes)
	}
}

func TestGetMemoryUsage_Success(t *testing.T) {
	server := mockMLServer(t, map[string]http.HandlerFunc{
		"/status/memory": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(MemoryStats{
				BirdNET: 500000000,
				VAD:     0,
				LLM:     0,
				Total:   500000000,
			})
		},
	})
	defer server.Close()

	client := New(server.URL)
	stats, err := client.GetMemoryUsage(context.Background())

	if err != nil {
		t.Fatalf("GetMemoryUsage() error = %v", err)
	}
	if stats.BirdNET != 500000000 {
		t.Errorf("BirdNET = %d, want 500000000", stats.BirdNET)
	}
	if stats.Total != 500000000 {
		t.Errorf("Total = %d, want 500000000", stats.Total)
	}
}

func TestIsHealthy(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantResult bool
	}{
		{
			name: "healthy service",
			handler: func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			},
			wantResult: true,
		},
		{
			name: "unhealthy service",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			},
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := mockMLServer(t, map[string]http.HandlerFunc{
				"/status/health": tt.handler,
			})
			defer server.Close()

			client := New(server.URL)
			result := client.IsHealthy(context.Background())

			if result != tt.wantResult {
				t.Errorf("IsHealthy() = %v, want %v", result, tt.wantResult)
			}
		})
	}
}

func TestCheckVAD_NotImplemented(t *testing.T) {
	server := mockMLServer(t, map[string]http.HandlerFunc{
		"/vad/check": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotImplemented)
			json.NewEncoder(w).Encode(map[string]string{"detail": "VAD not implemented yet"})
		},
	})
	defer server.Close()

	client := New(server.URL)
	_, err := client.CheckVAD(context.Background(), "/path/to/audio.wav")

	if err == nil {
		t.Error("CheckVAD() should error with 501")
	}
}

func TestCheckVAD_Success(t *testing.T) {
	server := mockMLServer(t, map[string]http.HandlerFunc{
		"/vad/check": func(w http.ResponseWriter, r *http.Request) {
			// Verify request body
			var req VADRequest
			json.NewDecoder(r.Body).Decode(&req)
			if req.AudioPath != "/path/to/audio.wav" {
				t.Errorf("AudioPath = %s, want /path/to/audio.wav", req.AudioPath)
			}

			json.NewEncoder(w).Encode(VADResult{
				HasSpeech:      true,
				SpeechScore:    0.85,
				SpeechDuration: 2.5,
				ShouldSkip:     true,
			})
		},
	})
	defer server.Close()

	client := New(server.URL)
	result, err := client.CheckVAD(context.Background(), "/path/to/audio.wav")

	if err != nil {
		t.Fatalf("CheckVAD() error = %v", err)
	}
	if !result.HasSpeech {
		t.Error("HasSpeech should be true")
	}
	if result.SpeechScore != 0.85 {
		t.Errorf("SpeechScore = %f, want 0.85", result.SpeechScore)
	}
}

func TestAskLLM_NotImplemented(t *testing.T) {
	server := mockMLServer(t, map[string]http.HandlerFunc{
		"/llm/ask": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotImplemented)
			json.NewEncoder(w).Encode(map[string]string{"detail": "LLM not implemented yet"})
		},
	})
	defer server.Close()

	client := New(server.URL)
	_, err := client.AskLLM(context.Background(), &LLMRequest{Question: "test"})

	if err == nil {
		t.Error("AskLLM() should error with 501")
	}
}

func TestAskLLM_Success(t *testing.T) {
	server := mockMLServer(t, map[string]http.HandlerFunc{
		"/llm/ask": func(w http.ResponseWriter, r *http.Request) {
			var req LLMRequest
			json.NewDecoder(r.Body).Decode(&req)

			json.NewEncoder(w).Encode(LLMResponse{
				Answer:     "This is a test response",
				TokensUsed: 42,
			})
		},
	})
	defer server.Close()

	client := New(server.URL)
	result, err := client.AskLLM(context.Background(), &LLMRequest{
		Question:    "What bird is this?",
		MaxTokens:   100,
		Temperature: 0.7,
	})

	if err != nil {
		t.Fatalf("AskLLM() error = %v", err)
	}
	if result.Answer != "This is a test response" {
		t.Errorf("Answer = %s", result.Answer)
	}
	if result.TokensUsed != 42 {
		t.Errorf("TokensUsed = %d, want 42", result.TokensUsed)
	}
}

func TestContextCancellation(t *testing.T) {
	// Server that delays response
	server := mockMLServer(t, map[string]http.HandlerFunc{
		"/status/health": func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(5 * time.Second)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		},
	})
	defer server.Close()

	client := New(server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := client.GetHealth(ctx)
	if err == nil {
		t.Error("GetHealth() should error on context timeout")
	}
}
