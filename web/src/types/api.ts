/**
 * API Types - These mirror the Go API response structs exactly.
 * Any changes to the Go API should be reflected here.
 *
 * Source: internal/api/*.go
 */

// =============================================================================
// Detection Types (internal/api/detections.go)
// =============================================================================

/** Single detection from the database */
export interface Detection {
  date: string;
  time: string;
  sci_name: string;
  com_name: string;
  confidence: number;
  lat?: number;
  lon?: number;
  file_name: string;
}

/** Response from GET /api/detections */
export interface ListDetectionsResponse {
  detections: Detection[];
  total: number;
  page: number;
  per_page: number;
}

/** Query parameters for GET /api/detections */
export interface ListDetectionsParams {
  page?: number;
  per_page?: number;
  date?: string;
  start_date?: string;
  end_date?: string;
  species?: string;
  /** Text search across com_name, sci_name, file_name, time */
  search?: string;
  /** Minimum confidence threshold (0.0-1.0) */
  min_confidence?: number;
}

/** Response from DELETE /api/detections/{date}/{time}/{species} */
export interface DeleteDetectionResponse {
  status: 'deleted';
}

// =============================================================================
// Species Types (internal/api/species.go)
// =============================================================================

/** Species summary */
export interface Species {
  sci_name: string;
  com_name: string;
  detection_count: number;
  max_confidence: number;
  last_seen?: string;
}

/** Response from GET /api/species */
export interface ListSpeciesResponse {
  species: Species[];
  total: number;
}

/** Query parameters for GET /api/species */
export interface ListSpeciesParams {
  today?: 'true' | 'false';
  sort?: 'alphabetical' | 'occurrences' | 'confidence' | 'date';
}

/** Response from GET /api/species/{name} */
export interface SpeciesDetail {
  sci_name: string;
  com_name: string;
  detection_count: number;
  max_confidence: number;
  best_date: string;
  best_time: string;
  best_file_name: string;
  audio_url: string;
  spectrogram_url: string;
}

/** Single day entry in species detection history */
export interface SpeciesHistoryDayEntry {
  date: string;
  detection_count: number;
}

/** Response from GET /api/species/{name}/history */
export interface SpeciesHistoryResponse {
  species: string;
  days: number;
  history: SpeciesHistoryDayEntry[];
}

/** Query parameters for GET /api/species/{name}/history */
export interface SpeciesHistoryParams {
  days?: number;
}

// =============================================================================
// Stats Types (internal/api/stats.go)
// =============================================================================

/** Daily statistics */
export interface DailyStat {
  date: string;
  detection_count: number;
  species_count: number;
  avg_confidence: number;
}

/** Hourly distribution entry */
export interface HourlyStat {
  hour: number;
  detection_count: number;
}

/** Top species entry */
export interface TopSpecies {
  sci_name: string;
  com_name: string;
  detection_count: number;
}

/** New species detected today (first time ever) */
export interface NewSpecies {
  sci_name: string;
  com_name: string;
  first_time: string;
  max_confidence: number;
  detection_count: number;
}

/** Response from GET /api/stats */
export interface StatsResponse {
  total_detections: number;
  total_species: number;
  detections_today: number;
  detections_last_hour: number;
  species_today: number;
  daily_stats?: DailyStat[];
  hourly_distribution?: HourlyStat[];
  top_species?: TopSpecies[];
  new_species_today?: NewSpecies[];
}

/** Query parameters for GET /api/stats */
export interface StatsParams {
  days?: number;
  include_daily?: 'true' | 'false';
  include_hourly?: 'true' | 'false';
  include_top_species?: 'true' | 'false';
  include_new_species?: 'true' | 'false';
  top_limit?: number;
}

// =============================================================================
// Heatmap Types (internal/api/heatmap.go)
// =============================================================================

/** Response from GET /api/heatmap/today */
export interface HeatmapResponse {
  date: string;
  species: string[];
  hours: number[];
  data: number[][];  // [species_index][hour] = count
  total_detections: number;
}

// =============================================================================
// System Types (internal/api/system.go)
// =============================================================================

