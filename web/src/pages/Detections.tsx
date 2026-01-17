import { useState, useEffect, useCallback, useMemo } from 'preact/hooks';
import { route } from 'preact-router';
import type { JSX } from 'preact';
import { useWebSocket } from '../hooks/useWebSocket';
import { fetchDetections, fetchDates } from '../hooks/useApi';
import type { Detection, DetectionNotification } from '../types/api';
import { DetectionList } from '../components/DetectionList';
import { StatsHeader } from '../components/StatsHeader';
import { DatePicker } from '../components/DatePicker';
import { SearchFilters } from '../components/SearchFilters';

/**
 * Get today's date in YYYY-MM-DD format (local timezone)
 */
function getTodayDate(): string {
  const now = new Date();
  const year = now.getFullYear();
  const month = String(now.getMonth() + 1).padStart(2, '0');
  const day = String(now.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

/**
 * Props for the Detections component
 */
interface DetectionsProps {
  /** Date from URL parameter (YYYY-MM-DD) */
  date?: string;
}

/**
 * Unified Detections page component.
 * Displays bird detections for any date with real-time updates when viewing today.
 * Uses URL parameter for date selection: /detections or /detections?date=2024-01-15
 */
export function Detections({ date }: DetectionsProps): JSX.Element {
  // Determine if we're viewing today
  const today = useMemo(() => getTodayDate(), []);
  const selectedDate = date || today;
  const isToday = selectedDate === today;

  // Date picker state
  const [availableDates, setAvailableDates] = useState<string[]>([]);
  const [datesLoading, setDatesLoading] = useState(true);

  // Detection state
  const [detections, setDetections] = useState<Detection[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const perPage = 20;

  // Filter state
  const [search, setSearch] = useState('');
  const [minConfidence, setMinConfidence] = useState(0);

  // WebSocket connection (only used when viewing today)
  const wsUrl = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws`;
  const { isConnected, subscribe } = useWebSocket(wsUrl);

  // Load available dates on mount
  useEffect(() => {
    async function loadDates() {
      try {
        setDatesLoading(true);
        const data = await fetchDates({ limit: 365 });
        setAvailableDates(data.dates || []);
      } catch (err) {
        console.error('Failed to load dates:', err);
        setAvailableDates([]);
      } finally {
        setDatesLoading(false);
      }
    }
    loadDates();
  }, []);

  // Load detections when date or filters change
  const loadDetections = useCallback(async () => {
    try {
      setLoading(true);
      const data = await fetchDetections({
        date: selectedDate,
        page,
        per_page: perPage,
        search: search || undefined,
        min_confidence: minConfidence > 0 ? minConfidence / 100 : undefined,
      });
      setDetections(data.detections || []);
      setTotal(data.total || 0);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setDetections([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }, [selectedDate, page, search, minConfidence]);

  // Reload detections when dependencies change
  useEffect(() => {
    loadDetections();
  }, [loadDetections]);

  // Subscribe to real-time detection updates (only when viewing today)
  useEffect(() => {
    if (!isToday) {
      return; // Don't subscribe to WebSocket updates for historical dates
    }

    const unsubscribe = subscribe<DetectionNotification>('detection', (payload) => {
      // Only add if it passes current filters
      const confidencePercent = payload.confidence * 100;
      if (minConfidence > 0 && confidencePercent < minConfidence) {
        return;
      }

      if (search) {
        const searchLower = search.toLowerCase();
        const matchesSearch =
          payload.com_name.toLowerCase().includes(searchLower) ||
          payload.sci_name.toLowerCase().includes(searchLower) ||
          payload.file_name.toLowerCase().includes(searchLower) ||
          payload.time.includes(search);
        if (!matchesSearch) {
          return;
        }
      }

      // Convert notification to Detection format
      const detection: Detection = {
        date: payload.date,
        time: payload.time,
        sci_name: payload.sci_name,
        com_name: payload.com_name,
        confidence: payload.confidence,
        file_name: payload.file_name,
      };

      // Add new detection to the top of the list
      setDetections((prev) => [detection, ...prev]);
      setTotal((prev) => prev + 1);
    });

    return unsubscribe;
  }, [subscribe, search, minConfidence, isToday]);

  // Handle date change via DatePicker - update URL
  const handleDateChange = useCallback(
    (newDate: string) => {
      // Reset page when date changes
      setPage(1);
      // Update URL - use base path for today, query param for other dates
      if (newDate === today) {
        route('/detections');
      } else {
        route(`/detections?date=${newDate}`);
      }
    },
    [today]
  );

  // Handle search change - reset to page 1
  const handleSearchChange = useCallback((newSearch: string) => {
    setSearch(newSearch);
    setPage(1);
  }, []);

  // Handle confidence change - reset to page 1
  const handleConfidenceChange = useCallback((newConfidence: number) => {
    setMinConfidence(newConfidence);
    setPage(1);
  }, []);

  // Handle detection deletion
  const handleDelete = useCallback((detection: Detection) => {
    setDetections((prev) =>
      prev.filter(
        (d) =>
          !(d.date === detection.date && d.time === detection.time && d.sci_name === detection.sci_name)
      )
    );
    setTotal((prev) => Math.max(0, prev - 1));
  }, []);

  const totalPages = Math.ceil(total / perPage);

  // Format selected date for display
  const formattedDate = new Date(selectedDate + 'T00:00:00').toLocaleDateString('en-US', {
    weekday: 'long',
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  });

  return (
    <div class="space-y-6">
      {/* Header with title and connection status (only for today) */}
      <div class="flex items-center justify-between">
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Detections</h1>
        {isToday && (
          <span class={`flex items-center text-sm ${isConnected ? 'text-green-600' : 'text-red-600'}`}>
            <span class={`w-2 h-2 rounded-full mr-2 ${isConnected ? 'bg-green-600' : 'bg-red-600'}`}></span>
            {isConnected ? 'Live' : 'Offline'}
          </span>
        )}
      </div>

      {/* Stats Header - only shown when viewing today */}
      {isToday && <StatsHeader />}

      {/* Date Picker Card */}
      <div class="card p-4">
        <DatePicker
          selectedDate={selectedDate}
          availableDates={availableDates}
          onDateChange={handleDateChange}
          loading={datesLoading || loading}
        />
      </div>

      {/* Search and Filter Controls */}
      <SearchFilters
        search={search}
        minConfidence={minConfidence}
        onSearchChange={handleSearchChange}
        onConfidenceChange={handleConfidenceChange}
        loading={loading}
      />

      {/* Detection List Card */}
      <div class="card">
        {loading ? (
          <div class="flex items-center justify-center h-64">
            <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
          </div>
        ) : error ? (
          <div class="p-6 text-center">
            <p class="text-red-600 dark:text-red-400">Error: {error}</p>
            <button class="btn btn-primary mt-4" onClick={() => loadDetections()}>
              Retry
            </button>
          </div>
        ) : (
          <>
            {/* Results summary */}
            <div class="p-4 border-b border-gray-200 dark:border-gray-700">
              <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2">
                <span class="text-lg font-medium text-gray-900 dark:text-white">
                  {isToday ? "Today's Detections" : formattedDate}
                </span>
                <span class="text-sm text-gray-500 dark:text-gray-400">
                  {total} detection{total !== 1 ? 's' : ''}
                  {search && ` matching "${search}"`}
                  {minConfidence > 0 && ` (${minConfidence}%+ confidence)`}
                </span>
              </div>
            </div>

            {/* Detection list */}
            <DetectionList detections={detections} onDelete={handleDelete} />

            {/* Pagination */}
            {totalPages > 1 && (
              <div class="p-4 border-t border-gray-200 dark:border-gray-700 flex items-center justify-between">
                <button
                  class="btn btn-secondary"
                  disabled={page === 1}
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                >
                  Previous
                </button>
                <span class="text-sm text-gray-500 dark:text-gray-400">
                  Page {page} of {totalPages}
                </span>
                <button
                  class="btn btn-secondary"
                  disabled={page >= totalPages}
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                >
                  Next
                </button>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}
