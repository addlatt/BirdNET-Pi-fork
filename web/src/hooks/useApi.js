import { useState, useCallback } from 'preact/hooks';

const API_BASE = '/api';

/**
 * Custom hook for making API requests.
 * @returns {Object} API methods and state
 */
export function useApi() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const request = useCallback(async (endpoint, options = {}) => {
    setLoading(true);
    setError(null);

    try {
      const response = await fetch(`${API_BASE}${endpoint}`, {
        ...options,
        headers: {
          'Content-Type': 'application/json',
          ...options.headers,
        },
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `HTTP error ${response.status}`);
      }

      const data = await response.json();
      setLoading(false);
      return data;
    } catch (err) {
      setError(err.message);
      setLoading(false);
      throw err;
    }
  }, []);

  const get = useCallback((endpoint) => request(endpoint), [request]);

  const post = useCallback(
    (endpoint, data) =>
      request(endpoint, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    [request]
  );

  const put = useCallback(
    (endpoint, data) =>
      request(endpoint, {
        method: 'PUT',
        body: JSON.stringify(data),
      }),
    [request]
  );

  const del = useCallback(
    (endpoint) =>
      request(endpoint, {
        method: 'DELETE',
      }),
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

/**
 * Fetch detections from the API.
 * @param {Object} params - Query parameters
 * @returns {Promise<Object>} Detections response
 */
export async function fetchDetections(params = {}) {
  const query = new URLSearchParams(params).toString();
  const response = await fetch(`${API_BASE}/detections${query ? `?${query}` : ''}`);
  if (!response.ok) {
    throw new Error('Failed to fetch detections');
  }
  return response.json();
}

/**
 * Fetch species from the API.
 * @param {Object} params - Query parameters
 * @returns {Promise<Object>} Species response
 */
export async function fetchSpecies(params = {}) {
  const query = new URLSearchParams(params).toString();
  const response = await fetch(`${API_BASE}/species${query ? `?${query}` : ''}`);
  if (!response.ok) {
    throw new Error('Failed to fetch species');
  }
  return response.json();
}

/**
 * Fetch stats from the API.
 * @param {Object} params - Query parameters
 * @returns {Promise<Object>} Stats response
 */
export async function fetchStats(params = {}) {
  const query = new URLSearchParams(params).toString();
  const response = await fetch(`${API_BASE}/stats${query ? `?${query}` : ''}`);
  if (!response.ok) {
    throw new Error('Failed to fetch stats');
  }
  return response.json();
}

/**
 * Fetch system status from the API.
 * @returns {Promise<Object>} System status response
 */
export async function fetchSystemStatus() {
  const response = await fetch(`${API_BASE}/system/status`);
  if (!response.ok) {
    throw new Error('Failed to fetch system status');
  }
  return response.json();
}
