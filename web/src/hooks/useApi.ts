import { useState, useCallback } from 'preact/hooks';
import type {
  Detection,
  ListDetectionsResponse,
  ListDetectionsParams,
  DeleteDetectionResponse,
  Species,
  ListSpeciesResponse,
  ListSpeciesParams,
  SpeciesDetail,
  SpeciesHistoryResponse,
  SpeciesHistoryParams,
  SpeciesRankingResponse,
  SpeciesRankingParams,
  StatsResponse,
  StatsParams,
  HeatmapResponse,
  SystemStatus,
  SystemMemoryResponse,
  HealthResponse,
  ListDatesResponse,
  ListDatesParams,
  SpeciesListsResponse,
  SpeciesListType,
  LabelsResponse,
  SpeciesCountResponse,
  DeleteSpeciesResponse,
  SpectrogramInfoResponse,
  RecentDetection,
  RecentDetectionsResponse,
  // Recordings types
  ListRecordingDatesResponse,
  RecordingSpecies,
  ListRecordingSpeciesResponse,
  ListRecordingSpeciesParams,
  RecordingFile,
  ListRecordingFilesResponse,
  ListRecordingFilesParams,
  ToggleLockResponse,
  ToggleShiftResponse,
  ExclusionListResponse,
  // Backup types
  RestoreResponse,
  RestoreStatusResponse,
} from '../types/api';

const API_BASE = '/api';

/**
 * Generic API request options
 */
interface RequestOptions extends Omit<RequestInit, 'body'> {
  body?: unknown;
}

/**
 * API hook state and methods
 */
interface UseApiReturn {
  loading: boolean;
  error: string | null;
  get: <T>(endpoint: string) => Promise<T>;
  post: <T>(endpoint: string, data: unknown) => Promise<T>;
  put: <T>(endpoint: string, data: unknown) => Promise<T>;
  delete: <T>(endpoint: string) => Promise<T>;
}

/**
 * Custom hook for making API requests.
 * @returns API methods and state
 */
export function useApi(): UseApiReturn {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const request = useCallback(async <T>(endpoint: string, options: RequestOptions = {}): Promise<T> => {
    setLoading(true);
    setError(null);

    try {
      const { body, ...restOptions } = options;
      const response = await fetch(`${API_BASE}${endpoint}`, {
        ...restOptions,
        headers: {
          'Content-Type': 'application/json',
          ...restOptions.headers,
        },
        body: body ? JSON.stringify(body) : undefined,
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `HTTP error ${response.status}`);
      }

      const data = await response.json() as T;
      setLoading(false);
      return data;
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      setError(message);
      setLoading(false);
      throw err;
    }
  }, []);

  const get = useCallback(<T>(endpoint: string): Promise<T> => request<T>(endpoint), [request]);

  const post = useCallback(
    <T>(endpoint: string, data: unknown): Promise<T> =>
      request<T>(endpoint, { method: 'POST', body: data }),
    [request]
  );

  const put = useCallback(
    <T>(endpoint: string, data: unknown): Promise<T> =>
      request<T>(endpoint, { method: 'PUT', body: data }),
    [request]
  );

  const del = useCallback(
    <T>(endpoint: string): Promise<T> =>
      request<T>(endpoint, { method: 'DELETE' }),
    [request]
  );

  return {
    loading,
    error,
    get,
    post,
    put,
    delete: del,
  };
}

// =============================================================================
// Typed API Functions
// =============================================================================

/**
 * Build query string from params object
 */
function buildQuery(params: Record<string, string | number | boolean | undefined>): string {
  const entries = Object.entries(params)
    .filter(([, value]) => value !== undefined)
    .map(([key, value]) => [key, String(value)]);

  if (entries.length === 0) return '';
  return '?' + new URLSearchParams(entries as [string, string][]).toString();
}

/**
 * Generic fetch wrapper with error handling
 */
async function apiFetch<T>(url: string): Promise<T> {
  const response = await fetch(url);
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || `HTTP error ${response.status}`);
  }
  return response.json() as Promise<T>;
}

// =============================================================================
// Detection Endpoints
// =============================================================================

/**
 * Fetch detections from the API.
 * GET /api/detections
 */
export async function fetchDetections(params: ListDetectionsParams = {}): Promise<ListDetectionsResponse> {
  const query = buildQuery(params as Record<string, string | number | undefined>);
  return apiFetch<ListDetectionsResponse>(`${API_BASE}/detections${query}`);
}

/**
 * Delete a detection by its composite key.
 * DELETE /api/detections/{date}/{time}/{species}
 */
