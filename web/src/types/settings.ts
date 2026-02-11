/**
 * Settings Types - These mirror the Go API response structs exactly.
 * Any changes to the Go API should be reflected here.
 *
 * Source: internal/api/settings.go, internal/api/services.go, internal/config/types.go
 */

// =============================================================================
// Configuration Types (internal/config/types.go)
// =============================================================================

/** Complete BirdNET-Pi configuration */
export interface Config {
  // === Location & Identity ===
  site_name: string;
  latitude: number;
  longitude: number;
  birdnet_user: string;
  install_dir: string;

  // === BirdNET Model ===
  model: string;
  sf_thresh: number;
  data_model_version: number;

  // === Analysis Parameters ===
  confidence: number;
  sensitivity: number;
  overlap: number;
  privacy_threshold: number;

  // === Recording Settings ===
  rec_card: string;
  channels: number;
  recording_length: number;
  extraction_length: number;
  audiofmt: string;

  // === Storage Paths ===
  recs_dir: string;
  processed: string;
  extracted: string;
  idfile: string;

  // === Disk Management ===
  full_disk: string;
  purge_threshold: number;
  max_files_species: number;

  // === RTSP Streaming ===
  rtsp_stream: string;
  rtsp_stream_to_livestream: string;
  activate_freqshift_in_livestream: string;

  // === BirdWeather Integration ===
  birdweather_id: string;

  // === Apprise Notifications ===
  apprise_notification_title: string;
  apprise_notify_each_detection: number;
  apprise_notify_new_species: number;
  apprise_notify_new_species_each_day: number;
  apprise_weekly_report: number;
  apprise_minimum_seconds_between_notifications_per_species: number;
  apprise_only_notify_species_names: string;
  apprise_only_notify_species_names_2: string;

  // === Image Provider ===
  image_provider: string;
  flickr_api_key: string;
  flickr_filter_email: string;

  // === UI Display ===
  color_scheme: string;
  info_site: string;

  // === Language ===
  database_lang: string;

  // === Authentication (omitted in GET for security) ===
  caddy_pwd?: string;
  ice_pwd?: string;

  // === Custom URL ===
  birdnetpi_url: string;

  // === Frequency Shifting (Accessibility) ===
  freqshift_tool: string;
  freqshift_hi: number;
  freqshift_lo: number;
  freqshift_pitch: number;
  freqshift_reconnect_delay: number;

  // === Options ===
  silence_update_indicator: number;
  automatic_update: number;
  raw_spectrogram: number;
  rare_species_threshold: number;
  heartbeat_url: string;

  // === Custom Image ===
  custom_image: string;
  custom_image_title: string;

  // === Logging ===
  log_level_birdnet_recording_service: string;
  log_level_live_audio_stream_service: string;
  log_level_spectrogram_viewer_service: string;
}

/** Partial config update - all fields optional */
export interface ConfigUpdate {
  // === Location & Identity ===
  site_name?: string;
  latitude?: number;
  longitude?: number;

  // === BirdNET Model ===
  model?: string;
  sf_thresh?: number;
  data_model_version?: number;

  // === Analysis Parameters ===
  confidence?: number;
  sensitivity?: number;
  overlap?: number;
  privacy_threshold?: number;

  // === Recording Settings ===
  rec_card?: string;
  channels?: number;
  recording_length?: number;
  extraction_length?: number;
  audiofmt?: string;

  // === Disk Management ===
  full_disk?: string;
  purge_threshold?: number;
  max_files_species?: number;

  // === RTSP Streaming ===
  rtsp_stream?: string;
  rtsp_stream_to_livestream?: string;
  activate_freqshift_in_livestream?: string;

  // === BirdWeather Integration ===
  birdweather_id?: string;

  // === Apprise Notifications ===
  apprise_notification_title?: string;
  apprise_notify_each_detection?: number;
  apprise_notify_new_species?: number;
  apprise_notify_new_species_each_day?: number;
  apprise_weekly_report?: number;
  apprise_minimum_seconds_between_notifications_per_species?: number;
  apprise_only_notify_species_names?: string;
  apprise_only_notify_species_names_2?: string;

  // === Image Provider ===
  image_provider?: string;
  flickr_api_key?: string;
  flickr_filter_email?: string;

  // === UI Display ===
  color_scheme?: string;
  info_site?: string;

  // === Language ===
  database_lang?: string;

  // === Authentication ===
  caddy_pwd?: string;
  ice_pwd?: string;

  // === Custom URL ===
  birdnetpi_url?: string;

  // === Frequency Shifting (Accessibility) ===
  freqshift_tool?: string;
  freqshift_hi?: number;
  freqshift_lo?: number;
  freqshift_pitch?: number;
  freqshift_reconnect_delay?: number;

  // === Options ===
  silence_update_indicator?: number;
  automatic_update?: number;
  raw_spectrogram?: number;
  rare_species_threshold?: number;
  heartbeat_url?: string;

  // === Custom Image ===
  custom_image?: string;
  custom_image_title?: string;

  // === Logging ===
  log_level_birdnet_recording_service?: string;
  log_level_live_audio_stream_service?: string;
  log_level_spectrogram_viewer_service?: string;

  // === Special Fields (not in config file) ===
  apprise_config?: string;
  apprise_body?: string;
  timezone?: string;
  manual_date?: string;
  manual_time?: string;
  use_ntp?: boolean;
}

// =============================================================================
// Schema Types (internal/config/schema.go)
// =============================================================================

