// Package config provides configuration management for BirdNET-Pi.
// It handles loading, saving, and validating configuration from INI-style config files.
package config

import (
	"time"
)

// Config represents the complete BirdNET-Pi configuration.
// Field names match the INI file keys for easy mapping.
// All fields include JSON tags for API serialization and ini tags for file parsing.
type Config struct {
	// === Location & Identity ===
	SiteName     string  `json:"site_name" ini:"SITE_NAME"`
	Latitude     float64 `json:"latitude" ini:"LATITUDE"`
	Longitude    float64 `json:"longitude" ini:"LONGITUDE"`
	BirdnetUser  string  `json:"birdnet_user" ini:"BIRDNET_USER"`
	InstallDir   string  `json:"install_dir" ini:"INSTALL_DIR"`

	// === BirdNET Model ===
	Model            string  `json:"model" ini:"MODEL"`
	SFThresh         float64 `json:"sf_thresh" ini:"SF_THRESH"`
	DataModelVersion int     `json:"data_model_version" ini:"DATA_MODEL_VERSION"`

	// === Analysis Parameters ===
	Confidence       float64 `json:"confidence" ini:"CONFIDENCE"`
	Sensitivity      float64 `json:"sensitivity" ini:"SENSITIVITY"`
	Overlap          float64 `json:"overlap" ini:"OVERLAP"`
	PrivacyThreshold int     `json:"privacy_threshold" ini:"PRIVACY_THRESHOLD"`

	// === Recording Settings ===
	RecCard          string `json:"rec_card" ini:"REC_CARD"`
	Channels         int    `json:"channels" ini:"CHANNELS"`
	RecordingLength  int    `json:"recording_length" ini:"RECORDING_LENGTH"`
	ExtractionLength int    `json:"extraction_length" ini:"EXTRACTION_LENGTH"`
	AudioFmt         string `json:"audiofmt" ini:"AUDIOFMT"`

	// === Storage Paths ===
	RecsDir   string `json:"recs_dir" ini:"RECS_DIR"`
	Processed string `json:"processed" ini:"PROCESSED"`
	Extracted string `json:"extracted" ini:"EXTRACTED"`
	IDFile    string `json:"idfile" ini:"IDFILE"`

	// === Disk Management ===
	FullDisk        string `json:"full_disk" ini:"FULL_DISK"`
	PurgeThreshold  int    `json:"purge_threshold" ini:"PURGE_THRESHOLD"`
	MaxFilesSpecies int    `json:"max_files_species" ini:"MAX_FILES_SPECIES"`

	// === RTSP Streaming ===
	RTSPStream                     string `json:"rtsp_stream" ini:"RTSP_STREAM"`
	RTSPStreamToLivestream         string `json:"rtsp_stream_to_livestream" ini:"RTSP_STREAM_TO_LIVESTREAM"`
	ActivateFreqshiftInLivestream  string `json:"activate_freqshift_in_livestream" ini:"ACTIVATE_FREQSHIFT_IN_LIVESTREAM"`

	// === BirdWeather Integration ===
	BirdweatherID string `json:"birdweather_id" ini:"BIRDWEATHER_ID"`

	// === Apprise Notifications ===
	AppriseNotificationTitle                            string `json:"apprise_notification_title" ini:"APPRISE_NOTIFICATION_TITLE"`
	AppriseNotifyEachDetection                          int    `json:"apprise_notify_each_detection" ini:"APPRISE_NOTIFY_EACH_DETECTION"`
	AppriseNotifyNewSpecies                             int    `json:"apprise_notify_new_species" ini:"APPRISE_NOTIFY_NEW_SPECIES"`
	AppriseNotifyNewSpeciesEachDay                      int    `json:"apprise_notify_new_species_each_day" ini:"APPRISE_NOTIFY_NEW_SPECIES_EACH_DAY"`
	AppriseWeeklyReport                                 int    `json:"apprise_weekly_report" ini:"APPRISE_WEEKLY_REPORT"`
	AppriseMinimumSecondsBetweenNotificationsPerSpecies int    `json:"apprise_minimum_seconds_between_notifications_per_species" ini:"APPRISE_MINIMUM_SECONDS_BETWEEN_NOTIFICATIONS_PER_SPECIES"`
	AppriseOnlyNotifySpeciesNames                       string `json:"apprise_only_notify_species_names" ini:"APPRISE_ONLY_NOTIFY_SPECIES_NAMES"`
	AppriseOnlyNotifySpeciesNames2                      string `json:"apprise_only_notify_species_names_2" ini:"APPRISE_ONLY_NOTIFY_SPECIES_NAMES_2"`

	// === Image Provider ===
	ImageProvider     string `json:"image_provider" ini:"IMAGE_PROVIDER"`
	FlickrAPIKey      string `json:"flickr_api_key" ini:"FLICKR_API_KEY"`
	FlickrFilterEmail string `json:"flickr_filter_email" ini:"FLICKR_FILTER_EMAIL"`

	// === UI Display ===
	ColorScheme string `json:"color_scheme" ini:"COLOR_SCHEME"`
	InfoSite    string `json:"info_site" ini:"INFO_SITE"`

	// === Language ===
	DatabaseLang string `json:"database_lang" ini:"DATABASE_LANG"`

	// === Authentication ===
	CaddyPwd string `json:"caddy_pwd,omitempty" ini:"CADDY_PWD"` // omitempty to not expose in GET
	IcePwd   string `json:"ice_pwd,omitempty" ini:"ICE_PWD"`     // omitempty to not expose in GET

	// === Custom URL ===
	BirdnetpiURL string `json:"birdnetpi_url" ini:"BIRDNETPI_URL"`

	// === Frequency Shifting (Accessibility) ===
	FreqshiftTool           string `json:"freqshift_tool" ini:"FREQSHIFT_TOOL"`
	FreqshiftHi             int    `json:"freqshift_hi" ini:"FREQSHIFT_HI"`
	FreqshiftLo             int    `json:"freqshift_lo" ini:"FREQSHIFT_LO"`
	FreqshiftPitch          int    `json:"freqshift_pitch" ini:"FREQSHIFT_PITCH"`
	FreqshiftReconnectDelay int    `json:"freqshift_reconnect_delay" ini:"FREQSHIFT_RECONNECT_DELAY"`

	// === Options ===
	SilenceUpdateIndicator int    `json:"silence_update_indicator" ini:"SILENCE_UPDATE_INDICATOR"`
	AutomaticUpdate        int    `json:"automatic_update" ini:"AUTOMATIC_UPDATE"`
	RawSpectrogram         int    `json:"raw_spectrogram" ini:"RAW_SPECTROGRAM"`
	RareSpeciesThreshold   int    `json:"rare_species_threshold" ini:"RARE_SPECIES_THRESHOLD"`
	HeartbeatURL           string `json:"heartbeat_url" ini:"HEARTBEAT_URL"`

	// === Custom Image ===
	CustomImage      string `json:"custom_image" ini:"CUSTOM_IMAGE"`
	CustomImageTitle string `json:"custom_image_title" ini:"CUSTOM_IMAGE_TITLE"`

	// === Logging ===
	LogLevelBirdnetRecordingService   string `json:"log_level_birdnet_recording_service" ini:"LogLevel_BirdnetRecordingService"`
	LogLevelLiveAudioStreamService    string `json:"log_level_live_audio_stream_service" ini:"LogLevel_LiveAudioStreamService"`
	LogLevelSpectrogramViewerService  string `json:"log_level_spectrogram_viewer_service" ini:"LogLevel_SpectrogramViewerService"`
}

