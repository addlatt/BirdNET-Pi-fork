package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/birdnet-pi/birdnet/internal/ws"
)

// DetectionNotification represents a detection notification from the Python ML service.
type DetectionNotification struct {
	ID         int64   `json:"id"`
	Date       string  `json:"date"`
	Time       string  `json:"time"`
	SciName    string  `json:"sci_name"`
	ComName    string  `json:"com_name"`
	Confidence float64 `json:"confidence"`
	FileName   string  `json:"file_name"`
	Lat        *float64 `json:"lat,omitempty"`
	Lon        *float64 `json:"lon,omitempty"`
}

// ReceiveDetection handles POST /internal/detection requests.
// This endpoint receives detection notifications from the Python ML service
// and broadcasts them to connected WebSocket clients.
func (h *Handlers) ReceiveDetection(w http.ResponseWriter, r *http.Request) {
	var notification DetectionNotification

	if err := json.NewDecoder(r.Body).Decode(&notification); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid detection notification")
		return
	}

	// Validate required fields
	if notification.SciName == "" || notification.ComName == "" {
		writeError(w, http.StatusBadRequest, "Missing required fields: sci_name, com_name")
		return
	}

	// Create WebSocket payload
	payload := &ws.DetectionPayload{
		ID:         notification.ID,
		Date:       notification.Date,
		Time:       notification.Time,
		SciName:    notification.SciName,
		ComName:    notification.ComName,
		Confidence: notification.Confidence,
		FileName:   notification.FileName,
	}

	// Broadcast to WebSocket clients
	if err := h.hub.BroadcastDetection(payload); err != nil {
		log.Printf("Failed to broadcast detection: %v", err)
		// Don't fail the request - detection is already in DB
	}

	// Log the detection
	log.Printf("Detection received: %s (%s) - %.2f%% confidence",
		notification.ComName, notification.SciName, notification.Confidence*100)

	// Return success
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"message": "Detection notification received",
	})
}
