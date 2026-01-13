package config

import (
	"fmt"
	"strings"
)

// FieldType represents the type of a configuration field.
type FieldType string

const (
	FieldTypeString  FieldType = "string"
	FieldTypeNumber  FieldType = "number"
	FieldTypeInteger FieldType = "integer"
	FieldTypeBoolean FieldType = "boolean"
)

// FieldSchema defines validation rules for a configuration field.
type FieldSchema struct {
	Type        FieldType     `json:"type"`
	Description string        `json:"description,omitempty"`
	Default     interface{}   `json:"default,omitempty"`
	Minimum     *float64      `json:"minimum,omitempty"`
	Maximum     *float64      `json:"maximum,omitempty"`
	Enum        []interface{} `json:"enum,omitempty"`
	Required    bool          `json:"required,omitempty"`
}

// Schema contains all field schemas for the configuration.
type Schema struct {
	Fields   map[string]FieldSchema `json:"properties"`
	Required []string               `json:"required"`
}

// ptr returns a pointer to the given float64.
func ptr(f float64) *float64 {
	return &f
}

// NewSchema creates a new Schema with all BirdNET-Pi configuration field definitions.
// This matches the JSON schema from data/config_schema.json.
func NewSchema() *Schema {
	return &Schema{
		Required: []string{"LATITUDE", "LONGITUDE", "BIRDNET_USER"},
		Fields: map[string]FieldSchema{
			// === Location & Identity ===
			"SITE_NAME": {
				Type:        FieldTypeString,
				Description: "Site title for banner display",
				Default:     "",
			},
			"LATITUDE": {
				Type:        FieldTypeNumber,
				Description: "Latitude for species filtering (4 decimal places)",
				Minimum:     ptr(-90),
				Maximum:     ptr(90),
				Required:    true,
			},
			"LONGITUDE": {
				Type:        FieldTypeNumber,
				Description: "Longitude for species filtering (4 decimal places)",
				Minimum:     ptr(-180),
				Maximum:     ptr(180),
				Required:    true,
			},
			"BIRDNET_USER": {
				Type:        FieldTypeString,
				Description: "System user running BirdNET-Pi",
				Required:    true,
			},
			"INSTALL_DIR": {
				Type:        FieldTypeString,
				Description: "BirdNET-Pi installation directory",
				Default:     "",
			},

			// === BirdNET Model ===
			"MODEL": {
				Type:        FieldTypeString,
				Description: "BirdNET model used for detection",
				Default:     "BirdNET_GLOBAL_6K_V2.4_Model_FP16",
				Enum:        []interface{}{"BirdNET_GLOBAL_6K_V2.4_Model_FP16", "BirdNET_6K_GLOBAL_MODEL"},
			},
			"SF_THRESH": {
				Type:        FieldTypeNumber,
				Description: "Species filter threshold",
				Default:     0.03,
				Minimum:     ptr(0.0),
				Maximum:     ptr(1.0),
			},
			"DATA_MODEL_VERSION": {
				Type:        FieldTypeInteger,
				Description: "Model data version marker",
				Default:     1,
				Minimum:     ptr(1),
			},

			// === Analysis Parameters ===
			"CONFIDENCE": {
				Type:        FieldTypeNumber,
				Description: "Minimum confidence level for detections",
				Default:     0.7,
				Minimum:     ptr(0.01),
				Maximum:     ptr(0.99),
			},
			"SENSITIVITY": {
				Type:        FieldTypeNumber,
				Description: "Detection sensitivity",
				Default:     1.25,
				Minimum:     ptr(0.5),
				Maximum:     ptr(1.5),
			},
			"OVERLAP": {
				Type:        FieldTypeNumber,
				Description: "Analysis overlap in seconds",
				Default:     0.0,
				Minimum:     ptr(0.0),
				Maximum:     ptr(2.9),
			},
			"PRIVACY_THRESHOLD": {
				Type:        FieldTypeInteger,
				Description: "Human sound sensitivity level (0=off, 1-3=increasing sensitivity)",
				Default:     0,
				Minimum:     ptr(0),
				Maximum:     ptr(3),
			},

			// === Recording Settings ===
			"REC_CARD": {
				Type:        FieldTypeString,
				Description: "Sound card device (use 'default' for PulseAudio)",
				Default:     "default",
			},
			"CHANNELS": {
				Type:        FieldTypeInteger,
				Description: "Audio recording channels",
				Default:     2,
				Minimum:     ptr(1),
				Maximum:     ptr(8),
			},
			"RECORDING_LENGTH": {
				Type:        FieldTypeInteger,
				Description: "Recording chunk duration in seconds",
				Default:     15,
				Minimum:     ptr(3),
				Maximum:     ptr(60),
			},
			"EXTRACTION_LENGTH": {
				Type:        FieldTypeInteger,
				Description: "Extracted clip duration in seconds",
				Default:     6,
				Minimum:     ptr(1),
				Maximum:     ptr(30),
			},
			"AUDIOFMT": {
				Type:        FieldTypeString,
				Description: "Audio format for extracted clips",
				Default:     "mp3",
				Enum:        []interface{}{"mp3", "wav", "flac", "ogg", "opus"},
			},

			// === Storage Paths ===
			"RECS_DIR": {
				Type:        FieldTypeString,
				Description: "Base directory for recordings",
				Default:     "",
			},
			"PROCESSED": {
				Type:        FieldTypeString,
				Description: "Directory for processed recordings",
				Default:     "",
			},
			"EXTRACTED": {
				Type:        FieldTypeString,
				Description: "Directory for extracted audio clips",
				Default:     "",
			},
			"IDFILE": {
				Type:        FieldTypeString,
				Description: "Path to identification tracking file",
				Default:     "",
			},

			// === Disk Management ===
			"FULL_DISK": {
				Type:        FieldTypeString,
				Description: "Action when disk is full",
				Default:     "purge",
				Enum:        []interface{}{"purge", "keep"},
			},
			"PURGE_THRESHOLD": {
				Type:        FieldTypeInteger,
				Description: "Disk usage percentage to trigger purge",
				Default:     95,
				Minimum:     ptr(1),
				Maximum:     ptr(100),
			},
			"MAX_FILES_SPECIES": {
				Type:        FieldTypeInteger,
				Description: "Maximum files per species (0 = unlimited)",
				Default:     0,
				Minimum:     ptr(0),
			},

			// === RTSP Streaming ===
			"RTSP_STREAM": {
				Type:        FieldTypeString,
				Description: "RTSP stream URL (empty for direct recording)",
				Default:     "",
			},
			"RTSP_STREAM_TO_LIVESTREAM": {
				Type:        FieldTypeString,
				Description: "Stream index for livestream (0 = first stream)",
				Default:     "0",
			},
			"ACTIVATE_FREQSHIFT_IN_LIVESTREAM": {
				Type:        FieldTypeString,
				Description: "Enable frequency shifting in livestream (0=off, 1=on)",
				Default:     "0",
				Enum:        []interface{}{"0", "1"},
			},

			// === BirdWeather Integration ===
			"BIRDWEATHER_ID": {
				Type:        FieldTypeString,
				Description: "BirdWeather station ID for reporting",
				Default:     "",
			},

			// === Apprise Notifications ===
			"APPRISE_NOTIFICATION_TITLE": {
				Type:        FieldTypeString,
				Description: "Title for Apprise notifications",
				Default:     "New BirdNET-Pi Detection",
			},
			"APPRISE_NOTIFY_EACH_DETECTION": {
				Type:        FieldTypeInteger,
				Description: "Notify on every detection (0=off, 1=on)",
				Default:     0,
				Enum:        []interface{}{0, 1},
			},
			"APPRISE_NOTIFY_NEW_SPECIES": {
				Type:        FieldTypeInteger,
				Description: "Notify on new species (weekly) (0=off, 1=on)",
				Default:     0,
				Enum:        []interface{}{0, 1},
			},
			"APPRISE_NOTIFY_NEW_SPECIES_EACH_DAY": {
				Type:        FieldTypeInteger,
				Description: "Notify on first detection each day (0=off, 1=on)",
				Default:     0,
				Enum:        []interface{}{0, 1},
			},
			"APPRISE_WEEKLY_REPORT": {
				Type:        FieldTypeInteger,
				Description: "Send weekly reports (0=off, 1=on)",
				Default:     1,
				Enum:        []interface{}{0, 1},
			},
			"APPRISE_MINIMUM_SECONDS_BETWEEN_NOTIFICATIONS_PER_SPECIES": {
				Type:        FieldTypeInteger,
				Description: "Rate limit seconds between notifications per species",
				Default:     0,
				Minimum:     ptr(0),
			},
			"APPRISE_ONLY_NOTIFY_SPECIES_NAMES": {
				Type:        FieldTypeString,
				Description: "Comma-separated species names to exclude from notifications",
				Default:     "",
			},
			"APPRISE_ONLY_NOTIFY_SPECIES_NAMES_2": {
				Type:        FieldTypeString,
				Description: "Comma-separated species names to include in notifications",
				Default:     "",
			},

			// === Image Provider ===
			"IMAGE_PROVIDER": {
				Type:        FieldTypeString,
				Description: "Source for bird images",
				Default:     "WIKIPEDIA",
				Enum:        []interface{}{"WIKIPEDIA", "FLICKR", ""},
			},
			"FLICKR_API_KEY": {
				Type:        FieldTypeString,
				Description: "Flickr API key for image retrieval",
				Default:     "",
			},
			"FLICKR_FILTER_EMAIL": {
				Type:        FieldTypeString,
				Description: "Filter Flickr images by user email",
				Default:     "",
			},

			// === UI Display ===
			"COLOR_SCHEME": {
				Type:        FieldTypeString,
				Description: "UI color theme",
				Default:     "light",
				Enum:        []interface{}{"light", "dark"},
			},
			"INFO_SITE": {
				Type:        FieldTypeString,
				Description: "Site for species information links",
				Default:     "ALLABOUTBIRDS",
				Enum:        []interface{}{"ALLABOUTBIRDS", "EBIRD"},
			},

			// === Language ===
			"DATABASE_LANG": {
				Type:        FieldTypeString,
				Description: "Language for species database",
				Default:     "en",
			},

			// === Authentication ===
			"CADDY_PWD": {
				Type:        FieldTypeString,
				Description: "Password for web interface authentication",
				Default:     "",
			},
			"ICE_PWD": {
				Type:        FieldTypeString,
				Description: "Icecast2 authentication password",
				Default:     "birdnetpi",
			},

			// === Custom URL ===
			"BIRDNETPI_URL": {
				Type:        FieldTypeString,
				Description: "Public URL for web hosting (empty for local only)",
				Default:     "",
			},

			// === Frequency Shifting (Accessibility) ===
			"FREQSHIFT_TOOL": {
				Type:        FieldTypeString,
				Description: "Tool for frequency shifting",
				Default:     "sox",
				Enum:        []interface{}{"sox", "ffmpeg"},
			},
			"FREQSHIFT_HI": {
				Type:        FieldTypeInteger,
				Description: "High frequency for ffmpeg shift (Hz)",
				Default:     6000,
				Minimum:     ptr(1000),
				Maximum:     ptr(20000),
			},
			"FREQSHIFT_LO": {
				Type:        FieldTypeInteger,
				Description: "Low frequency for ffmpeg shift (Hz)",
				Default:     3000,
				Minimum:     ptr(100),
				Maximum:     ptr(10000),
			},
			"FREQSHIFT_PITCH": {
				Type:        FieldTypeInteger,
				Description: "Pitch shift for sox (100ths of semitone)",
				Default:     -1500,
				Minimum:     ptr(-3000),
				Maximum:     ptr(3000),
			},
			"FREQSHIFT_RECONNECT_DELAY": {
				Type:        FieldTypeInteger,
				Description: "Reconnect delay for frequency shift (ms)",
				Default:     4000,
				Minimum:     ptr(0),
			},

			// === Options ===
			"SILENCE_UPDATE_INDICATOR": {
				Type:        FieldTypeInteger,
				Description: "Hide update notification badge (0=show, 1=hide)",
				Default:     0,
				Enum:        []interface{}{0, 1},
			},
			"AUTOMATIC_UPDATE": {
				Type:        FieldTypeInteger,
				Description: "Enable automatic updates from GitHub (0=off, 1=on)",
				Default:     0,
				Enum:        []interface{}{0, 1},
			},
			"RAW_SPECTROGRAM": {
				Type:        FieldTypeInteger,
				Description: "Remove axes/labels from spectrograms (0=off, 1=on)",
				Default:     0,
				Enum:        []interface{}{0, 1},
			},
			"RARE_SPECIES_THRESHOLD": {
				Type:        FieldTypeInteger,
				Description: "Days since last detection to mark species as rare",
				Default:     30,
				Minimum:     ptr(1),
			},
			"HEARTBEAT_URL": {
				Type:        FieldTypeString,
				Description: "URL to ping after analysis (health check)",
				Default:     "",
			},

			// === Custom Image ===
			"CUSTOM_IMAGE": {
				Type:        FieldTypeString,
				Description: "Path to custom image for Overview page",
				Default:     "",
			},
			"CUSTOM_IMAGE_TITLE": {
				Type:        FieldTypeString,
				Description: "Title for custom image",
				Default:     "",
			},

			// === Logging ===
			"LogLevel_BirdnetRecordingService": {
				Type:        FieldTypeString,
				Description: "Log level for recording service",
				Default:     "error",
				Enum:        []interface{}{"debug", "info", "warning", "error"},
			},
			"LogLevel_LiveAudioStreamService": {
				Type:        FieldTypeString,
				Description: "Log level for livestream service",
				Default:     "error",
				Enum:        []interface{}{"debug", "info", "warning", "error"},
			},
			"LogLevel_SpectrogramViewerService": {
				Type:        FieldTypeString,
				Description: "Log level for spectrogram service",
				Default:     "error",
				Enum:        []interface{}{"debug", "info", "warning", "error"},
			},
		},
	}
}

