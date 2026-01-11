package ws

import (
	"encoding/json"
)

// WSMessage is the standard WebSocket message format.
type WSMessage struct {
	Type    string          `json:"type"`
	Channel string          `json:"channel,omitempty"`
	Payload json.RawMessage `json:"payload"`
}

// Part 1 message types
const (
	TypeDetection = "detection"
	TypeStatus    = "status"
	TypeSubscribe = "subscribe"
	TypeUnsubscribe = "unsubscribe"
	TypePing      = "ping"
	TypePong      = "pong"
)

// Part 2 message types (defined now, used later)
const (
	TypeSpectrogramFrame = "spectrogram_frame"
	TypeLLMStream        = "llm_stream"
	TypeVADResult        = "vad_result"
)

// Default channels
const (
	ChannelDetections  = "detections"
	ChannelStatus      = "status"
	ChannelSpectrogram = "spectrogram" // Part 2
	ChannelLLM         = "llm"         // Part 2
)

// DetectionPayload is the payload for detection notifications.
type DetectionPayload struct {
	ID         int64   `json:"id"`
	Date       string  `json:"date"`
	Time       string  `json:"time"`
	SciName    string  `json:"sci_name"`
	ComName    string  `json:"com_name"`
	Confidence float64 `json:"confidence"`
	FileName   string  `json:"file_name"`
}

// StatusPayload is the payload for system status updates.
type StatusPayload struct {
	SystemStatus  string `json:"system_status"`
	MLServiceUp   bool   `json:"ml_service_up"`
	ActiveClients int    `json:"active_clients"`
}

// SubscribePayload is the payload for subscription requests.
type SubscribePayload struct {
	Channels []string `json:"channels"`
}

// NewMessage creates a new WSMessage with the given type and payload.
func NewMessage(msgType string, payload interface{}) (*WSMessage, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &WSMessage{
		Type:    msgType,
		Payload: payloadBytes,
	}, nil
}

// NewChannelMessage creates a new WSMessage for a specific channel.
func NewChannelMessage(msgType string, channel string, payload interface{}) (*WSMessage, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &WSMessage{
		Type:    msgType,
		Channel: channel,
		Payload: payloadBytes,
	}, nil
}

// ParsePayload parses the payload into the given struct.
func (m *WSMessage) ParsePayload(v interface{}) error {
	return json.Unmarshal(m.Payload, v)
}