/** Field type for validation */
export type FieldType = 'string' | 'number' | 'integer' | 'boolean';

/** Schema field definition */
export interface FieldSchema {
  type: FieldType;
  description?: string;
  min?: number;
  max?: number;
  enum?: (string | number)[];
  default?: string | number | boolean;
}

/** Configuration schema */
export interface ConfigSchema {
  fields: Record<string, FieldSchema>;
  required: string[];
}

// =============================================================================
// Settings API Response Types (internal/api/settings.go)
// =============================================================================

/** Response from GET /api/settings */
export interface ConfigResponse {
  settings: Config;
  apprise_config: string;
  apprise_body: string;
  timezone: string;
  ntp_enabled: boolean;
  current_time: string;
  schema?: ConfigSchema;
  available_timezones?: string[];
  available_languages?: Record<string, string>;
}

/** Validation error */
export interface ValidationError {
  field: string;
  message: string;
}

/** Response from PUT /api/settings */
export interface UpdateResponse {
  status: 'success' | 'partial' | 'error';
  restarted_services?: string[];
  errors?: ValidationError[];
  message?: string;
}

/** Response from GET /api/settings/schema */
export interface SchemaResponse {
  schema: ConfigSchema;
  available_models: string[];
  available_formats: string[];
  available_languages: Record<string, string>;
  available_log_levels: string[];
}

// =============================================================================
// Service Types (internal/api/services.go)
// =============================================================================

/** Service status values */
export type ServiceStatusValue = 'active' | 'inactive' | 'failed' | 'stalled' | 'unknown';

/** Service status */
export interface ServiceStatus {
  name: string;
  display_name: string;
  status: ServiceStatusValue;
  enabled: boolean;
  message?: string;
}

/** Response from GET /api/services */
export interface ServicesResponse {
  services: ServiceStatus[];
}

/** Service action types */
export type ServiceAction = 'start' | 'stop' | 'restart' | 'enable' | 'disable';

/** Response from POST /api/services/{name}/{action} */
export interface ServiceActionResponse {
  status: 'success' | 'error';
  message?: string;
  output?: string;
}

// =============================================================================
// Available Options
// =============================================================================

/** Available BirdNET models */
export const AVAILABLE_MODELS = [
  'BirdNET_GLOBAL_6K_V2.4_Model_FP16',
  'BirdNET_6K_GLOBAL_MODEL',
] as const;

/** Available audio formats */
export const AVAILABLE_AUDIO_FORMATS = ['mp3', 'wav', 'flac', 'ogg', 'opus'] as const;

/** Available log levels */
export const AVAILABLE_LOG_LEVELS = ['debug', 'info', 'warning', 'error'] as const;

/** Available color schemes */
export const AVAILABLE_COLOR_SCHEMES = ['light', 'dark'] as const;

/** Available full disk actions */
export const AVAILABLE_FULL_DISK_ACTIONS = ['purge', 'keep'] as const;

/** Available info sites */
export const AVAILABLE_INFO_SITES = ['ALLABOUTBIRDS'] as const;

/** Available image providers */
export const AVAILABLE_IMAGE_PROVIDERS = ['WIKIPEDIA', 'FLICKR', ''] as const;

/** Available frequency shift tools */
export const AVAILABLE_FREQSHIFT_TOOLS = ['sox', 'ffmpeg'] as const;

// =============================================================================
// Settings Section Types (for UI organization)
// =============================================================================

/** Settings sections */
export type SettingsSection =
  | 'location'
  | 'model'
  | 'analysis'
  | 'recording'
  | 'notifications'
  | 'display'
  | 'advanced'
  | 'services';

/** Section metadata for UI */
export interface SectionMeta {
  id: SettingsSection;
  title: string;
  description: string;
  icon?: string;
}

/** Available sections */
export const SETTINGS_SECTIONS: SectionMeta[] = [
  {
    id: 'location',
    title: 'Location & Identity',
    description: 'Configure your site name and location for species filtering',
  },
  {
    id: 'model',
    title: 'BirdNET Model',
    description: 'Select the AI model and species filter threshold',
  },
  {
    id: 'analysis',
    title: 'Analysis Parameters',
    description: 'Adjust detection sensitivity and confidence levels',
  },
  {
    id: 'recording',
    title: 'Recording Settings',
    description: 'Configure audio recording parameters',
  },
  {
    id: 'notifications',
    title: 'Notifications',
    description: 'Set up Apprise notifications and BirdWeather integration',
  },
  {
    id: 'display',
    title: 'Display Options',
    description: 'Customize the user interface appearance',
  },
  {
    id: 'advanced',
    title: 'Advanced Settings',
    description: 'Privacy, disk management, passwords, and more',
  },
  {
    id: 'services',
    title: 'Service Controls',
    description: 'Start, stop, and manage BirdNET-Pi services',
  },
];

// =============================================================================
// Managed Services List
// =============================================================================

/** Managed services that can be controlled */
export const MANAGED_SERVICES = [
  { name: 'livestream.service', displayName: 'Live Audio Stream' },
  { name: 'web_terminal.service', displayName: 'Web Terminal' },
  { name: 'birdnet_analysis.service', displayName: 'BirdNET Analysis' },
  { name: 'birdnet_stats.service', displayName: 'Streamlit Statistics' },
  { name: 'birdnet_recording.service', displayName: 'Recording Service' },
  { name: 'chart_viewer.service', displayName: 'Chart Viewer' },
  { name: 'spectrogram_viewer.service', displayName: 'Spectrogram Viewer' },
] as const;