export async function deleteDetection(date: string, time: string, sciName: string): Promise<DeleteDetectionResponse> {
  const response = await fetch(
    `${API_BASE}/detections/${encodeURIComponent(date)}/${encodeURIComponent(time)}/${encodeURIComponent(sciName)}`,
    { method: 'DELETE' }
  );
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || `HTTP error ${response.status}`);
  }
  return response.json() as Promise<DeleteDetectionResponse>;
}

// =============================================================================
// Species Endpoints
// =============================================================================

/**
 * Fetch species list from the API.
 * GET /api/species
 */
export async function fetchSpecies(params: ListSpeciesParams = {}): Promise<ListSpeciesResponse> {
  const query = buildQuery(params as Record<string, string | undefined>);
  return apiFetch<ListSpeciesResponse>(`${API_BASE}/species${query}`);
}

/**
 * Fetch species detail from the API.
 * GET /api/species/{name}
 */
export async function fetchSpeciesDetail(name: string): Promise<SpeciesDetail> {
  return apiFetch<SpeciesDetail>(`${API_BASE}/species/${encodeURIComponent(name)}`);
}

/**
 * Fetch species detection history for mini-chart.
 * GET /api/species/{name}/history
 */
export async function fetchSpeciesHistory(name: string, params: SpeciesHistoryParams = {}): Promise<SpeciesHistoryResponse> {
  const query = buildQuery(params as Record<string, string | number | undefined>);
  return apiFetch<SpeciesHistoryResponse>(`${API_BASE}/species/${encodeURIComponent(name)}/history${query}`);
}

/**
 * Fetch all species with last_seen date.
 * GET /api/species/all
 */
export async function fetchAllSpecies(): Promise<ListSpeciesResponse> {
  return apiFetch<ListSpeciesResponse>(`${API_BASE}/species/all`);
}

/**
 * Fetch species ranking with latest and best detection info.
 * GET /api/species/ranking
 */
export async function fetchSpeciesRanking(params: SpeciesRankingParams = {}): Promise<SpeciesRankingResponse> {
  const query = buildQuery(params as Record<string, string | undefined>);
  return apiFetch<SpeciesRankingResponse>(`${API_BASE}/species/ranking${query}`);
}

/**
 * Get count of detections and files for a species before deletion.
 * GET /api/species/{name}/count
 */
export async function fetchSpeciesCount(name: string): Promise<SpeciesCountResponse> {
  return apiFetch<SpeciesCountResponse>(`${API_BASE}/species/${encodeURIComponent(name)}/count`);
}

/**
 * Delete all detections and files for a species.
 * DELETE /api/species/{name}/all
 */
export async function deleteAllSpeciesDetections(name: string): Promise<DeleteSpeciesResponse> {
  const response = await fetch(
    `${API_BASE}/species/${encodeURIComponent(name)}/all`,
    { method: 'DELETE' }
  );
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || `HTTP error ${response.status}`);
  }
  return response.json() as Promise<DeleteSpeciesResponse>;
}

// =============================================================================
// Species Lists Endpoints (for Species Management)
// =============================================================================

/**
 * Fetch all species lists (confirmed, excluded, whitelisted, include).
 * GET /api/species-lists
 */
export async function fetchSpeciesLists(): Promise<SpeciesListsResponse> {
  return apiFetch<SpeciesListsResponse>(`${API_BASE}/species-lists`);
}

/**
 * Add a species to a list.
 * POST /api/species-lists/{listType}/add
 */
export async function addToSpeciesList(listType: SpeciesListType, species: string): Promise<{ status: string }> {
  const response = await fetch(`${API_BASE}/species-lists/${listType}/add`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ species }),
  });
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || `HTTP error ${response.status}`);
  }
  return response.json();
}

/**
 * Remove a species from a list.
 * POST /api/species-lists/{listType}/remove
 */
export async function removeFromSpeciesList(listType: SpeciesListType, species: string): Promise<{ status: string }> {
  const response = await fetch(`${API_BASE}/species-lists/${listType}/remove`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ species }),
  });
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || `HTTP error ${response.status}`);
  }
  return response.json();
}

/**
 * Replace an entire species list.
 * PUT /api/species-lists/{listType}
 */