// ConfigUpdate represents a partial config update from the API.
// All fields are pointers so we can distinguish between "not set" and "set to zero value".
type ConfigUpdate struct {
	// === Location & Identity ===
	SiteName  *string  `json:"site_name,omitempty"`
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`

	// === BirdNET Model ===
	Model            *string  `json:"model,omitempty"`
	SFThresh         *float64 `json:"sf_thresh,omitempty"`
	DataModelVersion *int     `json:"data_model_version,omitempty"`

	// === Analysis Parameters ===
	Confidence       *float64 `json:"confidence,omitempty"`
	Sensitivity      *float64 `json:"sensitivity,omitempty"`
	Overlap          *float64 `json:"overlap,omitempty"`
	PrivacyThreshold *int     `json:"privacy_threshold,omitempty"`

	// === Recording Settings ===
	RecCard          *string `json:"rec_card,omitempty"`
	Channels         *int    `json:"channels,omitempty"`
	RecordingLength  *int    `json:"recording_length,omitempty"`
	ExtractionLength *int    `json:"extraction_length,omitempty"`
	AudioFmt         *string `json:"audiofmt,omitempty"`

	// === Disk Management ===
	FullDisk        *string `json:"full_disk,omitempty"`
	PurgeThreshold  *int    `json:"purge_threshold,omitempty"`
	MaxFilesSpecies *int    `json:"max_files_species,omitempty"`

	// === RTSP Streaming ===
	RTSPStream                    *string `json:"rtsp_stream,omitempty"`
	RTSPStreamToLivestream        *string `json:"rtsp_stream_to_livestream,omitempty"`
	ActivateFreqshiftInLivestream *string `json:"activate_freqshift_in_livestream,omitempty"`

	// === BirdWeather Integration ===
	BirdweatherID *string `json:"birdweather_id,omitempty"`

	// === Apprise Notifications ===
	AppriseNotificationTitle                            *string `json:"apprise_notification_title,omitempty"`
	AppriseNotifyEachDetection                          *int    `json:"apprise_notify_each_detection,omitempty"`
	AppriseNotifyNewSpecies                             *int    `json:"apprise_notify_new_species,omitempty"`
	AppriseNotifyNewSpeciesEachDay                      *int    `json:"apprise_notify_new_species_each_day,omitempty"`
	AppriseWeeklyReport                                 *int    `json:"apprise_weekly_report,omitempty"`
	AppriseMinimumSecondsBetweenNotificationsPerSpecies *int    `json:"apprise_minimum_seconds_between_notifications_per_species,omitempty"`
	AppriseOnlyNotifySpeciesNames                       *string `json:"apprise_only_notify_species_names,omitempty"`
	AppriseOnlyNotifySpeciesNames2                      *string `json:"apprise_only_notify_species_names_2,omitempty"`

	// === Image Provider ===
	ImageProvider     *string `json:"image_provider,omitempty"`
	FlickrAPIKey      *string `json:"flickr_api_key,omitempty"`
	FlickrFilterEmail *string `json:"flickr_filter_email,omitempty"`

	// === UI Display ===
	ColorScheme *string `json:"color_scheme,omitempty"`
	InfoSite    *string `json:"info_site,omitempty"`

	// === Language ===
	DatabaseLang *string `json:"database_lang,omitempty"`

	// === Authentication ===
	CaddyPwd *string `json:"caddy_pwd,omitempty"`
	IcePwd   *string `json:"ice_pwd,omitempty"`

	// === Custom URL ===
	BirdnetpiURL *string `json:"birdnetpi_url,omitempty"`

	// === Frequency Shifting (Accessibility) ===
	FreqshiftTool           *string `json:"freqshift_tool,omitempty"`
	FreqshiftHi             *int    `json:"freqshift_hi,omitempty"`
	FreqshiftLo             *int    `json:"freqshift_lo,omitempty"`
	FreqshiftPitch          *int    `json:"freqshift_pitch,omitempty"`
	FreqshiftReconnectDelay *int    `json:"freqshift_reconnect_delay,omitempty"`

	// === Options ===
	SilenceUpdateIndicator *int    `json:"silence_update_indicator,omitempty"`
	AutomaticUpdate        *int    `json:"automatic_update,omitempty"`
	RawSpectrogram         *int    `json:"raw_spectrogram,omitempty"`
	RareSpeciesThreshold   *int    `json:"rare_species_threshold,omitempty"`
	HeartbeatURL           *string `json:"heartbeat_url,omitempty"`

	// === Custom Image ===
	CustomImage      *string `json:"custom_image,omitempty"`
	CustomImageTitle *string `json:"custom_image_title,omitempty"`

	// === Logging ===
	LogLevelBirdnetRecordingService  *string `json:"log_level_birdnet_recording_service,omitempty"`
	LogLevelLiveAudioStreamService   *string `json:"log_level_live_audio_stream_service,omitempty"`
	LogLevelSpectrogramViewerService *string `json:"log_level_spectrogram_viewer_service,omitempty"`

	// === Special Fields (not in config file) ===
	// Apprise config is stored in a separate file
	AppriseConfig *string `json:"apprise_config,omitempty"`
	// Apprise body is stored in a separate file
	AppriseBody *string `json:"apprise_body,omitempty"`
	// Timezone is handled specially via timedatectl
	Timezone *string `json:"timezone,omitempty"`
	// Date/time for manual setting (when NTP is disabled)
	ManualDate *string `json:"manual_date,omitempty"`
	ManualTime *string `json:"manual_time,omitempty"`
	UseNTP     *bool   `json:"use_ntp,omitempty"`
}

// ServiceStatus represents the status of a systemd service.
type ServiceStatus struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Status      string `json:"status"` // "active", "inactive", "failed", "stalled"
	Enabled     bool   `json:"enabled"`
	Message     string `json:"message,omitempty"` // Additional info like backlog count
}

// ServiceAction represents an action to perform on a service.
type ServiceAction string

const (
	ServiceActionStart   ServiceAction = "start"
	ServiceActionStop    ServiceAction = "stop"
	ServiceActionRestart ServiceAction = "restart"
	ServiceActionEnable  ServiceAction = "enable"
	ServiceActionDisable ServiceAction = "disable"
)

// ManagedServices defines the list of services that can be controlled.
var ManagedServices = []struct {
	Name        string
	DisplayName string
}{
	{"livestream.service", "Live Audio Stream"},
	{"web_terminal.service", "Web Terminal"},
	{"birdnet_log.service", "BirdNET Log"},
	{"birdnet_analysis.service", "BirdNET Analysis"},
	{"birdnet_stats.service", "Streamlit Statistics"},
	{"birdnet_recording.service", "Recording Service"},
	{"chart_viewer.service", "Chart Viewer"},
	{"spectrogram_viewer.service", "Spectrogram Viewer"},
}

// ValidationError represents a configuration validation error.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ConfigResponse is the API response for GET /api/settings.
type ConfigResponse struct {
	Settings       *Config                `json:"settings"`
	AppriseConfig  string                 `json:"apprise_config"`
	AppriseBody    string                 `json:"apprise_body"`
	Timezone       string                 `json:"timezone"`
	NTPEnabled     bool                   `json:"ntp_enabled"`
	CurrentTime    time.Time              `json:"current_time"`
	Schema         map[string]interface{} `json:"schema,omitempty"`
	AvailableTimezones []string           `json:"available_timezones,omitempty"`
	AvailableLanguages map[string]string  `json:"available_languages,omitempty"`
}

// UpdateResponse is the API response for PUT /api/settings.
type UpdateResponse struct {
	Status           string   `json:"status"`
	RestartedServices []string `json:"restarted_services,omitempty"`
	Errors           []ValidationError `json:"errors,omitempty"`
	Message          string   `json:"message,omitempty"`
}

// ServicesResponse is the API response for GET /api/services.
type ServicesResponse struct {
	Services []ServiceStatus `json:"services"`
}

// ServiceActionResponse is the API response for service control actions.
type ServiceActionResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Output  string `json:"output,omitempty"`
}

// AvailableLanguages defines the supported database languages.
// Maps language code to display name.
var AvailableLanguages = map[string]string{
	"af":    "Afrikaans",
	"ar":    "Arabic",
	"ca":    "Catalan",
	"cs":    "Czech",
	"zh_CN": "Chinese (simplified)",
	"zh_TW": "Chinese (traditional)",
	"hr":    "Croatian",
	"da":    "Danish",
	"nl":    "Dutch",
	"en":    "English",
	"et":    "Estonian",
	"fi":    "Finnish",
	"fr":    "French",
	"de":    "German",
	"hu":    "Hungarian",
	"is":    "Icelandic",
	"id":    "Indonesia",
	"it":    "Italian",
	"ja":    "Japanese",
	"ko":    "Korean",
	"lv":    "Latvian",
	"lt":    "Lithuania",
	"no":    "Norwegian",
	"pl":    "Polish",
	"pt":    "Portuguese",
	"ro":    "Romanian",
	"ru":    "Russian",
	"sr":    "Serbian",
	"sk":    "Slovak",
	"sl":    "Slovenian",
	"es":    "Spanish",
	"sv":    "Swedish",
	"th":    "Thai",
	"tr":    "Turkish",
	"uk":    "Ukrainian",
	"vi":    "Vietnamese",
}

// AvailableModels defines the available BirdNET models.
var AvailableModels = []string{
	"BirdNET_GLOBAL_6K_V2.4_Model_FP16",
	"BirdNET_6K_GLOBAL_MODEL",
}

// AvailableAudioFormats defines the available audio extraction formats.
var AvailableAudioFormats = []string{
	"mp3", "wav", "flac", "ogg", "opus",
}

// AvailableLogLevels defines the available log levels.
var AvailableLogLevels = []string{
	"debug", "info", "warning", "error",
}
