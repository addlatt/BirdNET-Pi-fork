import type { JSX } from 'preact';
import { useCallback } from 'preact/hooks';
import { useServices } from '../hooks/useSettings';
import type { ServiceStatus, ServiceAction } from '../types/settings';
import { AlertMessage } from './settings/FormInputs';

/**
 * ServiceControls component - Manage BirdNET-Pi services.
 */
export function ServiceControls(): JSX.Element {
  const {
    services,
    loading,
    actionLoading,
    error,
    actionError,
    lastActionResult,
    refresh,
    performAction,
    restartAll,
    clearErrors,
  } = useServices(true, 10000); // Auto-refresh every 10 seconds

  const handleAction = useCallback(
    async (serviceName: string, action: ServiceAction) => {
      await performAction(serviceName, action);
    },
    [performAction]
  );

  const handleRestartAll = useCallback(async () => {
    await restartAll();
  }, [restartAll]);

  // Get status badge color
  const getStatusColor = (status: ServiceStatus['status']): string => {
    switch (status) {
      case 'active':
        return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200';
      case 'inactive':
        return 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300';
      case 'failed':
        return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200';
      case 'stalled':
        return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200';
      default:
        return 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400';
    }
  };

  // Get status icon
  const getStatusIcon = (status: ServiceStatus['status']): JSX.Element => {
    switch (status) {
      case 'active':
        return (
          <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
            <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
          </svg>
        );
      case 'inactive':
        return (
          <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
            <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8 7a1 1 0 00-1 1v4a1 1 0 001 1h4a1 1 0 001-1V8a1 1 0 00-1-1H8z" clip-rule="evenodd" />
          </svg>
        );
      case 'failed':
        return (
          <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
            <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
          </svg>
        );
      case 'stalled':
        return (
          <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
            <path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd" />
          </svg>
        );
      default:
        return (
          <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
            <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clip-rule="evenodd" />
          </svg>
        );
    }
  };

  if (loading && services.length === 0) {
    return (
      <div class="flex items-center justify-center min-h-[200px]">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600" />
      </div>
    );
  }

  return (
    <div class="max-w-4xl mx-auto p-4 sm:p-6">
      <div class="mb-6 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Service Controls</h1>
          <p class="text-gray-500 dark:text-gray-400 mt-1">
            Manage BirdNET-Pi background services
          </p>
        </div>
        <div class="flex gap-2">
          <button
            type="button"
            onClick={refresh}
            disabled={loading}
            class="inline-flex items-center px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg
                   text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700
                   disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <svg class={`w-4 h-4 mr-2 ${loading ? 'animate-spin' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            Refresh
          </button>
          <button
            type="button"
            onClick={handleRestartAll}
            disabled={actionLoading === 'all'}
            class="inline-flex items-center px-3 py-2 bg-primary-600 text-white rounded-lg
                   hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {actionLoading === 'all' && (
              <svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
              </svg>
            )}
            Restart All
          </button>
        </div>
      </div>

      {/* Status Messages */}
      {error && (
        <AlertMessage type="error" message={error} onDismiss={clearErrors} />
      )}
      {actionError && (
        <AlertMessage type="error" message={actionError} onDismiss={clearErrors} />
      )}
      {lastActionResult?.status === 'success' && (
        <AlertMessage type="success" message={lastActionResult.message || 'Action completed'} onDismiss={clearErrors} />
      )}

      {/* Services List */}
      <div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 overflow-hidden">
        <ul class="divide-y divide-gray-200 dark:divide-gray-700">
          {services.map((service) => (
            <li key={service.name} class="p-4">
              <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                {/* Service Info */}
                <div class="flex-1 min-w-0">
                  <div class="flex items-center gap-3">
                    <h3 class="text-sm font-semibold text-gray-900 dark:text-white truncate">
                      {service.display_name}
                    </h3>
                    <span class={`inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium ${getStatusColor(service.status)}`}>
                      {getStatusIcon(service.status)}
                      {service.status}
                    </span>
                    {!service.enabled && (
                      <span class="px-2 py-0.5 rounded text-xs bg-gray-200 dark:bg-gray-600 text-gray-600 dark:text-gray-300">
                        Disabled
                      </span>
                    )}
                  </div>
                  <p class="text-xs text-gray-500 dark:text-gray-400 mt-1 font-mono">
                    {service.name}
                  </p>
                  {service.message && (
                    <p class="text-xs text-yellow-600 dark:text-yellow-400 mt-1">
                      {service.message}
                    </p>
                  )}
                </div>

                {/* Action Buttons */}
                <div class="flex flex-wrap gap-2">
                  {service.status === 'active' ? (
                    <>
                      <ServiceButton
                        onClick={() => handleAction(service.name, 'restart')}
                        loading={actionLoading === service.name}
                        variant="primary"
                      >
                        Restart
                      </ServiceButton>
                      <ServiceButton
                        onClick={() => handleAction(service.name, 'stop')}
                        loading={actionLoading === service.name}
                        variant="danger"
                      >
                        Stop
                      </ServiceButton>
                    </>
                  ) : (
                    <ServiceButton
                      onClick={() => handleAction(service.name, 'start')}
                      loading={actionLoading === service.name}
                      variant="success"
                    >
                      Start
                    </ServiceButton>
                  )}
                  {service.enabled ? (
                    <ServiceButton
                      onClick={() => handleAction(service.name, 'disable')}
                      loading={actionLoading === service.name}
                      variant="secondary"
                    >
                      Disable
                    </ServiceButton>
                  ) : (
                    <ServiceButton
                      onClick={() => handleAction(service.name, 'enable')}
                      loading={actionLoading === service.name}
                      variant="secondary"
                    >
                      Enable
                    </ServiceButton>
                  )}
                </div>
              </div>
            </li>
          ))}
        </ul>
      </div>

      {/* Help Text */}
      <div class="mt-6 p-4 bg-blue-50 dark:bg-blue-900/30 rounded-lg border border-blue-200 dark:border-blue-700">
        <h4 class="text-sm font-medium text-blue-800 dark:text-blue-200 mb-2">
          Service Status Guide
        </h4>
        <ul class="text-sm text-blue-700 dark:text-blue-300 space-y-1">
          <li class="flex items-center gap-2">
            <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200">
              {getStatusIcon('active')} active
            </span>
            Service is running normally
          </li>
          <li class="flex items-center gap-2">
            <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300">
              {getStatusIcon('inactive')} inactive
            </span>
            Service is stopped
          </li>
          <li class="flex items-center gap-2">
            <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200">
              {getStatusIcon('failed')} failed
            </span>
            Service crashed or failed to start
          </li>
          <li class="flex items-center gap-2">
            <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200">
              {getStatusIcon('stalled')} stalled
            </span>
            Service appears stuck (large backlog)
          </li>
        </ul>
      </div>

      {/* Link to Settings */}
      <div class="mt-4 p-4 bg-gray-50 dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
        <p class="text-gray-600 dark:text-gray-400">
          Configure service logging levels in{' '}
          <a href="/advanced-settings" class="text-primary-600 hover:underline">
            Advanced Settings
          </a>
        </p>
      </div>
    </div>
  );
}

// =============================================================================
// ServiceButton Component
// =============================================================================

interface ServiceButtonProps {
  onClick: () => void;
  loading: boolean;
  variant: 'primary' | 'secondary' | 'success' | 'danger';
  children: string;
}

function ServiceButton({ onClick, loading, variant, children }: ServiceButtonProps): JSX.Element {
  const variantClasses = {
    primary: 'bg-primary-600 hover:bg-primary-700 text-white',
    secondary: 'border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700',
    success: 'bg-green-600 hover:bg-green-700 text-white',
    danger: 'bg-red-600 hover:bg-red-700 text-white',
  };

  return (
    <button
      type="button"
      onClick={onClick}
      disabled={loading}
      class={`
        inline-flex items-center px-3 py-1.5 rounded text-sm font-medium
        transition-colors duration-200
        disabled:opacity-50 disabled:cursor-not-allowed
        ${variantClasses[variant]}
      `}
    >
      {loading && (
        <svg class="animate-spin -ml-1 mr-1.5 h-3 w-3" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
        </svg>
      )}
      {children}
    </button>
  );
}

export default ServiceControls;