export async function updateSpeciesList(listType: SpeciesListType, species: string[]): Promise<{ status: string }> {
  const response = await fetch(`${API_BASE}/species-lists/${listType}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ species }),
  });
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || `HTTP error ${response.status}`);
  }
  return response.json();
}

/**
 * Fetch all available species labels from labels.txt.
 * GET /api/labels
 */
export async function fetchLabels(): Promise<LabelsResponse> {
  return apiFetch<LabelsResponse>(`${API_BASE}/labels`);
}

// =============================================================================
// Dates Endpoints (for History page)
// =============================================================================

/**
 * Fetch dates with detections from the API.
 * GET /api/dates
 */
export async function fetchDates(params: ListDatesParams = {}): Promise<ListDatesResponse> {
  const query = buildQuery(params as Record<string, string | number | undefined>);
  return apiFetch<ListDatesResponse>(`${API_BASE}/dates${query}`);
}

// =============================================================================
// Stats Endpoints
// =============================================================================

/**
 * Fetch stats from the API.
 * GET /api/stats
 */
export async function fetchStats(params: StatsParams = {}): Promise<StatsResponse> {
  const query = buildQuery(params as Record<string, string | number | undefined>);
  return apiFetch<StatsResponse>(`${API_BASE}/stats${query}`);
}

// =============================================================================
// Heatmap Endpoints
// =============================================================================

/**
 * Fetch today's species-hourly heatmap data.
 * GET /api/heatmap/today
 */
export async function fetchHeatmapToday(): Promise<HeatmapResponse> {
  return apiFetch<HeatmapResponse>(`${API_BASE}/heatmap/today`);
}

// =============================================================================
// System Endpoints
// =============================================================================

/**
 * Fetch system status from the API.
 * GET /api/system/status
 */
export async function fetchSystemStatus(): Promise<SystemStatus> {
  return apiFetch<SystemStatus>(`${API_BASE}/system/status`);
}

/**
 * Fetch system memory from the API.
 * GET /api/system/memory
 */
export async function fetchSystemMemory(): Promise<SystemMemoryResponse> {
  return apiFetch<SystemMemoryResponse>(`${API_BASE}/system/memory`);
}

// =============================================================================
// Health Endpoints
// =============================================================================

/**
 * Fetch health status from the API.
 * GET /api/health
 */
export async function fetchHealth(): Promise<HealthResponse> {
  return apiFetch<HealthResponse>(`${API_BASE}/health`);
}

// =============================================================================
// Spectrogram Endpoints
// =============================================================================

/**
 * Fetch spectrogram info/metadata from the API.
 * GET /api/spectrogram/info
 */
export async function fetchSpectrogramInfo(): Promise<SpectrogramInfoResponse> {
  return apiFetch<SpectrogramInfoResponse>(`${API_BASE}/spectrogram/info`);
}

/**
 * Get the spectrogram image URL with cache-busting parameter.
 */
export function getSpectrogramImageUrl(): string {
  return `${API_BASE}/spectrogram/image?t=${Date.now()}`;
}

/** Query parameters for GET /api/spectrogram/detections */
export interface RecentDetectionsParams {
  limit?: number;
}

/**
 * Fetch recent detections for the spectrogram page.
 * GET /api/spectrogram/detections
 */
export async function fetchRecentDetections(params: RecentDetectionsParams = {}): Promise<RecentDetectionsResponse> {
  const query = buildQuery(params as Record<string, string | number | undefined>);
  return apiFetch<RecentDetectionsResponse>(`${API_BASE}/spectrogram/detections${query}`);
}

// =============================================================================
// Recordings Endpoints
// =============================================================================

/**
 * Fetch recording dates from the API.
 * GET /api/recordings/dates
 */
export async function fetchRecordingDates(limit?: number): Promise<ListRecordingDatesResponse> {
  const query = limit ? `?limit=${limit}` : '';
  return apiFetch<ListRecordingDatesResponse>(`${API_BASE}/recordings/dates${query}`);
}

/**
 * Fetch recording species list from the API.
 * GET /api/recordings/species
 */
export async function fetchRecordingSpecies(params: ListRecordingSpeciesParams = {}): Promise<ListRecordingSpeciesResponse> {
  const query = buildQuery(params as Record<string, string | undefined>);
  return apiFetch<ListRecordingSpeciesResponse>(`${API_BASE}/recordings/species${query}`);
}

/**
 * Fetch species for a specific date.
 * GET /api/recordings/by-date/{date}
 */
export async function fetchRecordingsByDate(date: string): Promise<ListRecordingSpeciesResponse> {
  return apiFetch<ListRecordingSpeciesResponse>(`${API_BASE}/recordings/by-date/${encodeURIComponent(date)}`);
}

/**
 * Fetch recording files for a specific species.
 * GET /api/recordings/by-species/{name}
 */
export async function fetchRecordingsBySpecies(name: string, params: ListRecordingFilesParams = {}): Promise<ListRecordingFilesResponse> {
  const query = buildQuery(params as Record<string, string | number | boolean | undefined>);
  return apiFetch<ListRecordingFilesResponse>(`${API_BASE}/recordings/by-species/${encodeURIComponent(name)}${query}`);
}

/**
 * Delete a recording.
 * POST /api/recordings/{date}/{species}/{filename}/delete
 */
export async function deleteRecording(date: string, species: string, filename: string): Promise<{ status: string }> {
  const response = await fetch(
    `${API_BASE}/recordings/${encodeURIComponent(date)}/${encodeURIComponent(species)}/${encodeURIComponent(filename)}/delete`,
    { method: 'POST' }
  );
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || `HTTP error ${response.status}`);
  }
  return response.json();
}

/**
 * Change the identification of a recording.
 * POST /api/recordings/{date}/{species}/{filename}/change
 */
export async function changeRecordingIdentification(date: string, species: string, filename: string, newSpecies: string): Promise<{ status: string }> {
  const response = await fetch(
    `${API_BASE}/recordings/${encodeURIComponent(date)}/${encodeURIComponent(species)}/${encodeURIComponent(filename)}/change`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ new_species: newSpecies }),
    }
  );
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || `HTTP error ${response.status}`);
  }
  return response.json();
}

/**
 * Toggle the purge lock on a recording.
 * POST /api/recordings/{date}/{species}/{filename}/lock
 */
export async function toggleRecordingLock(date: string, species: string, filename: string): Promise<ToggleLockResponse> {
  const response = await fetch(
    `${API_BASE}/recordings/${encodeURIComponent(date)}/${encodeURIComponent(species)}/${encodeURIComponent(filename)}/lock`,
    { method: 'POST' }
  );
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || `HTTP error ${response.status}`);
  }
  return response.json();
}

/**
 * Toggle the frequency shift on a recording.
 * POST /api/recordings/{date}/{species}/{filename}/shift
 */
export async function toggleRecordingShift(date: string, species: string, filename: string): Promise<ToggleShiftResponse> {
  const response = await fetch(
    `${API_BASE}/recordings/${encodeURIComponent(date)}/${encodeURIComponent(species)}/${encodeURIComponent(filename)}/shift`,
    { method: 'POST' }
  );
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || `HTTP error ${response.status}`);
  }
  return response.json();
}

/**
 * Fetch the exclusion list.
 * GET /api/recordings/exclusions
 */
export async function fetchExclusionList(): Promise<ExclusionListResponse> {
  return apiFetch<ExclusionListResponse>(`${API_BASE}/recordings/exclusions`);
}

// =============================================================================
// Backup Endpoints
// =============================================================================

/**
 * Download a backup archive.
 * POST /api/backup/create - returns a binary .tar.gz stream.
 */
export async function downloadBackup(): Promise<void> {
  const response = await fetch(`${API_BASE}/backup/create`, { method: 'POST' });
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || `HTTP error ${response.status}`);
  }

  const blob = await response.blob();
  const disposition = response.headers.get('Content-Disposition');
  let filename = 'birdnet-backup.tar.gz';
  if (disposition) {
    const match = disposition.match(/filename="?([^"]+)"?/);
    if (match) {
      filename = match[1];
    }
  }

  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}

/**
 * Upload a backup file for restore.
 * POST /api/backup/restore (multipart/form-data)
 */
export async function uploadRestore(file: File): Promise<RestoreResponse> {
  const formData = new FormData();
  formData.append('backup', file);

  const response = await fetch(`${API_BASE}/backup/restore`, {
    method: 'POST',
    body: formData,
  });
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || `HTTP error ${response.status}`);
  }
  return response.json() as Promise<RestoreResponse>;
}

/**
 * Fetch restore status by ID.
 * GET /api/backup/status?id={restore_id}
 */
export async function fetchRestoreStatus(id: string): Promise<RestoreStatusResponse> {
  return apiFetch<RestoreStatusResponse>(`${API_BASE}/backup/status?id=${encodeURIComponent(id)}`);
}

// =============================================================================
// Re-export types for convenience
// =============================================================================

export type {
  Detection,
  ListDetectionsResponse,
  ListDetectionsParams,
  DeleteDetectionResponse,
  Species,
  ListSpeciesResponse,
  ListSpeciesParams,
  SpeciesDetail,
  SpeciesHistoryResponse,
  SpeciesHistoryParams,
  SpeciesRankingResponse,
  SpeciesRankingParams,
  StatsResponse,
  StatsParams,
  HeatmapResponse,
  SystemStatus,
  SystemMemoryResponse,
  HealthResponse,
  ListDatesResponse,
  ListDatesParams,
  SpeciesListsResponse,
  SpeciesListType,
  LabelsResponse,
  SpeciesCountResponse,
  DeleteSpeciesResponse,
  SpectrogramInfoResponse,
  RecentDetection,
  RecentDetectionsResponse,
  // Recordings types
  ListRecordingDatesResponse,
  RecordingSpecies,
  ListRecordingSpeciesResponse,
  ListRecordingSpeciesParams,
  RecordingFile,
  ListRecordingFilesResponse,
  ListRecordingFilesParams,
  ToggleLockResponse,
  ToggleShiftResponse,
  ExclusionListResponse,
  // Backup types
  RestoreResponse,
  RestoreStatusResponse,
};