/** Database status */
export interface DatabaseStatus {
  connected: boolean;
  path?: string;
}

/** Response from GET /api/system/status */
export interface SystemStatus {
  status: string;
  version: string;
  go_version: string;
  uptime_seconds: number;
  ml_service_status: string;
  websocket_clients: number;
  database: DatabaseStatus;
}

/** Go runtime memory stats */
export interface GoMemory {
  heap_alloc: number;
  heap_sys: number;
  heap_in_use: number;
  stack_in_use: number;
  num_goroutine: number;
}

/** System memory stats */
export interface SystemMem {
  total: number;
  free: number;
  available: number;
  used: number;
}

/** ML service memory stats */
export interface MLMemory {
  birdnet: number;
  vad: number;
  llm: number;
  total: number;
}

/** Response from GET /api/system/memory */
export interface SystemMemoryResponse {
  go: GoMemory;
  system: SystemMem;
  ml?: MLMemory;
}

// =============================================================================
// Health Types (internal/api/health.go)
// =============================================================================

/** Response from GET /api/health */
export interface HealthResponse {
  status: string;
}

// =============================================================================
// WebSocket Types (internal/ws/messages.go)
// =============================================================================

/** WebSocket message envelope */
export interface WSMessage<T = unknown> {
  type: string;
  channel?: string;
  payload: T;
}

/** Detection notification payload */
export interface DetectionNotification {
  date: string;
  time: string;
  sci_name: string;
  com_name: string;
  confidence: number;
  file_name: string;
}

/** WebSocket message types */
export type WSMessageType =
  | 'detection'
  | 'status'
  | 'spectrogram_frame'
  | 'llm_stream'
  | 'vad_result';

/** WebSocket subscription request */
export interface WSSubscription {
  type: 'subscribe' | 'unsubscribe';
  channel: string;
}

// =============================================================================
// Dates Types (for History page)
// =============================================================================

/** Response from GET /api/dates */
export interface ListDatesResponse {
  dates: string[];
  total: number;
}

/** Query parameters for GET /api/dates */
export interface ListDatesParams {
  limit?: number;
}

// =============================================================================
// Species Management Types (internal/api/species_lists.go)
// =============================================================================

/** List types for species management */
export type SpeciesListType = 'confirmed' | 'excluded' | 'whitelisted' | 'include';

/** Response from GET /api/species-lists */
export interface SpeciesListsResponse {
  confirmed: string[];
  excluded: string[];
  whitelisted: string[];
  include: string[];
}

/** Request body for add/remove operations */
export interface SpeciesListRequest {
  species: string;
}

/** Request body for full list replacement */
export interface UpdateSpeciesListRequest {
  species: string[];
}

/** Response from GET /api/labels */
export interface LabelsResponse {
  labels: string[];
  total: number;
}

/** Response from GET /api/species/{name}/count */
export interface SpeciesCountResponse {
  detection_count: number;
  file_count: number;
}

/** Response from DELETE /api/species/{name}/all */
export interface DeleteSpeciesResponse {
  detections_deleted: number;
  files_deleted: number;
}

// =============================================================================
// Spectrogram Types (internal/api/spectrogram.go)
// =============================================================================

/** Response from GET /api/spectrogram/info */
export interface SpectrogramInfoResponse {
  image_url: string;
  last_modified?: string;
  available: boolean;
  livestream_url: string;
  refresh_seconds: number;
}

/** Recent detection for spectrogram page */
export interface RecentDetection {
  time: string;
  com_name: string;
  sci_name: string;
  confidence: number;
  file_name: string;
}

/** Response from GET /api/spectrogram/detections */
export interface RecentDetectionsResponse {
  detections: RecentDetection[];
  total: number;
}

// =============================================================================
// API Error Types
// =============================================================================

/** Error response from API */
export interface ApiError {
  error: string;
  status?: number;
}

/** Type guard to check if response is an error */
export function isApiError(response: unknown): response is ApiError {
  return (
    typeof response === 'object' &&
    response !== null &&
    'error' in response &&
    typeof (response as ApiError).error === 'string'
  );
}
