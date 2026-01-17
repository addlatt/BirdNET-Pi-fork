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
  TextAreaInput,
  FormSection,
  SaveButton,
  AlertMessage,
} from '../components/settings/FormInputs';
import {
  AVAILABLE_MODELS,
  AVAILABLE_AUDIO_FORMATS,
  AVAILABLE_COLOR_SCHEMES,
  AVAILABLE_INFO_SITES,
  AVAILABLE_IMAGE_PROVIDERS,
} from '../types/settings';

/**
 * Settings page - Basic configuration options.
 * Mirrors the functionality of the PHP config.php page.
 */
export function Settings(): JSX.Element {
  const {
    config,
    appriseConfig,
    appriseBody,
    timezone,
    availableTimezones,
    availableLanguages,
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

  // Local form state (copy of config for editing)
  const [formData, setFormData] = useState<ConfigUpdate>({});
  const [localAppriseConfig, setLocalAppriseConfig] = useState('');
  const [localAppriseBody, setLocalAppriseBody] = useState('');
  const [localTimezone, setLocalTimezone] = useState('');
  const [hasChanges, setHasChanges] = useState(false);
  const [showSuccess, setShowSuccess] = useState(false);

  // Initialize form data when config loads
  useEffect(() => {
    if (config) {
      setFormData({
        site_name: config.site_name,
        latitude: config.latitude,
        longitude: config.longitude,
        model: config.model,
        sf_thresh: config.sf_thresh,
        confidence: config.confidence,
        sensitivity: config.sensitivity,
        recording_length: config.recording_length,
        extraction_length: config.extraction_length,
        audiofmt: config.audiofmt,
        birdweather_id: config.birdweather_id,
        apprise_notification_title: config.apprise_notification_title,
        apprise_notify_each_detection: config.apprise_notify_each_detection,
        apprise_notify_new_species: config.apprise_notify_new_species,
        apprise_notify_new_species_each_day: config.apprise_notify_new_species_each_day,
        apprise_weekly_report: config.apprise_weekly_report,
        image_provider: config.image_provider,
        flickr_api_key: config.flickr_api_key,
        color_scheme: config.color_scheme,
        info_site: config.info_site,
        database_lang: config.database_lang,
      });
      setLocalAppriseConfig(appriseConfig);
      setLocalAppriseBody(appriseBody);
      setLocalTimezone(timezone);
      setHasChanges(false);
    }
  }, [config, appriseConfig, appriseBody, timezone]);

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
      apprise_config: localAppriseConfig !== appriseConfig ? localAppriseConfig : undefined,
      apprise_body: localAppriseBody !== appriseBody ? localAppriseBody : undefined,
      timezone: localTimezone !== timezone ? localTimezone : undefined,
    };

    const success = await save(update);
    if (success) {
      setHasChanges(false);
      setShowSuccess(true);
      setTimeout(() => setShowSuccess(false), 5000);
    }
  }, [formData, localAppriseConfig, localAppriseBody, localTimezone, appriseConfig, appriseBody, timezone, save, clearErrors]);

  // Build select options
  const modelOptions = AVAILABLE_MODELS.map((m) => ({ value: m, label: m }));
  const formatOptions = AVAILABLE_AUDIO_FORMATS.map((f) => ({ value: f, label: f.toUpperCase() }));
  const colorOptions = AVAILABLE_COLOR_SCHEMES.map((c) => ({ value: c, label: c.charAt(0).toUpperCase() + c.slice(1) }));
  const infoSiteOptions = AVAILABLE_INFO_SITES.map((s) => ({ value: s, label: 'All About Birds' }));
  const imageProviderOptions = AVAILABLE_IMAGE_PROVIDERS.map((p) => ({
    value: p,
    label: p === '' ? 'None' : p.charAt(0) + p.slice(1).toLowerCase(),
  }));
  const languageOptions = Object.entries(availableLanguages).map(([code, name]) => ({
    value: code,
    label: name,
  }));
  const timezoneOptions = availableTimezones.map((tz) => ({ value: tz, label: tz }));

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
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Settings</h1>
        <p class="text-gray-500 dark:text-gray-400 mt-1">
          Configure your BirdNET-Pi installation
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

      {/* Location & Identity */}
      <FormSection title="Location & Identity" description="Configure your site name and location for species filtering">
        <TextInput
          label="Site Name"
          value={formData.site_name || ''}
          onChange={(v) => updateField('site_name', v)}
          placeholder="My Bird Station"
          helpText="Displayed in the page header"
          error={getError('SITE_NAME')}
        />
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <NumberInput
            label="Latitude"
            value={formData.latitude || 0}
            onChange={(v) => updateField('latitude', v)}
            min={-90}
            max={90}
            step={0.0001}
            helpText="-90 to 90"
            error={getError('LATITUDE')}
            required
          />
          <NumberInput
            label="Longitude"
            value={formData.longitude || 0}
            onChange={(v) => updateField('longitude', v)}
            min={-180}
            max={180}
            step={0.0001}
            helpText="-180 to 180"
            error={getError('LONGITUDE')}
            required
          />
        </div>
        <SelectInput
          label="Timezone"
          value={localTimezone}
          onChange={(v) => { setLocalTimezone(v); setHasChanges(true); }}
          options={timezoneOptions}
          helpText="System timezone for scheduling"
        />
      </FormSection>

      {/* BirdNET Model */}
      <FormSection title="BirdNET Model" description="Configure the AI model and detection parameters">
        <SelectInput
          label="Model"
          value={formData.model || ''}
          onChange={(v) => updateField('model', v)}
          options={modelOptions}
          helpText="BirdNET model used for species detection"
          error={getError('MODEL')}
        />
        <SliderInput
          label="Species Filter Threshold"
          value={formData.sf_thresh || 0.03}
          onChange={(v) => updateField('sf_thresh', v)}
          min={0}
          max={1}
          step={0.01}
          formatValue={(v) => v.toFixed(2)}
          helpText="Filter out species unlikely in your area"
        />
      </FormSection>

      {/* Analysis Parameters */}
      <FormSection title="Analysis Parameters" description="Adjust detection sensitivity and confidence levels">
        <SliderInput
          label="Minimum Confidence"
          value={formData.confidence || 0.7}
          onChange={(v) => updateField('confidence', v)}
          min={0.01}
          max={0.99}
          step={0.01}
          formatValue={(v) => `${Math.round(v * 100)}%`}
          helpText="Minimum confidence level to record a detection"
        />
        <SliderInput
          label="Sensitivity"
          value={formData.sensitivity || 1.25}
          onChange={(v) => updateField('sensitivity', v)}
          min={0.5}
          max={1.5}
          step={0.05}
          formatValue={(v) => v.toFixed(2)}
          helpText="Higher = more sensitive, may increase false positives"
        />
      </FormSection>

      {/* Recording Settings */}
      <FormSection title="Recording Settings" description="Configure audio recording and extraction">
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <NumberInput
            label="Recording Length"
            value={formData.recording_length || 15}
            onChange={(v) => updateField('recording_length', v)}
            min={3}
            max={60}
            helpText="Seconds per chunk (3-60)"
            error={getError('RECORDING_LENGTH')}
          />
          <NumberInput
            label="Extraction Length"
            value={formData.extraction_length || 6}
            onChange={(v) => updateField('extraction_length', v)}
            min={1}
            max={30}
            helpText="Seconds to extract (1-30)"
            error={getError('EXTRACTION_LENGTH')}
          />
        </div>
        <SelectInput
          label="Audio Format"
          value={formData.audiofmt || 'mp3'}
          onChange={(v) => updateField('audiofmt', v)}
          options={formatOptions}
          helpText="Format for extracted audio clips"
        />
      </FormSection>

      {/* Integrations */}
      <FormSection title="Integrations" description="BirdWeather and notification settings">
        <TextInput
          label="BirdWeather Station ID"
          value={formData.birdweather_id || ''}
          onChange={(v) => updateField('birdweather_id', v)}
          placeholder="Leave empty to disable"
          helpText="Your BirdWeather station ID for automatic uploads"
        />
      </FormSection>

      {/* Notifications */}
      <FormSection title="Notifications" description="Configure Apprise notification settings">
        <TextInput
          label="Notification Title"
          value={formData.apprise_notification_title || ''}
          onChange={(v) => updateField('apprise_notification_title', v)}
          placeholder="New BirdNET-Pi Detection"
          helpText="Title shown in notifications"
        />
        <CheckboxInput
          label="Notify on each detection"
          value={formData.apprise_notify_each_detection || 0}
          onChange={(v) => updateField('apprise_notify_each_detection', v)}
          helpText="Send a notification for every detection"
        />
        <CheckboxInput
          label="Notify on new species (weekly)"
          value={formData.apprise_notify_new_species || 0}
          onChange={(v) => updateField('apprise_notify_new_species', v)}
          helpText="Notify when a species not seen this week is detected"
        />
        <CheckboxInput
          label="Notify on first detection each day"
          value={formData.apprise_notify_new_species_each_day || 0}
          onChange={(v) => updateField('apprise_notify_new_species_each_day', v)}
          helpText="Notify on first detection of each species per day"
        />
        <CheckboxInput
          label="Send weekly report"
          value={formData.apprise_weekly_report ?? 1}
          onChange={(v) => updateField('apprise_weekly_report', v)}
          helpText="Send a weekly summary of detections"
        />
        <TextAreaInput
          label="Apprise URLs"
          value={localAppriseConfig}
          onChange={(v) => { setLocalAppriseConfig(v); setHasChanges(true); }}
          placeholder="One URL per line..."
          rows={4}
          helpText="Notification service URLs (one per line)"
        />
        <TextAreaInput
          label="Notification Body Template"
          value={localAppriseBody}
          onChange={(v) => { setLocalAppriseBody(v); setHasChanges(true); }}
          placeholder="$comname detected at $time with confidence $confidence..."
          rows={4}
          helpText="Variables: $comname, $sciname, $time, $date, $confidence, $listenurl"
        />
      </FormSection>

      {/* Display Options */}
      <FormSection title="Display Options" description="Customize the user interface">
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <SelectInput
            label="Color Scheme"
            value={formData.color_scheme || 'light'}
            onChange={(v) => updateField('color_scheme', v)}
            options={colorOptions}
          />
          <SelectInput
            label="Species Info Site"
            value={formData.info_site || 'ALLABOUTBIRDS'}
            onChange={(v) => updateField('info_site', v)}
            options={infoSiteOptions}
            helpText="Site for species information links"
          />
        </div>
        <SelectInput
          label="Image Provider"
          value={formData.image_provider || 'WIKIPEDIA'}
          onChange={(v) => updateField('image_provider', v)}
          options={imageProviderOptions}
          helpText="Source for bird images"
        />
        {formData.image_provider === 'FLICKR' && (
          <TextInput
            label="Flickr API Key"
            value={formData.flickr_api_key || ''}
            onChange={(v) => updateField('flickr_api_key', v)}
            helpText="Required for Flickr image provider"
          />
        )}
        <SelectInput
          label="Database Language"
          value={formData.database_lang || 'en'}
          onChange={(v) => updateField('database_lang', v)}
          options={languageOptions}
          helpText="Language for species names"
        />
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

      {/* Link to Advanced Settings */}
      <div class="mt-8 p-4 bg-gray-50 dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
        <p class="text-gray-600 dark:text-gray-400">
          Need more options?{' '}
          <a href="/advanced-settings" class="text-primary-600 hover:underline">
            Advanced Settings
          </a>{' '}
          includes privacy controls, disk management, passwords, and service configuration.
        </p>
      </div>
    </div>
  );
}