// Validate validates a single field value against its schema.
func (s *Schema) Validate(key string, value interface{}) *ValidationError {
	field, ok := s.Fields[key]
	if !ok {
		// Unknown key, allow it
		return nil
	}

	// Type validation
	switch field.Type {
	case FieldTypeNumber:
		num, ok := toFloat64(value)
		if !ok {
			return &ValidationError{Field: key, Message: fmt.Sprintf("%s must be a number", key)}
		}
		if field.Minimum != nil && num < *field.Minimum {
			return &ValidationError{Field: key, Message: fmt.Sprintf("%s must be at least %v", key, *field.Minimum)}
		}
		if field.Maximum != nil && num > *field.Maximum {
			return &ValidationError{Field: key, Message: fmt.Sprintf("%s must be at most %v", key, *field.Maximum)}
		}

	case FieldTypeInteger:
		num, ok := toFloat64(value)
		if !ok {
			return &ValidationError{Field: key, Message: fmt.Sprintf("%s must be an integer", key)}
		}
		if field.Minimum != nil && num < *field.Minimum {
			return &ValidationError{Field: key, Message: fmt.Sprintf("%s must be at least %v", key, *field.Minimum)}
		}
		if field.Maximum != nil && num > *field.Maximum {
			return &ValidationError{Field: key, Message: fmt.Sprintf("%s must be at most %v", key, *field.Maximum)}
		}

	case FieldTypeString:
		str, ok := value.(string)
		if !ok {
			return &ValidationError{Field: key, Message: fmt.Sprintf("%s must be a string", key)}
		}
		// Check enum if present
		if len(field.Enum) > 0 {
			found := false
			for _, e := range field.Enum {
				if enumStr, ok := e.(string); ok && enumStr == str {
					found = true
					break
				}
			}
			if !found {
				allowed := make([]string, len(field.Enum))
				for i, e := range field.Enum {
					allowed[i] = fmt.Sprintf("%v", e)
				}
				return &ValidationError{
					Field:   key,
					Message: fmt.Sprintf("%s must be one of: %s", key, strings.Join(allowed, ", ")),
				}
			}
		}
	}

	// Enum validation for numeric types
	if len(field.Enum) > 0 && (field.Type == FieldTypeInteger || field.Type == FieldTypeNumber) {
		num, _ := toFloat64(value)
		found := false
		for _, e := range field.Enum {
			if enumNum, ok := toFloat64(e); ok && enumNum == num {
				found = true
				break
			}
		}
		if !found {
			allowed := make([]string, len(field.Enum))
			for i, e := range field.Enum {
				allowed[i] = fmt.Sprintf("%v", e)
			}
			return &ValidationError{
				Field:   key,
				Message: fmt.Sprintf("%s must be one of: %s", key, strings.Join(allowed, ", ")),
			}
		}
	}

	return nil
}

// ValidateAll validates all fields in a map of values.
func (s *Schema) ValidateAll(values map[string]interface{}) []ValidationError {
	var errors []ValidationError

	// Check required fields
	for _, reqField := range s.Required {
		if _, ok := values[reqField]; !ok {
			errors = append(errors, ValidationError{
				Field:   reqField,
				Message: fmt.Sprintf("%s is required", reqField),
			})
		}
	}

	// Validate each field
	for key, value := range values {
		if err := s.Validate(key, value); err != nil {
			errors = append(errors, *err)
		}
	}

	return errors
}

// GetDefault returns the default value for a field.
func (s *Schema) GetDefault(key string) interface{} {
	if field, ok := s.Fields[key]; ok {
		return field.Default
	}
	return nil
}

// ToMap converts the schema to a JSON-serializable map.
func (s *Schema) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": s.Fields,
		"required":   s.Required,
	}
}

// toFloat64 converts various numeric types to float64.
func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case int32:
		return float64(val), true
	case string:
		// Allow empty strings for optional numeric fields
		if val == "" {
			return 0, true
		}
		return 0, false
	default:
		return 0, false
	}
}
