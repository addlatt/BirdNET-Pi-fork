import type { JSX } from 'preact';
import { useState, useEffect, useCallback } from 'preact/hooks';
import { fetchSpeciesHistory } from '../hooks/useApi';
import type { SpeciesHistoryDayEntry } from '../types/api';

/**
 * SpeciesMiniChart props
 */
interface SpeciesMiniChartProps {
  /** Scientific name of the species */
  sciName: string;
  /** Common name of the species */
  comName: string;
  /** Callback to close the modal */
  onClose: () => void;
}

/**
 * Modal displaying detection history for a species as a bar chart.
 */
export function SpeciesMiniChart({ sciName, comName, onClose }: SpeciesMiniChartProps): JSX.Element {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [history, setHistory] = useState<SpeciesHistoryDayEntry[]>([]);
  const [days, setDays] = useState(30);

  // Load history data
  useEffect(() => {
    async function loadHistory() {
      try {
        setLoading(true);
        setError(null);
        const data = await fetchSpeciesHistory(sciName, { days });
        setHistory(data.history || []);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load history');
      } finally {
        setLoading(false);
      }
    }
    loadHistory();
  }, [sciName, days]);

  // Handle click outside to close
  const handleBackdropClick = useCallback(
    (e: MouseEvent) => {
      if ((e.target as HTMLElement).classList.contains('modal-backdrop')) {
        onClose();
      }
    },
    [onClose]
  );

  // Handle escape key to close
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
      }
    };
    document.addEventListener('keydown', handleEscape);
    return () => document.removeEventListener('keydown', handleEscape);
  }, [onClose]);

  // Calculate max value for chart scaling
  const maxCount = Math.max(1, ...history.map((h) => h.detection_count));
  const totalDetections = history.reduce((sum, h) => sum + h.detection_count, 0);
  const daysWithDetections = history.filter((h) => h.detection_count > 0).length;

  return (
    <div
      class="modal-backdrop fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      onClick={handleBackdropClick}
    >
      <div class="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-2xl w-full mx-4 max-h-[90vh] overflow-hidden">
        {/* Header */}
        <div class="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700">
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{comName}</h2>
            <p class="text-sm text-gray-500 dark:text-gray-400 italic">{sciName}</p>
          </div>
          <button
            onClick={onClose}
            class="p-2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Content */}
        <div class="p-4">
          {/* Days selector */}
          <div class="flex items-center justify-between mb-4">
            <span class="text-sm text-gray-500 dark:text-gray-400">Detection History</span>
            <select
              value={days}
              onChange={(e) => setDays(Number((e.target as HTMLSelectElement).value))}
              class="text-sm py-1 px-2 border border-gray-300 dark:border-gray-600 rounded
                     bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
            >
              <option value={7}>Last 7 days</option>
              <option value={14}>Last 14 days</option>
              <option value={30}>Last 30 days</option>
              <option value={60}>Last 60 days</option>
              <option value={90}>Last 90 days</option>
            </select>
          </div>

          {loading ? (
            <div class="flex items-center justify-center h-48">
              <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
            </div>
          ) : error ? (
            <div class="text-center text-red-600 dark:text-red-400 py-8">{error}</div>
          ) : history.length === 0 ? (
            <div class="text-center text-gray-500 dark:text-gray-400 py-8">
              No detections in the last {days} days
            </div>
          ) : (
            <>
              {/* Summary stats */}
              <div class="grid grid-cols-3 gap-4 mb-4">
                <div class="text-center p-2 bg-gray-50 dark:bg-gray-700/50 rounded">
                  <div class="text-xl font-bold text-gray-900 dark:text-white">{totalDetections}</div>
                  <div class="text-xs text-gray-500 dark:text-gray-400">Total</div>
                </div>
                <div class="text-center p-2 bg-gray-50 dark:bg-gray-700/50 rounded">
                  <div class="text-xl font-bold text-gray-900 dark:text-white">{daysWithDetections}</div>
                  <div class="text-xs text-gray-500 dark:text-gray-400">Days Active</div>
                </div>
                <div class="text-center p-2 bg-gray-50 dark:bg-gray-700/50 rounded">
                  <div class="text-xl font-bold text-gray-900 dark:text-white">
                    {daysWithDetections > 0 ? Math.round(totalDetections / daysWithDetections) : 0}
                  </div>
                  <div class="text-xs text-gray-500 dark:text-gray-400">Avg/Day</div>
                </div>
              </div>

              {/* Bar Chart */}
              <div class="h-48 flex items-end gap-1">
                {history.map((entry) => {
                  const height = (entry.detection_count / maxCount) * 100;
                  const dateObj = new Date(entry.date);
                  const dayLabel = dateObj.toLocaleDateString('en-US', { weekday: 'short' });
                  const dateLabel = dateObj.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });

                  return (
                    <div
                      key={entry.date}
                      class="flex-1 flex flex-col items-center justify-end group relative"
                    >
                      {/* Tooltip */}
                      <div
                        class="absolute bottom-full mb-2 px-2 py-1 bg-gray-900 text-white text-xs rounded
                               opacity-0 group-hover:opacity-100 transition-opacity whitespace-nowrap z-10"
                      >
                        {dateLabel}: {entry.detection_count} detection{entry.detection_count !== 1 ? 's' : ''}
                      </div>

                      {/* Bar */}
                      <div
                        class={`w-full rounded-t transition-all ${
                          entry.detection_count > 0
                            ? 'bg-primary-500 hover:bg-primary-600'
                            : 'bg-gray-200 dark:bg-gray-700'
                        }`}
                        style={{ height: `${Math.max(height, entry.detection_count > 0 ? 4 : 2)}%` }}
                      />

                      {/* Day label (only show for some bars to avoid crowding) */}
                      {(days <= 14 || history.indexOf(entry) % 3 === 0) && (
                        <div class="text-xs text-gray-400 mt-1 truncate w-full text-center">
                          {days <= 14 ? dayLabel.charAt(0) : dateObj.getDate()}
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>

              {/* X-axis labels */}
              <div class="flex justify-between mt-2 text-xs text-gray-400">
                <span>
                  {history[0] ? new Date(history[0].date).toLocaleDateString('en-US', { month: 'short', day: 'numeric' }) : ''}
                </span>
                <span>
                  {history[history.length - 1]
                    ? new Date(history[history.length - 1].date).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
                    : ''}
                </span>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
