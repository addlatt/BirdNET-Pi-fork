package config

import (
	"bufio"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// INIFile represents an INI-style configuration file.
// It preserves the original file content and comments while allowing updates.
type INIFile struct {
	path     string
	content  string // Original file content
	values   map[string]string
	modified bool
}

// NewINIFile creates a new INI file handler.
func NewINIFile(path string) *INIFile {
	return &INIFile{
		path:   path,
		values: make(map[string]string),
	}
}

// Load reads and parses the INI file.
func (ini *INIFile) Load() error {
	data, err := os.ReadFile(ini.path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	ini.content = string(data)
	ini.values = make(map[string]string)
	ini.modified = false

	scanner := bufio.NewScanner(strings.NewReader(ini.content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key=value
		idx := strings.Index(line, "=")
		if idx == -1 {
			continue
		}

		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])

		// Remove quotes from value
		value = unquote(value)

		ini.values[key] = value
	}

	return scanner.Err()
}

// Get retrieves a value by key.
func (ini *INIFile) Get(key string) (string, bool) {
	val, ok := ini.values[key]
	return val, ok
}

// GetString retrieves a string value with a default.
func (ini *INIFile) GetString(key, defaultVal string) string {
	if val, ok := ini.values[key]; ok {
		return val
	}
	return defaultVal
}

// GetInt retrieves an integer value with a default.
func (ini *INIFile) GetInt(key string, defaultVal int) int {
	if val, ok := ini.values[key]; ok {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

// GetFloat retrieves a float value with a default.
func (ini *INIFile) GetFloat(key string, defaultVal float64) float64 {
	if val, ok := ini.values[key]; ok {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return defaultVal
}

// Set updates a value in the file content using regex replacement.
// This mirrors the PHP approach of using preg_replace.
func (ini *INIFile) Set(key, value string) {
	// Update in-memory map
	ini.values[key] = value
	ini.modified = true

	// Determine if value needs quoting
	quotedValue := value
	if needsQuoting(key, value) {
		quotedValue = fmt.Sprintf(`"%s"`, value)
	}

	// Create regex pattern to match existing key
	pattern := regexp.MustCompile(fmt.Sprintf(`(?m)^%s=.*$`, regexp.QuoteMeta(key)))

	if pattern.MatchString(ini.content) {
		// Replace existing value
		ini.content = pattern.ReplaceAllString(ini.content, fmt.Sprintf("%s=%s", key, quotedValue))
	} else {
		// Append new key=value at end of file
		if !strings.HasSuffix(ini.content, "\n") {
			ini.content += "\n"
		}
		ini.content += fmt.Sprintf("%s=%s\n", key, quotedValue)
	}
}

// SetInt sets an integer value.
func (ini *INIFile) SetInt(key string, value int) {
	ini.Set(key, strconv.Itoa(value))
}

// SetFloat sets a float value.
func (ini *INIFile) SetFloat(key string, value float64) {
	// Use -1 precision to get the minimum digits needed
	ini.Set(key, strconv.FormatFloat(value, 'f', -1, 64))
}

// Save writes the modified content back to the file.
func (ini *INIFile) Save() error {
	if !ini.modified {
		return nil
	}

	err := os.WriteFile(ini.path, []byte(ini.content), 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	ini.modified = false
	return nil
}

// IsModified returns true if the config has unsaved changes.
func (ini *INIFile) IsModified() bool {
	return ini.modified
}

// Path returns the config file path.
func (ini *INIFile) Path() string {
	return ini.path
}

// unquote removes surrounding quotes from a value.
func unquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') ||
			(s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// needsQuoting determines if a value should be quoted in the config file.
// Values that need quoting: strings with spaces, special characters, or certain keys.
func needsQuoting(key, value string) bool {
	// These keys always need quoting (based on PHP behavior)
	quotedKeys := map[string]bool{
		"SITE_NAME":                       true,
		"REC_CARD":                        true,
		"RTSP_STREAM":                     true,
		"RTSP_STREAM_TO_LIVESTREAM":       true,
		"ACTIVATE_FREQSHIFT_IN_LIVESTREAM": true,
		"CADDY_PWD":                       true,
		"APPRISE_NOTIFICATION_TITLE":      true,
		"APPRISE_ONLY_NOTIFY_SPECIES_NAMES":   true,
		"APPRISE_ONLY_NOTIFY_SPECIES_NAMES_2": true,
		"CUSTOM_IMAGE_TITLE":              true,
		"LogLevel_BirdnetRecordingService":   true,
		"LogLevel_LiveAudioStreamService":    true,
		"LogLevel_SpectrogramViewerService":  true,
	}

	if quotedKeys[key] {
		return true
	}

	// Check for spaces or special characters
	if strings.ContainsAny(value, " \t\"'\\") {
		return true
	}

	return false
}

// LoadIntoStruct populates a Config struct from the INI file.
func (ini *INIFile) LoadIntoStruct(cfg *Config) error {
	if err := ini.Load(); err != nil {
		return err
	}

	// Use reflection to map INI values to struct fields
	v := reflect.ValueOf(cfg).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		iniTag := field.Tag.Get("ini")
		if iniTag == "" {
			continue
		}

		iniValue, ok := ini.values[iniTag]
		if !ok {
			continue
		}

		fieldValue := v.Field(i)
		if !fieldValue.CanSet() {
			continue
		}

		switch fieldValue.Kind() {
		case reflect.String:
			fieldValue.SetString(iniValue)
		case reflect.Int, reflect.Int64:
			if intVal, err := strconv.ParseInt(iniValue, 10, 64); err == nil {
				fieldValue.SetInt(intVal)
			}
		case reflect.Float64:
			if floatVal, err := strconv.ParseFloat(iniValue, 64); err == nil {
				fieldValue.SetFloat(floatVal)
			}
		}
	}

	return nil
}

// SaveFromStruct saves a Config struct to the INI file.
func (ini *INIFile) SaveFromStruct(cfg *Config) error {
	// Use reflection to map struct fields to INI values
	v := reflect.ValueOf(cfg).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		iniTag := field.Tag.Get("ini")
		if iniTag == "" {
			continue
		}

		fieldValue := v.Field(i)

		var strValue string
		switch fieldValue.Kind() {
		case reflect.String:
			strValue = fieldValue.String()
		case reflect.Int, reflect.Int64:
			strValue = strconv.FormatInt(fieldValue.Int(), 10)
		case reflect.Float64:
			strValue = strconv.FormatFloat(fieldValue.Float(), 'f', -1, 64)
		default:
			continue
		}

		ini.Set(iniTag, strValue)
	}

	return ini.Save()
}

// ApplyUpdate applies a ConfigUpdate to the INI file, returning changed keys.
func (ini *INIFile) ApplyUpdate(update *ConfigUpdate) ([]string, error) {
	changed := []string{}

	v := reflect.ValueOf(update).Elem()
	t := v.Type()

	// Map JSON field names to INI keys
	jsonToINI := map[string]string{
		"site_name":                  "SITE_NAME",
		"latitude":                   "LATITUDE",
		"longitude":                  "LONGITUDE",
		"model":                      "MODEL",
		"sf_thresh":                  "SF_THRESH",
		"data_model_version":         "DATA_MODEL_VERSION",
		"confidence":                 "CONFIDENCE",
		"sensitivity":                "SENSITIVITY",
		"overlap":                    "OVERLAP",
		"privacy_threshold":          "PRIVACY_THRESHOLD",
		"rec_card":                   "REC_CARD",
		"channels":                   "CHANNELS",
		"recording_length":           "RECORDING_LENGTH",
		"extraction_length":          "EXTRACTION_LENGTH",
		"audiofmt":                   "AUDIOFMT",
		"full_disk":                  "FULL_DISK",
		"purge_threshold":            "PURGE_THRESHOLD",
		"max_files_species":          "MAX_FILES_SPECIES",
		"rtsp_stream":                "RTSP_STREAM",
		"rtsp_stream_to_livestream":  "RTSP_STREAM_TO_LIVESTREAM",
		"activate_freqshift_in_livestream": "ACTIVATE_FREQSHIFT_IN_LIVESTREAM",
		"birdweather_id":             "BIRDWEATHER_ID",
		"apprise_notification_title": "APPRISE_NOTIFICATION_TITLE",
		"apprise_notify_each_detection": "APPRISE_NOTIFY_EACH_DETECTION",
		"apprise_notify_new_species": "APPRISE_NOTIFY_NEW_SPECIES",
		"apprise_notify_new_species_each_day": "APPRISE_NOTIFY_NEW_SPECIES_EACH_DAY",
		"apprise_weekly_report":      "APPRISE_WEEKLY_REPORT",
		"apprise_minimum_seconds_between_notifications_per_species": "APPRISE_MINIMUM_SECONDS_BETWEEN_NOTIFICATIONS_PER_SPECIES",
		"apprise_only_notify_species_names":   "APPRISE_ONLY_NOTIFY_SPECIES_NAMES",
		"apprise_only_notify_species_names_2": "APPRISE_ONLY_NOTIFY_SPECIES_NAMES_2",
		"image_provider":             "IMAGE_PROVIDER",
		"flickr_api_key":             "FLICKR_API_KEY",
		"flickr_filter_email":        "FLICKR_FILTER_EMAIL",
		"color_scheme":               "COLOR_SCHEME",
		"info_site":                  "INFO_SITE",
		"database_lang":              "DATABASE_LANG",
		"caddy_pwd":                  "CADDY_PWD",
		"ice_pwd":                    "ICE_PWD",
		"birdnetpi_url":              "BIRDNETPI_URL",
		"freqshift_tool":             "FREQSHIFT_TOOL",
		"freqshift_hi":               "FREQSHIFT_HI",
		"freqshift_lo":               "FREQSHIFT_LO",
		"freqshift_pitch":            "FREQSHIFT_PITCH",
		"freqshift_reconnect_delay":  "FREQSHIFT_RECONNECT_DELAY",
		"silence_update_indicator":   "SILENCE_UPDATE_INDICATOR",
		"automatic_update":           "AUTOMATIC_UPDATE",
		"raw_spectrogram":            "RAW_SPECTROGRAM",
		"rare_species_threshold":     "RARE_SPECIES_THRESHOLD",
		"heartbeat_url":              "HEARTBEAT_URL",
		"custom_image":               "CUSTOM_IMAGE",
		"custom_image_title":         "CUSTOM_IMAGE_TITLE",
		"log_level_birdnet_recording_service":  "LogLevel_BirdnetRecordingService",
		"log_level_live_audio_stream_service":  "LogLevel_LiveAudioStreamService",
		"log_level_spectrogram_viewer_service": "LogLevel_SpectrogramViewerService",
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Skip nil pointer fields (not set in update)
		if fieldValue.IsNil() {
			continue
		}

		// Get JSON tag name
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" {
			continue
		}
		// Remove ",omitempty" suffix
		if idx := strings.Index(jsonTag, ","); idx != -1 {
			jsonTag = jsonTag[:idx]
		}

		// Skip special fields that aren't INI keys
		if jsonTag == "apprise_config" || jsonTag == "apprise_body" ||
			jsonTag == "timezone" || jsonTag == "manual_date" ||
			jsonTag == "manual_time" || jsonTag == "use_ntp" {
			continue
		}

		iniKey, ok := jsonToINI[jsonTag]
		if !ok {
			continue
		}

		// Get the actual value from the pointer
		elem := fieldValue.Elem()
		var strValue string

		switch elem.Kind() {
		case reflect.String:
			strValue = elem.String()
		case reflect.Int, reflect.Int64:
			strValue = strconv.FormatInt(elem.Int(), 10)
		case reflect.Float64:
			strValue = strconv.FormatFloat(elem.Float(), 'f', -1, 64)
		case reflect.Bool:
			if elem.Bool() {
				strValue = "1"
			} else {
				strValue = "0"
			}
		default:
			continue
		}

		// Check if value actually changed
		oldValue, _ := ini.Get(iniKey)
		if oldValue != strValue {
			ini.Set(iniKey, strValue)
			changed = append(changed, iniKey)
		}
	}

	return changed, nil
}
