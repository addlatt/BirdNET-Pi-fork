package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/birdnet-pi/birdnet/internal/config"
)

// GetSettings handles GET /api/settings requests.
// Returns the current configuration, apprise settings, timezone info, and schema.
func (h *Handlers) GetSettings(w http.ResponseWriter, r *http.Request) {
	if h.configMgr == nil {
		writeError(w, http.StatusServiceUnavailable, "Configuration manager not available")
		return
	}

	// Get current config
	cfg := h.configMgr.Get()
	if cfg == nil {
		writeError(w, http.StatusInternalServerError, "Failed to load configuration")
		return
	}

	// Get apprise config
	appriseConfig, err := h.configMgr.GetAppriseConfig()
	if err != nil {
		appriseConfig = "" // Non-fatal, just use empty string
	}

	// Get apprise body
	appriseBody, err := h.configMgr.GetAppriseBody()
	if err != nil {
		appriseBody = "" // Non-fatal, just use empty string
	}

	// Get timezone info
	timezone, err := h.configMgr.GetTimezone()
	if err != nil {
		timezone = "UTC" // Fallback
	}

	ntpEnabled, err := h.configMgr.IsNTPEnabled()
	if err != nil {
		ntpEnabled = true // Assume NTP is enabled by default
	}

	// Build schema for frontend
	schema := h.configMgr.GetSchema()
	schemaMap := buildSchemaResponse(schema)

	// Build response
	response := config.ConfigResponse{
		Settings:           cfg,
		AppriseConfig:      appriseConfig,
		AppriseBody:        appriseBody,
		Timezone:           timezone,
		NTPEnabled:         ntpEnabled,
		CurrentTime:        time.Now(),
		Schema:             schemaMap,
		AvailableTimezones: h.configMgr.GetAvailableTimezones(),
		AvailableLanguages: config.AvailableLanguages,
	}

	writeJSON(w, http.StatusOK, response)
}

// UpdateSettings handles PUT /api/settings requests.
// Validates and applies configuration changes, restarting services as needed.
func (h *Handlers) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	if h.configMgr == nil {
		writeError(w, http.StatusServiceUnavailable, "Configuration manager not available")
		return
	}

	// Parse request body
	var update config.ConfigUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	// Handle special fields that aren't in the INI file
	var specialErrors []config.ValidationError

	// Handle apprise config
	if update.AppriseConfig != nil {
		if err := h.configMgr.SetAppriseConfig(*update.AppriseConfig); err != nil {
			specialErrors = append(specialErrors, config.ValidationError{
				Field:   "apprise_config",
				Message: "Failed to save apprise config: " + err.Error(),
			})
		}
	}

	// Handle apprise body
	if update.AppriseBody != nil {
		if err := h.configMgr.SetAppriseBody(*update.AppriseBody); err != nil {
			specialErrors = append(specialErrors, config.ValidationError{
				Field:   "apprise_body",
				Message: "Failed to save apprise body: " + err.Error(),
			})
		}
	}

	// Handle timezone
	if update.Timezone != nil {
		if err := h.configMgr.SetTimezone(*update.Timezone); err != nil {
			specialErrors = append(specialErrors, config.ValidationError{
				Field:   "timezone",
				Message: "Failed to set timezone: " + err.Error(),
			})
		}
	}

	// Handle NTP setting
	if update.UseNTP != nil {
		if err := h.configMgr.SetNTP(*update.UseNTP); err != nil {
			specialErrors = append(specialErrors, config.ValidationError{
				Field:   "use_ntp",
				Message: "Failed to set NTP: " + err.Error(),
			})
		}
	}

	// Handle manual date/time (only when NTP is disabled)
	if update.ManualDate != nil && update.ManualTime != nil {
		if err := h.configMgr.SetManualTime(*update.ManualDate, *update.ManualTime); err != nil {
			specialErrors = append(specialErrors, config.ValidationError{
				Field:   "manual_time",
				Message: "Failed to set manual time: " + err.Error(),
			})
		}
	}

	// Build validation map from update for schema validation
	validationMap := buildValidationMap(&update)

	// Validate against schema (use ValidateUpdate for partial updates, not Validate)
	validationErrors := h.configMgr.ValidateUpdate(validationMap)

	// Combine all errors
	allErrors := append(specialErrors, validationErrors...)

	if len(allErrors) > 0 {
		writeJSON(w, http.StatusBadRequest, config.UpdateResponse{
			Status: "error",
			Errors: allErrors,
		})
		return
	}

	// Apply the update to the INI file
	changed, servicesToRestart, err := h.configMgr.Update(&update)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update configuration: "+err.Error())
		return
	}

	// Restart services that need it
	var restartedServices []string
	if len(servicesToRestart) > 0 {
		if err := h.configMgr.RestartServices(servicesToRestart); err != nil {
			// Log but don't fail the request
			writeJSON(w, http.StatusOK, config.UpdateResponse{
				Status:            "partial",
				RestartedServices: []string{},
				Message:           "Configuration saved but failed to restart some services: " + err.Error(),
			})
			return
		}
		restartedServices = servicesToRestart
	}

	// Build success response
	response := config.UpdateResponse{
		Status:            "success",
		RestartedServices: restartedServices,
	}

	if len(changed) == 0 && len(specialErrors) == 0 {
		response.Message = "No changes detected"
	} else {
		response.Message = "Configuration updated successfully"
	}

	writeJSON(w, http.StatusOK, response)
}

