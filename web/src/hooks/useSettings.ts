import { useState, useCallback, useEffect } from 'preact/hooks';
import type {
  Config,
  ConfigUpdate,
  ConfigResponse,
  UpdateResponse,
  SchemaResponse,
  ServicesResponse,
  ServiceStatus,
  ServiceAction,
  ServiceActionResponse,
  ValidationError,
} from '../types/settings';

const API_BASE = '/api';

/**
 * Generic fetch wrapper with error handling
 */
async function apiFetch<T>(url: string, options?: RequestInit): Promise<T> {
  const response = await fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || `HTTP error ${response.status}`);
  }

  return response.json() as Promise<T>;
}

// =============================================================================
// Settings API Functions
// =============================================================================

/**
 * Fetch current settings from the API.
 * GET /api/settings
 */
export async function fetchSettings(): Promise<ConfigResponse> {
  return apiFetch<ConfigResponse>(`${API_BASE}/settings`);
}

/**
 * Update settings via the API.
 * PUT /api/settings
 */
export async function updateSettings(update: ConfigUpdate): Promise<UpdateResponse> {
  return apiFetch<UpdateResponse>(`${API_BASE}/settings`, {
    method: 'PUT',
    body: JSON.stringify(update),
  });
}

/**
 * Fetch settings schema from the API.
 * GET /api/settings/schema
 */
export async function fetchSettingsSchema(): Promise<SchemaResponse> {
  return apiFetch<SchemaResponse>(`${API_BASE}/settings/schema`);
}

// =============================================================================
// Services API Functions
// =============================================================================

/**
 * Fetch service statuses from the API.
 * GET /api/services
 */
export async function fetchServices(): Promise<ServicesResponse> {
  return apiFetch<ServicesResponse>(`${API_BASE}/services`);
}

/**
 * Perform an action on a service.
 * POST /api/services/{name}/{action}
 */
export async function performServiceAction(
  serviceName: string,
  action: ServiceAction
): Promise<ServiceActionResponse> {
  return apiFetch<ServiceActionResponse>(
    `${API_BASE}/services/${encodeURIComponent(serviceName)}/${action}`,
    { method: 'POST' }
  );
}

/**
 * Restart all services.
 * POST /api/services/restart-all
 */
export async function restartAllServices(): Promise<ServiceActionResponse> {
  return apiFetch<ServiceActionResponse>(`${API_BASE}/services/restart-all`, {
    method: 'POST',
  });
}

// =============================================================================
// useSettings Hook
// =============================================================================

interface UseSettingsState {
  config: Config | null;
  appriseConfig: string;
  appriseBody: string;
  timezone: string;
  ntpEnabled: boolean;
  availableTimezones: string[];
  availableLanguages: Record<string, string>;
  loading: boolean;
  saving: boolean;
  error: string | null;
  saveError: string | null;
  validationErrors: ValidationError[];
  restartedServices: string[];
}

interface UseSettingsReturn extends UseSettingsState {
  /** Reload settings from server */
  refresh: () => Promise<void>;
  /** Save settings changes */
  save: (update: ConfigUpdate) => Promise<boolean>;
  /** Clear any errors */
  clearErrors: () => void;
}

/**
 * Hook for managing BirdNET-Pi settings.
 * Provides state management for configuration with auto-refresh capability.
 */
export function useSettings(autoRefresh = true): UseSettingsReturn {
  const [state, setState] = useState<UseSettingsState>({
    config: null,
    appriseConfig: '',
    appriseBody: '',
    timezone: '',
    ntpEnabled: true,
    availableTimezones: [],
    availableLanguages: {},
    loading: true,
    saving: false,
    error: null,
    saveError: null,
    validationErrors: [],
    restartedServices: [],
  });

  const refresh = useCallback(async () => {
    setState((prev) => ({ ...prev, loading: true, error: null }));

    try {
      const response = await fetchSettings();
      setState((prev) => ({
        ...prev,
        config: response.settings,
        appriseConfig: response.apprise_config,
        appriseBody: response.apprise_body,
        timezone: response.timezone,
        ntpEnabled: response.ntp_enabled,
        availableTimezones: response.available_timezones || [],
        availableLanguages: response.available_languages || {},
        loading: false,
        error: null,
      }));
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to load settings';
      setState((prev) => ({
        ...prev,
        loading: false,
        error: message,
      }));
    }
  }, []);

  const save = useCallback(async (update: ConfigUpdate): Promise<boolean> => {
    setState((prev) => ({
      ...prev,
      saving: true,
      saveError: null,
      validationErrors: [],
      restartedServices: [],
    }));

    try {
      const response = await updateSettings(update);

      if (response.status === 'error') {
        setState((prev) => ({
          ...prev,
          saving: false,
          saveError: response.message || 'Failed to save settings',
          validationErrors: response.errors || [],
        }));
        return false;
      }

      setState((prev) => ({
        ...prev,
        saving: false,
        restartedServices: response.restarted_services || [],
      }));

      // Refresh to get updated values
      await refresh();
      return true;
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to save settings';
      setState((prev) => ({
        ...prev,
        saving: false,
        saveError: message,
      }));
      return false;
    }
  }, [refresh]);

  const clearErrors = useCallback(() => {
    setState((prev) => ({
      ...prev,
      error: null,
      saveError: null,
      validationErrors: [],
    }));
  }, []);

  // Auto-refresh on mount
  useEffect(() => {
    if (autoRefresh) {
      refresh();
    }
  }, [autoRefresh, refresh]);

  return {
    ...state,
    refresh,
    save,
    clearErrors,
  };
}

