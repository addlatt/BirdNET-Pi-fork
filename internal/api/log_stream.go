package api

import (
	"bufio"
	"context"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

var logUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// LogStreamHandler handles WebSocket connections for streaming logs.
// This replaces the birdnet_log.sh script.
type LogStreamHandler struct {
	homeDir string
}

// NewLogStreamHandler creates a new log stream handler.
func NewLogStreamHandler(homeDir string) *LogStreamHandler {
	return &LogStreamHandler{
		homeDir: homeDir,
	}
}

// StreamLogs handles GET /ws/logs requests.
// Streams logs from journalctl via WebSocket.
func (h *Handlers) StreamLogs(w http.ResponseWriter, r *http.Request) {
	conn, err := logUpgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Could not upgrade connection", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	// Get the home directory for path stripping
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		homeDir = "/home/birdnet"
	}

	// Create context that cancels when client disconnects
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Handle client disconnection
	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				cancel()
				return
			}
		}
	}()

	// Start journalctl process
	cmd := exec.CommandContext(ctx, "journalctl",
		"--no-hostname",
		"-q",
		"-o", "short",
		"-f", // Follow
		"-u", "birdnet_analysis",
		"-u", "birdnet_recording",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("Error: Could not start log stream"))
		return
	}

	if err := cmd.Start(); err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("Error: Could not start journalctl"))
		return
	}

	// Process and stream logs
	scanner := bufio.NewScanner(stdout)

	// Date prefix pattern (e.g., "Jan 15 ")
	datePrefix := time.Now().Format("Jan 02 ")

	// Patterns to filter out
	filterPatterns := []*regexp.Regexp{
		regexp.MustCompile(`Line`),
		regexp.MustCompile(`find`),
		regexp.MustCompile(`systemd`),
	}

	// Pattern to simplify log format
	servicePattern := regexp.MustCompile(`\s+.*\[.*\]:\s+`)

	for scanner.Scan() {
		line := scanner.Text()

		// Filter out unwanted lines
		skip := false
		for _, pattern := range filterPatterns {
			if pattern.MatchString(line) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		// Remove date prefix if it matches today
		line = strings.Replace(line, datePrefix, "", 1)
		// Also try removing with single digit day
		altDatePrefix := time.Now().Format("Jan  2 ")
		line = strings.Replace(line, altDatePrefix, "", 1)

		// Remove home path from log
		line = strings.ReplaceAll(line, homeDir+"/", "")

		// Simplify service name pattern
		line = servicePattern.ReplaceAllString(line, "---")

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Send to client
		if err := conn.WriteMessage(websocket.TextMessage, []byte(line)); err != nil {
			break
		}
	}

	// Wait for command to finish
	cmd.Wait()
}

// StreamDetectionLogs handles GET /ws/logs/detections requests.
// Streams only detection-related log entries.
func (h *Handlers) StreamDetectionLogs(w http.ResponseWriter, r *http.Request) {
	conn, err := logUpgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Could not upgrade connection", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Handle client disconnection
	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				cancel()
				return
			}
		}
	}()

	// Start journalctl process for analysis service only
	cmd := exec.CommandContext(ctx, "journalctl",
		"--no-hostname",
		"-q",
		"-o", "short",
		"-f",
		"-u", "birdnet_analysis",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("Error: Could not start log stream"))
		return
	}

	if err := cmd.Start(); err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("Error: Could not start journalctl"))
		return
	}

	scanner := bufio.NewScanner(stdout)
	detectionPattern := regexp.MustCompile(`detected|confidence|species|analysis`)

	for scanner.Scan() {
		line := scanner.Text()

		// Only send lines that look like detections
		if !detectionPattern.MatchString(strings.ToLower(line)) {
			continue
		}

		if err := conn.WriteMessage(websocket.TextMessage, []byte(line)); err != nil {
			break
		}
	}

	cmd.Wait()
}

// GetRecentLogs handles GET /api/logs/recent requests.
// Returns the most recent log entries without streaming.
func (h *Handlers) GetRecentLogs(w http.ResponseWriter, r *http.Request) {
	// Get number of lines from query parameter, default 100
	lines := r.URL.Query().Get("lines")
	if lines == "" {
		lines = "100"
	}

	cmd := exec.Command("journalctl",
		"--no-hostname",
		"-q",
		"-o", "short",
		"-n", lines,
		"--no-pager",
		"-u", "birdnet_analysis",
		"-u", "birdnet_recording",
	)

	output, err := cmd.Output()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get logs: "+err.Error())
		return
	}

	// Process output similar to streaming
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		homeDir = "/home/birdnet"
	}

	var processedLines []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	filterPatterns := []*regexp.Regexp{
		regexp.MustCompile(`Line`),
		regexp.MustCompile(`find`),
		regexp.MustCompile(`systemd`),
	}

	for scanner.Scan() {
		line := scanner.Text()

		skip := false
		for _, pattern := range filterPatterns {
			if pattern.MatchString(line) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		line = strings.ReplaceAll(line, homeDir+"/", "")

		if strings.TrimSpace(line) != "" {
			processedLines = append(processedLines, line)
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"lines": processedLines,
		"count": len(processedLines),
	})
}