// GetSettingsSchema handles GET /api/settings/schema requests.
// Returns only the configuration schema (useful for form building).
func (h *Handlers) GetSettingsSchema(w http.ResponseWriter, r *http.Request) {
	if h.configMgr == nil {
		writeError(w, http.StatusServiceUnavailable, "Configuration manager not available")
		return
	}

	schema := h.configMgr.GetSchema()
	schemaMap := buildSchemaResponse(schema)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"schema":             schemaMap,
		"available_models":   config.AvailableModels,
		"available_formats":  config.AvailableAudioFormats,
		"available_languages": config.AvailableLanguages,
		"available_log_levels": config.AvailableLogLevels,
	})
}

// buildSchemaResponse converts the internal schema to a JSON-friendly format.
func buildSchemaResponse(schema *config.Schema) map[string]interface{} {
	result := make(map[string]interface{})

	fields := make(map[string]interface{})
	for key, field := range schema.Fields {
		fieldMap := map[string]interface{}{
			"type":        field.Type,
			"description": field.Description,
		}

		if field.Minimum != nil {
			fieldMap["min"] = *field.Minimum
		}
		if field.Maximum != nil {
			fieldMap["max"] = *field.Maximum
		}
		if len(field.Enum) > 0 {
			fieldMap["enum"] = field.Enum
		}
		if field.Default != nil {
			fieldMap["default"] = field.Default
		}

		fields[key] = fieldMap
	}

	result["fields"] = fields
	result["required"] = schema.Required

	return result
}

