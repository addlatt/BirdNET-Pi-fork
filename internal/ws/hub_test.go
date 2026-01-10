package ws

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestNewHub(t *testing.T) {
	hub := NewHub()

	if hub == nil {
		t.Fatal("NewHub returned nil")
	}
	if hub.clients == nil {
		t.Error("clients map is nil")
	}
	if hub.channels == nil {
		t.Error("channels map is nil")
	}
}

func TestHubClientCount(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Initially no clients
	if got := hub.GetClientCount(); got != 0 {
		t.Errorf("GetClientCount() = %d, want 0", got)
	}
}

func TestHubChannelClientCount(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// No clients in non-existent channel
	if got := hub.GetChannelClientCount("test"); got != 0 {
		t.Errorf("GetChannelClientCount() = %d, want 0", got)
	}
}

func TestBroadcastDetection(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Wait for hub to start
	time.Sleep(10 * time.Millisecond)

	payload := &DetectionPayload{
		ID:         1,
		Date:       "2024-01-15",
		Time:       "10:30:00",
		SciName:    "Turdus migratorius",
		ComName:    "American Robin",
		Confidence: 0.92,
		FileName:   "robin_001.mp3",
	}

	// Should not error even with no clients
	err := hub.BroadcastDetection(payload)
	if err != nil {
		t.Errorf("BroadcastDetection() error = %v", err)
	}
}

func TestBroadcastStatus(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Wait for hub to start
	time.Sleep(10 * time.Millisecond)

	status := &StatusPayload{
		SystemStatus:  "running",
		MLServiceUp:   true,
		ActiveClients: 0,
	}

	err := hub.BroadcastStatus(status)
	if err != nil {
		t.Errorf("BroadcastStatus() error = %v", err)
	}
}

func TestBroadcastAll(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Wait for hub to start
	time.Sleep(10 * time.Millisecond)

	err := hub.BroadcastAll(TypeStatus, map[string]string{"test": "value"})
	if err != nil {
		t.Errorf("BroadcastAll() error = %v", err)
	}
}

func TestWebSocketIntegration(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.HandleWebSocket(w, r)
	}))
	defer server.Close()

	// Convert http URL to ws URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect WebSocket client
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer ws.Close()

	// Wait for registration
	time.Sleep(50 * time.Millisecond)

	// Verify client connected
	if got := hub.GetClientCount(); got != 1 {
		t.Errorf("GetClientCount() = %d, want 1", got)
	}

	// Client should be auto-subscribed to detections channel
	if got := hub.GetChannelClientCount(ChannelDetections); got != 1 {
		t.Errorf("GetChannelClientCount(detections) = %d, want 1", got)
	}
}

func TestWebSocketReceivesDetection(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.HandleWebSocket(w, r)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer ws.Close()

	// Wait for registration
	time.Sleep(50 * time.Millisecond)

	// Use WaitGroup to synchronize
	var wg sync.WaitGroup
	var receivedMsg WSMessage
	var readErr error

	wg.Add(1)
	go func() {
		defer wg.Done()
		ws.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, message, err := ws.ReadMessage()
		if err != nil {
			readErr = err
			return
		}
		json.Unmarshal(message, &receivedMsg)
	}()

	// Broadcast a detection
	payload := &DetectionPayload{
		ID:         1,
		ComName:    "Test Bird",
		SciName:    "Testus birdus",
		Confidence: 0.85,
	}
	hub.BroadcastDetection(payload)

	wg.Wait()

	if readErr != nil {
		t.Fatalf("failed to read message: %v", readErr)
	}

	if receivedMsg.Type != TypeDetection {
		t.Errorf("message type = %s, want %s", receivedMsg.Type, TypeDetection)
	}
	if receivedMsg.Channel != ChannelDetections {
		t.Errorf("message channel = %s, want %s", receivedMsg.Channel, ChannelDetections)
	}
}

func TestMultipleClients(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.HandleWebSocket(w, r)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect multiple clients
	clients := make([]*websocket.Conn, 3)
	for i := 0; i < 3; i++ {
		ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("failed to connect client %d: %v", i, err)
		}
		clients[i] = ws
		defer ws.Close()
	}

	// Wait for registration
	time.Sleep(100 * time.Millisecond)

	if got := hub.GetClientCount(); got != 3 {
		t.Errorf("GetClientCount() = %d, want 3", got)
	}

	// Close one client
	clients[0].Close()
	time.Sleep(100 * time.Millisecond)

	if got := hub.GetClientCount(); got != 2 {
		t.Errorf("GetClientCount() after close = %d, want 2", got)
	}
}
