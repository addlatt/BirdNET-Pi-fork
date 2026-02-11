import { useState, useEffect, useCallback } from 'preact/hooks';
import type { JSX } from 'preact';
import { useSettings } from '../hooks/useSettings';
import type { ConfigUpdate } from '../types/settings';
import {
  TextInput,
  NumberInput,
  SliderInput,
  SelectInput,
  CheckboxInput,
  FormSection,
  SaveButton,
  AlertMessage,
} from '../components/settings/FormInputs';
import {
  AVAILABLE_LOG_LEVELS,
  AVAILABLE_FULL_DISK_ACTIONS,
  AVAILABLE_FREQSHIFT_TOOLS,
} from '../types/settings';

/**
 * Advanced Settings page - More technical configuration options.
 */
export function AdvancedSettings(): JSX.Element {
  const {
    config,
    ntpEnabled,
    loading,
    saving,
    error,
    saveError,
    validationErrors,
    restartedServices,
    refresh,
    save,
    clearErrors,
  } = useSettings();

  // Local form state
  const [formData, setFormData] = useState<ConfigUpdate>({});
  const [localNtpEnabled, setLocalNtpEnabled] = useState(true);
  const [hasChanges, setHasChanges] = useState(false);
  const [showSuccess, setShowSuccess] = useState(false);

  // Initialize form data when config loads
  useEffect(() => {
    if (config) {
      setFormData({
        // Privacy & Analysis
        privacy_threshold: config.privacy_threshold,
        overlap: config.overlap,
        rare_species_threshold: config.rare_species_threshold,
        raw_spectrogram: config.raw_spectrogram,

        // Disk Management
        full_disk: config.full_disk,
        purge_threshold: config.purge_threshold,
        max_files_species: config.max_files_species,

        // Recording
        rec_card: config.rec_card,
        channels: config.channels,
        rtsp_stream: config.rtsp_stream,
        rtsp_stream_to_livestream: config.rtsp_stream_to_livestream,

        // Passwords
        caddy_pwd: '', // Don't show existing password
        ice_pwd: config.ice_pwd,

        // URLs
        birdnetpi_url: config.birdnetpi_url,
        heartbeat_url: config.heartbeat_url,

        // Frequency Shifting
        activate_freqshift_in_livestream: config.activate_freqshift_in_livestream,
        freqshift_tool: config.freqshift_tool,
        freqshift_hi: config.freqshift_hi,
        freqshift_lo: config.freqshift_lo,
        freqshift_pitch: config.freqshift_pitch,
        freqshift_reconnect_delay: config.freqshift_reconnect_delay,

        // Options
        silence_update_indicator: config.silence_update_indicator,
        automatic_update: config.automatic_update,

        // Logging
        log_level_birdnet_recording_service: config.log_level_birdnet_recording_service,
        log_level_live_audio_stream_service: config.log_level_live_audio_stream_service,
        log_level_spectrogram_viewer_service: config.log_level_spectrogram_viewer_service,
      });
      setLocalNtpEnabled(ntpEnabled);
      setHasChanges(false);
    }
  }, [config, ntpEnabled]);

  // Update a form field
  const updateField = useCallback(<K extends keyof ConfigUpdate>(key: K, value: ConfigUpdate[K]) => {
    setFormData((prev) => ({ ...prev, [key]: value }));
    setHasChanges(true);
    setShowSuccess(false);
  }, []);

  // Handle save
  const handleSave = useCallback(async () => {
    clearErrors();
    setShowSuccess(false);

    // Build update payload
    const update: ConfigUpdate = {
      ...formData,
      // Only include password if changed (non-empty)
      caddy_pwd: formData.caddy_pwd ? formData.caddy_pwd : undefined,
      use_ntp: localNtpEnabled !== ntpEnabled ? localNtpEnabled : undefined,
    };

    const success = await save(update);
    if (success) {
      setHasChanges(false);
      setShowSuccess(true);
      setTimeout(() => setShowSuccess(false), 5000);
    }
  }, [formData, localNtpEnabled, ntpEnabled, save, clearErrors]);

  // Build select options
  const fullDiskOptions = AVAILABLE_FULL_DISK_ACTIONS.map((a) => ({
    value: a,
    label: a === 'purge' ? 'Purge old files' : 'Keep all (stop recording)',
  }));
  const logLevelOptions = AVAILABLE_LOG_LEVELS.map((l) => ({
    value: l,
    label: l.charAt(0).toUpperCase() + l.slice(1),
  }));
  const freqshiftToolOptions = AVAILABLE_FREQSHIFT_TOOLS.map((t) => ({
    value: t,
    label: t.toUpperCase(),
  }));
  const livestreamFreqshiftOptions = [
    { value: '0', label: 'Disabled' },
    { value: '1', label: 'Enabled' },
  ];

  // Get validation error for a field
  const getError = (field: string): string | undefined => {
    const err = validationErrors.find((e) => e.field === field);
    return err?.message;
  };

  if (loading) {
    return (
      <div class="flex items-center justify-center min-h-[400px]">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600" />
      </div>
    );
  }

  if (error) {
    return (
      <div class="p-4">
        <AlertMessage type="error" message={`Failed to load settings: ${error}`} onDismiss={refresh} />
        <button
          onClick={refresh}
          class="px-4 py-2 bg-primary-600 text-white rounded hover:bg-primary-700"
        >
          Retry
        </button>
      </div>
    );
  }

  return (
    <div class="max-w-4xl mx-auto p-4 sm:p-6">
      <div class="mb-6">
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Advanced Settings</h1>
        <p class="text-gray-500 dark:text-gray-400 mt-1">
          Technical configuration for advanced users
        </p>
      </div>

      {/* Status Messages */}
      {showSuccess && (
        <AlertMessage
          type="success"
          message={restartedServices.length > 0
            ? `Settings saved. Restarted: ${restartedServices.join(', ')}`
            : 'Settings saved successfully'}
          onDismiss={() => setShowSuccess(false)}
        />
      )}
      {saveError && (
        <AlertMessage type="error" message={saveError} onDismiss={clearErrors} />
      )}
      {validationErrors.length > 0 && !saveError && (
        <AlertMessage
          type="warning"
          message={`Validation errors: ${validationErrors.map((e) => e.message).join(', ')}`}
          onDismiss={clearErrors}
        />
      )}

      {/* Privacy & Analysis */}
      <FormSection title="Privacy & Analysis" description="Configure privacy filtering and analysis options">
        <SliderInput
          label="Privacy Threshold"
          value={formData.privacy_threshold || 0}
          onChange={(v) => updateField('privacy_threshold', Math.round(v))}
          min={0}
          max={3}
          step={1}
          formatValue={(v) => ['Off', 'Low', 'Medium', 'High'][v] || 'Off'}
          helpText="Filter out recordings with human sounds"
        />
        <SliderInput
          label="Analysis Overlap"
          value={formData.overlap || 0}
          onChange={(v) => updateField('overlap', v)}
          min={0}
          max={2.9}
          step={0.1}
          formatValue={(v) => `${v.toFixed(1)}s`}
          helpText="Overlap between analysis windows (0-2.9 seconds)"
        />
        <NumberInput
          label="Rare Species Threshold"
          value={formData.rare_species_threshold || 30}
          onChange={(v) => updateField('rare_species_threshold', v)}
          min={1}
          helpText="Days since last detection to mark species as rare"
        />
        <CheckboxInput
          label="Raw Spectrogram"
          value={formData.raw_spectrogram || 0}
          onChange={(v) => updateField('raw_spectrogram', v)}
          helpText="Remove axes and labels from spectrograms"
        />
      </FormSection>

      {/* Disk Management */}
      <FormSection title="Disk Management" description="Control how disk space is managed">
        <SelectInput
          label="Full Disk Action"
          value={formData.full_disk || 'purge'}
          onChange={(v) => updateField('full_disk', v)}
          options={fullDiskOptions}
          helpText="What to do when disk is full"
        />
        <SliderInput
          label="Purge Threshold"
          value={formData.purge_threshold || 95}
          onChange={(v) => updateField('purge_threshold', Math.round(v))}
          min={50}
          max={99}
          step={1}
          formatValue={(v) => `${Math.round(v)}%`}
          helpText="Disk usage percentage to trigger purge"
        />
        <NumberInput
          label="Max Files per Species"
          value={formData.max_files_species || 0}
          onChange={(v) => updateField('max_files_species', v)}
          min={0}
          helpText="0 = unlimited"
        />
      </FormSection>

      {/* Recording Hardware */}
      <FormSection title="Recording Hardware" description="Configure audio input settings">
        <TextInput
          label="Recording Device"
          value={formData.rec_card || 'default'}
          onChange={(v) => updateField('rec_card', v)}
          helpText="ALSA device name or 'default' for PulseAudio"
        />
        <NumberInput
          label="Audio Channels"
          value={formData.channels || 2}
          onChange={(v) => updateField('channels', v)}
          min={1}
          max={8}
          helpText="Number of recording channels (1-8)"
        />
        <TextInput
          label="RTSP Stream URL"
          value={formData.rtsp_stream || ''}
          onChange={(v) => updateField('rtsp_stream', v)}
          placeholder="rtsp://..."
          helpText="Optional RTSP stream for recording instead of local mic"
        />
        {formData.rtsp_stream && (
          <SelectInput
            label="RTSP Stream for Livestream"
            value={formData.rtsp_stream_to_livestream || '0'}
            onChange={(v) => updateField('rtsp_stream_to_livestream', v)}
            options={[
              { value: '0', label: 'Stream 0 (First)' },
              { value: '1', label: 'Stream 1 (Second)' },
            ]}
            helpText="Which RTSP stream to use for web livestream"
          />
        )}
      </FormSection>

      {/* Passwords */}
      <FormSection title="Passwords" description="Set access passwords">
        <TextInput
          label="Web Interface Password"
          value={formData.caddy_pwd || ''}
          onChange={(v) => updateField('caddy_pwd', v)}
          type="password"
          placeholder="Leave empty to keep current"
          helpText="Password for HTTP Basic Auth (empty = no auth)"
        />
        <TextInput
          label="Icecast Password"
          value={formData.ice_pwd || ''}
          onChange={(v) => updateField('ice_pwd', v)}
          type="password"
          helpText="Password for livestream access"
        />
      </FormSection>

      {/* URLs */}
      <FormSection title="URLs" description="Configure external URLs">
        <TextInput
          label="BirdNET-Pi Public URL"
          value={formData.birdnetpi_url || ''}
          onChange={(v) => updateField('birdnetpi_url', v)}
          placeholder="https://yoursite.com"
          helpText="Public URL for web hosting (empty for local only)"
        />
        <TextInput
          label="Heartbeat URL"
          value={formData.heartbeat_url || ''}
          onChange={(v) => updateField('heartbeat_url', v)}
          placeholder="https://uptime.yoursite.com/api/push/..."
          helpText="URL to ping after each analysis cycle (health monitoring)"
        />
      </FormSection>

      {/* Frequency Shifting (Accessibility) */}
      <FormSection title="Frequency Shifting" description="Make bird sounds audible to those with hearing loss">
        <SelectInput
          label="Enable in Livestream"
          value={formData.activate_freqshift_in_livestream || '0'}
          onChange={(v) => updateField('activate_freqshift_in_livestream', v)}
          options={livestreamFreqshiftOptions}
          helpText="Apply frequency shift to web livestream"
        />
        <SelectInput
          label="Processing Tool"
          value={formData.freqshift_tool || 'sox'}
          onChange={(v) => updateField('freqshift_tool', v)}
          options={freqshiftToolOptions}
          helpText="SOX or FFmpeg for audio processing"
        />
        {formData.freqshift_tool === 'sox' ? (
          <NumberInput
            label="Pitch Shift (SOX)"
            value={formData.freqshift_pitch || -1500}
            onChange={(v) => updateField('freqshift_pitch', v)}
            min={-3000}
            max={3000}
            helpText="Cents to shift (100ths of semitone, negative = lower)"
          />
        ) : (
          <>
            <NumberInput
              label="High Frequency (FFmpeg)"
              value={formData.freqshift_hi || 6000}
              onChange={(v) => updateField('freqshift_hi', v)}
              min={1000}
              max={20000}
              helpText="Hz"
            />
            <NumberInput
              label="Low Frequency (FFmpeg)"
              value={formData.freqshift_lo || 3000}
              onChange={(v) => updateField('freqshift_lo', v)}
              min={100}
              max={10000}
              helpText="Hz"
            />
          </>
        )}
        <NumberInput
          label="Reconnect Delay"
          value={formData.freqshift_reconnect_delay || 4000}
          onChange={(v) => updateField('freqshift_reconnect_delay', v)}
          min={0}
          helpText="Milliseconds"
        />
      </FormSection>

      {/* Time Settings */}
      <FormSection title="Time Settings" description="Configure system time">
        <CheckboxInput
          label="Use NTP (Network Time)"
          value={localNtpEnabled ? 1 : 0}
          onChange={(v) => { setLocalNtpEnabled(v === 1); setHasChanges(true); }}
          helpText="Automatically sync time from internet"
        />
      </FormSection>

      {/* Options */}
      <FormSection title="Options" description="Miscellaneous settings">
        <CheckboxInput
          label="Hide Update Indicator"
          value={formData.silence_update_indicator || 0}
          onChange={(v) => updateField('silence_update_indicator', v)}
          helpText="Don't show the update notification badge"
        />
        <CheckboxInput
          label="Automatic Updates"
          value={formData.automatic_update || 0}
          onChange={(v) => updateField('automatic_update', v)}
          helpText="Automatically update from GitHub"
        />
      </FormSection>

      {/* Logging */}
      <FormSection title="Service Logging" description="Configure log verbosity">
        <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <SelectInput
            label="Recording Service"
            value={formData.log_level_birdnet_recording_service || 'error'}
            onChange={(v) => updateField('log_level_birdnet_recording_service', v)}
            options={logLevelOptions}
          />
          <SelectInput
            label="Livestream Service"
            value={formData.log_level_live_audio_stream_service || 'error'}
            onChange={(v) => updateField('log_level_live_audio_stream_service', v)}
            options={logLevelOptions}
          />
          <SelectInput
            label="Spectrogram Service"
            value={formData.log_level_spectrogram_viewer_service || 'error'}
            onChange={(v) => updateField('log_level_spectrogram_viewer_service', v)}
            options={logLevelOptions}
          />
        </div>
      </FormSection>

      {/* Save Button */}
      <div class="flex justify-end gap-4 mt-6">
        <button
          type="button"
          onClick={refresh}
          disabled={saving}
          class="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg"
        >
          Reset
        </button>
        <SaveButton onClick={handleSave} saving={saving} disabled={!hasChanges} />
      </div>

      {/* Link to Basic Settings */}
      <div class="mt-8 p-4 bg-gray-50 dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
        <p class="text-gray-600 dark:text-gray-400">
          <a href="/settings" class="text-primary-600 hover:underline">
            Basic Settings
          </a>{' '}
          for common configuration options like location, model, and notifications.
        </p>
      </div>
    </div>
  );
}
