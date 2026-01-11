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
  StatsResponse,
  StatsParams,
  SystemStatus,
  SystemMemoryResponse,
  HealthResponse,
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
  StatsResponse,
  StatsParams,
  SystemStatus,
  SystemMemoryResponse,
  HealthResponse,
};