// =============================================================================
// useServices Hook
// =============================================================================

interface UseServicesState {
  services: ServiceStatus[];
  loading: boolean;
  actionLoading: string | null;
  error: string | null;
  actionError: string | null;
  lastActionResult: ServiceActionResponse | null;
}

interface UseServicesReturn extends UseServicesState {
  /** Reload services from server */
  refresh: () => Promise<void>;
  /** Perform an action on a service */
  performAction: (serviceName: string, action: ServiceAction) => Promise<boolean>;
  /** Restart all services */
  restartAll: () => Promise<boolean>;
  /** Clear any errors */
  clearErrors: () => void;
}

/**
 * Hook for managing BirdNET-Pi services.
 * Provides state management for service control with status polling.
 */
export function useServices(autoRefresh = true, pollInterval = 10000): UseServicesReturn {
  const [state, setState] = useState<UseServicesState>({
    services: [],
    loading: true,
    actionLoading: null,
    error: null,
    actionError: null,
    lastActionResult: null,
  });

  const refresh = useCallback(async () => {
    setState((prev) => ({ ...prev, loading: prev.services.length === 0, error: null }));

    try {
      const response = await fetchServices();
      setState((prev) => ({
        ...prev,
        services: response.services,
        loading: false,
        error: null,
      }));
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to load services';
      setState((prev) => ({
        ...prev,
        loading: false,
        error: message,
      }));
    }
  }, []);

  const performAction = useCallback(
    async (serviceName: string, action: ServiceAction): Promise<boolean> => {
      setState((prev) => ({
        ...prev,
        actionLoading: serviceName,
        actionError: null,
        lastActionResult: null,
      }));

      try {
        const result = await performServiceAction(serviceName, action);
        setState((prev) => ({
          ...prev,
          actionLoading: null,
          lastActionResult: result,
        }));

        // Refresh to get updated statuses
        await refresh();
        return result.status === 'success';
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Action failed';
        setState((prev) => ({
          ...prev,
          actionLoading: null,
          actionError: message,
        }));
        return false;
      }
    },
    [refresh]
  );

  const restartAll = useCallback(async (): Promise<boolean> => {
    setState((prev) => ({
      ...prev,
      actionLoading: 'all',
      actionError: null,
      lastActionResult: null,
    }));

    try {
      const result = await restartAllServices();
      setState((prev) => ({
        ...prev,
        actionLoading: null,
        lastActionResult: result,
      }));

      // Wait a moment then refresh
      setTimeout(refresh, 2000);
      return result.status === 'success';
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to restart all services';
      setState((prev) => ({
        ...prev,
        actionLoading: null,
        actionError: message,
      }));
      return false;
    }
  }, [refresh]);

  const clearErrors = useCallback(() => {
    setState((prev) => ({
      ...prev,
      error: null,
      actionError: null,
    }));
  }, []);

  // Auto-refresh on mount and poll
  useEffect(() => {
    if (autoRefresh) {
      refresh();

      if (pollInterval > 0) {
        const intervalId = setInterval(refresh, pollInterval);
        return () => clearInterval(intervalId);
      }
    }
  }, [autoRefresh, pollInterval, refresh]);

  return {
    ...state,
    refresh,
    performAction,
    restartAll,
    clearErrors,
  };
}

// =============================================================================
// Re-export types for convenience
// =============================================================================

export type {
  Config,
  ConfigUpdate,
  ConfigResponse,
  UpdateResponse,
  SchemaResponse,
  ServicesResponse,
  ServiceStatus,
  ServiceAction,
  ServiceActionResponse,
  ValidationError,
};