// buildValidationMap extracts non-nil values from ConfigUpdate for validation.
func buildValidationMap(update *config.ConfigUpdate) map[string]interface{} {
	result := make(map[string]interface{})

	// Location & Identity
	if update.SiteName != nil {
		result["SITE_NAME"] = *update.SiteName
	}
	if update.Latitude != nil {
		result["LATITUDE"] = *update.Latitude
	}
	if update.Longitude != nil {
		result["LONGITUDE"] = *update.Longitude
	}

	// BirdNET Model
	if update.Model != nil {
		result["MODEL"] = *update.Model
	}
	if update.SFThresh != nil {
		result["SF_THRESH"] = *update.SFThresh
	}
	if update.DataModelVersion != nil {
		result["DATA_MODEL_VERSION"] = *update.DataModelVersion
	}

	// Analysis Parameters
	if update.Confidence != nil {
		result["CONFIDENCE"] = *update.Confidence
	}
	if update.Sensitivity != nil {
		result["SENSITIVITY"] = *update.Sensitivity
	}
	if update.Overlap != nil {
		result["OVERLAP"] = *update.Overlap
	}
	if update.PrivacyThreshold != nil {
		result["PRIVACY_THRESHOLD"] = *update.PrivacyThreshold
	}

	// Recording Settings
	if update.RecCard != nil {
		result["REC_CARD"] = *update.RecCard
	}
	if update.Channels != nil {
		result["CHANNELS"] = *update.Channels
	}
	if update.RecordingLength != nil {
		result["RECORDING_LENGTH"] = *update.RecordingLength
	}
	if update.ExtractionLength != nil {
		result["EXTRACTION_LENGTH"] = *update.ExtractionLength
	}
	if update.AudioFmt != nil {
		result["AUDIOFMT"] = *update.AudioFmt
	}

	// Disk Management
	if update.FullDisk != nil {
		result["FULL_DISK"] = *update.FullDisk
	}
	if update.PurgeThreshold != nil {
		result["PURGE_THRESHOLD"] = *update.PurgeThreshold
	}
	if update.MaxFilesSpecies != nil {
		result["MAX_FILES_SPECIES"] = *update.MaxFilesSpecies
	}

	// RTSP Streaming
	if update.RTSPStream != nil {
		result["RTSP_STREAM"] = *update.RTSPStream
	}
	if update.RTSPStreamToLivestream != nil {
		result["RTSP_STREAM_TO_LIVESTREAM"] = *update.RTSPStreamToLivestream
	}
	if update.ActivateFreqshiftInLivestream != nil {
		result["ACTIVATE_FREQSHIFT_IN_LIVESTREAM"] = *update.ActivateFreqshiftInLivestream
	}

	// BirdWeather Integration
	if update.BirdweatherID != nil {
		result["BIRDWEATHER_ID"] = *update.BirdweatherID
	}

	// Apprise Notifications
	if update.AppriseNotificationTitle != nil {
		result["APPRISE_NOTIFICATION_TITLE"] = *update.AppriseNotificationTitle
	}
	if update.AppriseNotifyEachDetection != nil {
		result["APPRISE_NOTIFY_EACH_DETECTION"] = *update.AppriseNotifyEachDetection
	}
	if update.AppriseNotifyNewSpecies != nil {
		result["APPRISE_NOTIFY_NEW_SPECIES"] = *update.AppriseNotifyNewSpecies
	}
	if update.AppriseNotifyNewSpeciesEachDay != nil {
		result["APPRISE_NOTIFY_NEW_SPECIES_EACH_DAY"] = *update.AppriseNotifyNewSpeciesEachDay
	}
	if update.AppriseWeeklyReport != nil {
		result["APPRISE_WEEKLY_REPORT"] = *update.AppriseWeeklyReport
	}
	if update.AppriseMinimumSecondsBetweenNotificationsPerSpecies != nil {
		result["APPRISE_MINIMUM_SECONDS_BETWEEN_NOTIFICATIONS_PER_SPECIES"] = *update.AppriseMinimumSecondsBetweenNotificationsPerSpecies
	}
	if update.AppriseOnlyNotifySpeciesNames != nil {
		result["APPRISE_ONLY_NOTIFY_SPECIES_NAMES"] = *update.AppriseOnlyNotifySpeciesNames
	}
	if update.AppriseOnlyNotifySpeciesNames2 != nil {
		result["APPRISE_ONLY_NOTIFY_SPECIES_NAMES_2"] = *update.AppriseOnlyNotifySpeciesNames2
	}

	// Image Provider
	if update.ImageProvider != nil {
		result["IMAGE_PROVIDER"] = *update.ImageProvider
	}
	if update.FlickrAPIKey != nil {
		result["FLICKR_API_KEY"] = *update.FlickrAPIKey
	}
	if update.FlickrFilterEmail != nil {
		result["FLICKR_FILTER_EMAIL"] = *update.FlickrFilterEmail
	}

	// UI Display
	if update.ColorScheme != nil {
		result["COLOR_SCHEME"] = *update.ColorScheme
	}
	if update.InfoSite != nil {
		result["INFO_SITE"] = *update.InfoSite
	}

	// Language
	if update.DatabaseLang != nil {
		result["DATABASE_LANG"] = *update.DatabaseLang
	}

	// Authentication
	if update.CaddyPwd != nil {
		result["CADDY_PWD"] = *update.CaddyPwd
	}
	if update.IcePwd != nil {
		result["ICE_PWD"] = *update.IcePwd
	}

	// Custom URL
	if update.BirdnetpiURL != nil {
		result["BIRDNETPI_URL"] = *update.BirdnetpiURL
	}

	// Frequency Shifting
	if update.FreqshiftTool != nil {
		result["FREQSHIFT_TOOL"] = *update.FreqshiftTool
	}
	if update.FreqshiftHi != nil {
		result["FREQSHIFT_HI"] = *update.FreqshiftHi
	}
	if update.FreqshiftLo != nil {
		result["FREQSHIFT_LO"] = *update.FreqshiftLo
	}
	if update.FreqshiftPitch != nil {
		result["FREQSHIFT_PITCH"] = *update.FreqshiftPitch
	}
	if update.FreqshiftReconnectDelay != nil {
		result["FREQSHIFT_RECONNECT_DELAY"] = *update.FreqshiftReconnectDelay
	}

	// Options
	if update.SilenceUpdateIndicator != nil {
		result["SILENCE_UPDATE_INDICATOR"] = *update.SilenceUpdateIndicator
	}
	if update.AutomaticUpdate != nil {
		result["AUTOMATIC_UPDATE"] = *update.AutomaticUpdate
	}
	if update.RawSpectrogram != nil {
		result["RAW_SPECTROGRAM"] = *update.RawSpectrogram
	}
	if update.RareSpeciesThreshold != nil {
		result["RARE_SPECIES_THRESHOLD"] = *update.RareSpeciesThreshold
	}
	if update.HeartbeatURL != nil {
		result["HEARTBEAT_URL"] = *update.HeartbeatURL
	}

	// Custom Image
	if update.CustomImage != nil {
		result["CUSTOM_IMAGE"] = *update.CustomImage
	}
	if update.CustomImageTitle != nil {
		result["CUSTOM_IMAGE_TITLE"] = *update.CustomImageTitle
	}

	// Logging
	if update.LogLevelBirdnetRecordingService != nil {
		result["LogLevel_BirdnetRecordingService"] = *update.LogLevelBirdnetRecordingService
	}
	if update.LogLevelLiveAudioStreamService != nil {
		result["LogLevel_LiveAudioStreamService"] = *update.LogLevelLiveAudioStreamService
	}
	if update.LogLevelSpectrogramViewerService != nil {
		result["LogLevel_SpectrogramViewerService"] = *update.LogLevelSpectrogramViewerService
	}

	return result
}
