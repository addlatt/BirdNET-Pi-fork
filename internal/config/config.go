package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultConfigPath is the default location of the BirdNET-Pi config file.
	DefaultConfigPath = "/etc/birdnet/birdnet.conf"

	// DefaultAppriseConfigPath is where Apprise notification URLs are stored.
	DefaultAppriseConfigPath = "BirdNET-Pi/apprise.txt"

	// DefaultAppriseBodyPath is where the Apprise notification body template is stored.
	DefaultAppriseBodyPath = "BirdNET-Pi/body.txt"
)

// Manager handles all configuration operations.
type Manager struct {
	mu           sync.RWMutex
	configPath   string
	homeDir      string
	config       *Config
	ini          *INIFile
	schema       *Schema
	lastModified time.Time
}

// NewManager creates a new configuration manager.
func NewManager(configPath, homeDir string) *Manager {
	if configPath == "" {
		configPath = DefaultConfigPath
	}

	return &Manager{
		configPath: configPath,
		homeDir:    homeDir,
		schema:     NewSchema(),
		ini:        NewINIFile(configPath),
	}
}

// Load reads the configuration from disk.
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create a new config with defaults
	cfg := m.newConfigWithDefaults()

	// Load INI file into the config struct
	if err := m.ini.LoadIntoStruct(cfg); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	m.config = cfg

	// Update last modified time
	if info, err := os.Stat(m.configPath); err == nil {
		m.lastModified = info.ModTime()
	}

	return nil
}

// Get returns the current configuration.
// The returned config should be treated as read-only.
func (m *Manager) Get() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.config
}

// GetSchema returns the configuration schema.
func (m *Manager) GetSchema() *Schema {
	return m.schema
}

// Update applies a partial configuration update.
// Returns the list of changed fields and any services that need restarting.
func (m *Manager) Update(update *ConfigUpdate) (changed []string, servicesToRestart []string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Reload the INI file to get the latest state
	if err := m.ini.Load(); err != nil {
		return nil, nil, fmt.Errorf("failed to reload config: %w", err)
	}

	// Apply the update and get changed keys
	changed, err = m.ini.ApplyUpdate(update)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to apply update: %w", err)
	}

	if len(changed) == 0 {
		// No changes, nothing to do
		return nil, nil, nil
	}

	// Save the INI file
	if err := m.ini.Save(); err != nil {
		return nil, nil, fmt.Errorf("failed to save config: %w", err)
	}

	// Determine which services need restarting based on changed fields
	servicesToRestart = m.determineRestarts(changed)

	// Reload the config into memory
	cfg := m.newConfigWithDefaults()
	if err := m.ini.LoadIntoStruct(cfg); err != nil {
		return changed, servicesToRestart, fmt.Errorf("failed to reload config after update: %w", err)
	}
	m.config = cfg

	return changed, servicesToRestart, nil
}

// Validate validates a map of configuration values, including required field checks.
// Use this for validating a complete configuration (e.g., initial setup).
func (m *Manager) Validate(values map[string]interface{}) []ValidationError {
	return m.schema.ValidateAll(values)
}

// ValidateUpdate validates only the fields present in the map.
// Use this for partial updates where we only want to validate changed fields.
func (m *Manager) ValidateUpdate(values map[string]interface{}) []ValidationError {
	return m.schema.ValidateUpdate(values)
}

