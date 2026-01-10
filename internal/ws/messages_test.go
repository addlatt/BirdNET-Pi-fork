package ws

import (
	"encoding/json"
	"testing"
)

func TestNewMessage(t *testing.T) {
	tests := []struct {
		name     string
		msgType  string
		payload  interface{}
		wantErr  bool
	}{
		{
			name:    "simple string payload",
			msgType: TypeStatus,
			payload: "test",
			wantErr: false,
		},
		{
			name:    "struct payload",
			msgType: TypeDetection,
			payload: DetectionPayload{
				ID:         1,
				ComName:    "Test Bird",
				Confidence: 0.9,
			},
			wantErr: false,
		},
		{
			name:    "nil payload",
			msgType: TypePong,
			payload: nil,
			wantErr: false,
		},
		{
			name:    "map payload",
			msgType: TypeStatus,
			payload: map[string]string{"key": "value"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := NewMessage(tt.msgType, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if msg.Type != tt.msgType {
				t.Errorf("msg.Type = %s, want %s", msg.Type, tt.msgType)
			}
		})
	}
}

func TestNewChannelMessage(t *testing.T) {
	msg, err := NewChannelMessage(TypeDetection, ChannelDetections, DetectionPayload{
		ID:      1,
		ComName: "Test Bird",
	})
	if err != nil {
		t.Fatalf("NewChannelMessage() error = %v", err)
	}

	if msg.Type != TypeDetection {
		t.Errorf("msg.Type = %s, want %s", msg.Type, TypeDetection)
	}
	if msg.Channel != ChannelDetections {
		t.Errorf("msg.Channel = %s, want %s", msg.Channel, ChannelDetections)
	}
}

func TestWSMessage_ParsePayload(t *testing.T) {
	// Create a message with DetectionPayload
	original := DetectionPayload{
		ID:         42,
		Date:       "2024-01-15",
		Time:       "10:30:00",
		SciName:    "Turdus migratorius",
		ComName:    "American Robin",
		Confidence: 0.92,
		FileName:   "robin.mp3",
	}

	msg, err := NewMessage(TypeDetection, original)
	if err != nil {
		t.Fatalf("NewMessage() error = %v", err)
	}

	// Parse the payload back
	var parsed DetectionPayload
	if err := msg.ParsePayload(&parsed); err != nil {
		t.Fatalf("ParsePayload() error = %v", err)
	}

	if parsed.ID != original.ID {
		t.Errorf("ID = %d, want %d", parsed.ID, original.ID)
	}
	if parsed.ComName != original.ComName {
		t.Errorf("ComName = %s, want %s", parsed.ComName, original.ComName)
	}
	if parsed.Confidence != original.Confidence {
		t.Errorf("Confidence = %f, want %f", parsed.Confidence, original.Confidence)
	}
}

func TestWSMessage_JSONSerialization(t *testing.T) {
	msg, _ := NewChannelMessage(TypeDetection, ChannelDetections, DetectionPayload{
		ID:      1,
		ComName: "Test",
	})

	// Serialize
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Deserialize
	var parsed WSMessage
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if parsed.Type != msg.Type {
		t.Errorf("Type = %s, want %s", parsed.Type, msg.Type)
	}
	if parsed.Channel != msg.Channel {
		t.Errorf("Channel = %s, want %s", parsed.Channel, msg.Channel)
	}
}

func TestSubscribePayload(t *testing.T) {
	payload := SubscribePayload{
		Channels: []string{ChannelDetections, ChannelStatus},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var parsed SubscribePayload
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(parsed.Channels) != 2 {
		t.Errorf("len(Channels) = %d, want 2", len(parsed.Channels))
	}
}

func TestMessageTypeConstants(t *testing.T) {
	// Ensure message type constants are unique
	types := []string{
		TypeDetection,
		TypeStatus,
		TypeSubscribe,
		TypeUnsubscribe,
		TypePing,
		TypePong,
		TypeSpectrogramFrame,
		TypeLLMStream,
		TypeVADResult,
	}

	seen := make(map[string]bool)
	for _, typ := range types {
		if seen[typ] {
			t.Errorf("duplicate message type: %s", typ)
		}
		seen[typ] = true
	}
}

func TestChannelConstants(t *testing.T) {
	// Ensure channel constants are unique
	channels := []string{
		ChannelDetections,
		ChannelStatus,
		ChannelSpectrogram,
		ChannelLLM,
	}

	seen := make(map[string]bool)
	for _, ch := range channels {
		if seen[ch] {
			t.Errorf("duplicate channel: %s", ch)
		}
		seen[ch] = true
	}
}