// GetAppriseConfig reads the Apprise notification configuration.
func (m *Manager) GetAppriseConfig() (string, error) {
	path := filepath.Join(m.homeDir, DefaultAppriseConfigPath)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SetAppriseConfig writes the Apprise notification configuration.
func (m *Manager) SetAppriseConfig(content string) error {
	path := filepath.Join(m.homeDir, DefaultAppriseConfigPath)
	return os.WriteFile(path, []byte(content), 0644)
}

// GetAppriseBody reads the Apprise notification body template.
func (m *Manager) GetAppriseBody() (string, error) {
	path := filepath.Join(m.homeDir, DefaultAppriseBodyPath)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SetAppriseBody writes the Apprise notification body template.
func (m *Manager) SetAppriseBody(content string) error {
	path := filepath.Join(m.homeDir, DefaultAppriseBodyPath)
	return os.WriteFile(path, []byte(content), 0644)
}

// GetTimezone returns the current system timezone.
func (m *Manager) GetTimezone() (string, error) {
	out, err := exec.Command("timedatectl", "show", "--value", "--property=Timezone").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// SetTimezone sets the system timezone.
func (m *Manager) SetTimezone(tz string) error {
	// First validate the timezone
	if !isValidTimezone(tz) {
		return fmt.Errorf("invalid timezone: %s", tz)
	}

	// Set via timedatectl
	if err := exec.Command("sudo", "timedatectl", "set-timezone", tz).Run(); err != nil {
		return fmt.Errorf("failed to set timezone via timedatectl: %w", err)
	}

	// Also update /etc/timezone for compatibility
	if err := exec.Command("sh", "-c", fmt.Sprintf("echo %s | sudo tee /etc/timezone > /dev/null", tz)).Run(); err != nil {
		// Non-fatal, log but continue
		fmt.Fprintf(os.Stderr, "warning: failed to update /etc/timezone: %v\n", err)
	}

	return nil
}

// IsNTPEnabled returns whether NTP is enabled.
func (m *Manager) IsNTPEnabled() (bool, error) {
	out, err := exec.Command("timedatectl", "show", "--value", "--property=NTP").Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) == "yes", nil
}

// SetNTP enables or disables NTP.
func (m *Manager) SetNTP(enabled bool) error {
	val := "false"
	if enabled {
		val = "true"
	}
	return exec.Command("sudo", "timedatectl", "set-ntp", val).Run()
}

// SetManualTime sets the system time manually (only works when NTP is disabled).
func (m *Manager) SetManualTime(dateStr, timeStr string) error {
	// Validate format
	datetime := fmt.Sprintf("%s %s", dateStr, timeStr)
	_, err := time.Parse("2006-01-02 15:04", datetime)
	if err != nil {
		return fmt.Errorf("invalid date/time format: %w", err)
	}

	return exec.Command("sudo", "date", "-s", datetime).Run()
}

// GetAvailableTimezones returns a list of available timezones.
func (m *Manager) GetAvailableTimezones() []string {
	// Use timedatectl to list timezones
	out, err := exec.Command("timedatectl", "list-timezones").Output()
	if err != nil {
		// Fallback to common timezones
		return commonTimezones
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	return lines
}

// RestartServices restarts the specified services.
func (m *Manager) RestartServices(services []string) error {
	for _, svc := range services {
		if err := exec.Command("sudo", "systemctl", "restart", svc).Run(); err != nil {
			return fmt.Errorf("failed to restart %s: %w", svc, err)
		}
	}
	return nil
}

// RestartAllServices runs the main restart script.
func (m *Manager) RestartAllServices() error {
	return exec.Command("sudo", "restart_services.sh").Run()
}

// newConfigWithDefaults creates a new Config with default values from the schema.
func (m *Manager) newConfigWithDefaults() *Config {
	cfg := &Config{}

	// Apply schema defaults
	for key, field := range m.schema.Fields {
		if field.Default == nil {
			continue
		}

		switch key {
		case "SITE_NAME":
			if s, ok := field.Default.(string); ok {
				cfg.SiteName = s
			}
		case "MODEL":
			if s, ok := field.Default.(string); ok {
				cfg.Model = s
			}
		case "SF_THRESH":
			if f, ok := field.Default.(float64); ok {
				cfg.SFThresh = f
			}
		case "DATA_MODEL_VERSION":
			if i, ok := toInt(field.Default); ok {
				cfg.DataModelVersion = i
			}
		case "CONFIDENCE":
			if f, ok := field.Default.(float64); ok {
				cfg.Confidence = f
			}
		case "SENSITIVITY":
			if f, ok := field.Default.(float64); ok {
				cfg.Sensitivity = f
			}
		case "OVERLAP":
			if f, ok := field.Default.(float64); ok {
				cfg.Overlap = f
			}
		case "PRIVACY_THRESHOLD":
			if i, ok := toInt(field.Default); ok {
				cfg.PrivacyThreshold = i
			}
		case "REC_CARD":
			if s, ok := field.Default.(string); ok {
				cfg.RecCard = s
			}
		case "CHANNELS":
			if i, ok := toInt(field.Default); ok {
				cfg.Channels = i
			}
		case "RECORDING_LENGTH":
			if i, ok := toInt(field.Default); ok {
				cfg.RecordingLength = i
			}
		case "EXTRACTION_LENGTH":
			if i, ok := toInt(field.Default); ok {
				cfg.ExtractionLength = i
			}
		case "AUDIOFMT":
			if s, ok := field.Default.(string); ok {
				cfg.AudioFmt = s
			}
		case "FULL_DISK":
			if s, ok := field.Default.(string); ok {
				cfg.FullDisk = s
			}
		case "PURGE_THRESHOLD":
			if i, ok := toInt(field.Default); ok {
				cfg.PurgeThreshold = i
			}
		case "MAX_FILES_SPECIES":
			if i, ok := toInt(field.Default); ok {
				cfg.MaxFilesSpecies = i
			}
		case "RTSP_STREAM_TO_LIVESTREAM":
			if s, ok := field.Default.(string); ok {
				cfg.RTSPStreamToLivestream = s
			}
		case "ACTIVATE_FREQSHIFT_IN_LIVESTREAM":
			if s, ok := field.Default.(string); ok {
				cfg.ActivateFreqshiftInLivestream = s
			}
		case "APPRISE_NOTIFICATION_TITLE":
			if s, ok := field.Default.(string); ok {
				cfg.AppriseNotificationTitle = s
			}
		case "APPRISE_NOTIFY_EACH_DETECTION":
			if i, ok := toInt(field.Default); ok {
				cfg.AppriseNotifyEachDetection = i
			}
		case "APPRISE_NOTIFY_NEW_SPECIES":
			if i, ok := toInt(field.Default); ok {
				cfg.AppriseNotifyNewSpecies = i
			}
		case "APPRISE_NOTIFY_NEW_SPECIES_EACH_DAY":
			if i, ok := toInt(field.Default); ok {
				cfg.AppriseNotifyNewSpeciesEachDay = i
			}
		case "APPRISE_WEEKLY_REPORT":
			if i, ok := toInt(field.Default); ok {
				cfg.AppriseWeeklyReport = i
			}
		case "IMAGE_PROVIDER":
			if s, ok := field.Default.(string); ok {
				cfg.ImageProvider = s
			}
		case "COLOR_SCHEME":
			if s, ok := field.Default.(string); ok {
				cfg.ColorScheme = s
			}
		case "INFO_SITE":
			if s, ok := field.Default.(string); ok {
				cfg.InfoSite = s
			}
		case "DATABASE_LANG":
			if s, ok := field.Default.(string); ok {
				cfg.DatabaseLang = s
			}
		case "ICE_PWD":
			if s, ok := field.Default.(string); ok {
				cfg.IcePwd = s
			}
		case "FREQSHIFT_TOOL":
			if s, ok := field.Default.(string); ok {
				cfg.FreqshiftTool = s
			}
		case "FREQSHIFT_HI":
			if i, ok := toInt(field.Default); ok {
				cfg.FreqshiftHi = i
			}
		case "FREQSHIFT_LO":
			if i, ok := toInt(field.Default); ok {
				cfg.FreqshiftLo = i
			}
		case "FREQSHIFT_PITCH":
			if i, ok := toInt(field.Default); ok {
				cfg.FreqshiftPitch = i
			}
		case "FREQSHIFT_RECONNECT_DELAY":
			if i, ok := toInt(field.Default); ok {
				cfg.FreqshiftReconnectDelay = i
			}
		case "SILENCE_UPDATE_INDICATOR":
			if i, ok := toInt(field.Default); ok {
				cfg.SilenceUpdateIndicator = i
			}
		case "AUTOMATIC_UPDATE":
			if i, ok := toInt(field.Default); ok {
				cfg.AutomaticUpdate = i
			}
		case "RAW_SPECTROGRAM":
			if i, ok := toInt(field.Default); ok {
				cfg.RawSpectrogram = i
			}
		case "RARE_SPECIES_THRESHOLD":
			if i, ok := toInt(field.Default); ok {
				cfg.RareSpeciesThreshold = i
			}
		case "LogLevel_BirdnetRecordingService":
			if s, ok := field.Default.(string); ok {
				cfg.LogLevelBirdnetRecordingService = s
			}
		case "LogLevel_LiveAudioStreamService":
			if s, ok := field.Default.(string); ok {
				cfg.LogLevelLiveAudioStreamService = s
			}
		case "LogLevel_SpectrogramViewerService":
			if s, ok := field.Default.(string); ok {
				cfg.LogLevelSpectrogramViewerService = s
			}
		}
	}

	return cfg
}

// determineRestarts determines which services need to be restarted based on changed fields.
func (m *Manager) determineRestarts(changed []string) []string {
	restarts := make(map[string]bool)

	for _, key := range changed {
		switch key {
		// Livestream-related changes
		case "RTSP_STREAM", "RTSP_STREAM_TO_LIVESTREAM", "ACTIVATE_FREQSHIFT_IN_LIVESTREAM",
			"ICE_PWD", "LogLevel_LiveAudioStreamService":
			restarts["livestream.service"] = true

		// Recording-related changes
		case "REC_CARD", "CHANNELS", "RECORDING_LENGTH", "LogLevel_BirdnetRecordingService":
			restarts["birdnet_recording.service"] = true

		// Analysis-related changes
		case "MODEL", "CONFIDENCE", "SENSITIVITY", "OVERLAP", "PRIVACY_THRESHOLD",
			"SF_THRESH", "DATA_MODEL_VERSION", "DATABASE_LANG":
			restarts["birdnet_analysis.service"] = true

		// Chart viewer changes
		case "SITE_NAME", "COLOR_SCHEME":
			restarts["chart_viewer.service"] = true

		// Spectrogram viewer changes
		case "RAW_SPECTROGRAM", "LogLevel_SpectrogramViewerService":
			restarts["spectrogram_viewer.service"] = true

		// URL changes require Caddy update
		case "BIRDNETPI_URL", "CADDY_PWD":
			// These need special handling via update_caddyfile.sh
			// Don't add to restarts, handle separately
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(restarts))
	for svc := range restarts {
		result = append(result, svc)
	}

	return result
}

// toInt converts an interface to int.
func toInt(v interface{}) (int, bool) {
	switch val := v.(type) {
	case int:
		return val, true
	case int64:
		return int(val), true
	case float64:
		return int(val), true
	default:
		return 0, false
	}
}

// isValidTimezone checks if a timezone string is valid.
func isValidTimezone(tz string) bool {
	// Use timedatectl to validate
	out, err := exec.Command("timedatectl", "list-timezones").Output()
	if err != nil {
		return false
	}

	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) == tz {
			return true
		}
	}

	return false
}

// commonTimezones is a fallback list of common timezones.
var commonTimezones = []string{
	"Africa/Cairo",
	"Africa/Johannesburg",
	"America/Chicago",
	"America/Denver",
	"America/Los_Angeles",
	"America/New_York",
	"America/Sao_Paulo",
	"Asia/Dubai",
	"Asia/Hong_Kong",
	"Asia/Kolkata",
	"Asia/Shanghai",
	"Asia/Singapore",
	"Asia/Tokyo",
	"Australia/Melbourne",
	"Australia/Sydney",
	"Europe/Amsterdam",
	"Europe/Berlin",
	"Europe/London",
	"Europe/Moscow",
	"Europe/Paris",
	"Pacific/Auckland",
	"Pacific/Honolulu",
	"UTC",
}
