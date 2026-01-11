import { useState, useEffect, useCallback } from 'preact/hooks';
import type { JSX } from 'preact';
import { fetchDetections, fetchDates } from '../hooks/useApi';
import type { Detection } from '../types/api';
import { DetectionList } from '../components/DetectionList';
import { DatePicker } from '../components/DatePicker';
import { SearchFilters } from '../components/SearchFilters';

/**
 * History page component.
 * Displays bird detections for any selected date with a date picker.
 */
export function History(): JSX.Element {
  // Date state
  const [selectedDate, setSelectedDate] = useState<string>(() => {
    // Default to today
    return new Date().toISOString().split('T')[0];
  });
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

  // Load available dates on mount
  useEffect(() => {
    async function loadDates() {
      try {
        setDatesLoading(true);
        const data = await fetchDates({ limit: 365 });
        setAvailableDates(data.dates || []);
      } catch (err) {
        console.error('Failed to load dates:', err);
        // Don't block the UI if dates fail to load
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

  // Handle date change - reset to page 1
  const handleDateChange = useCallback((date: string) => {
    setSelectedDate(date);
    setPage(1);
  }, []);

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
      {/* Header */}
      <div class="flex items-center justify-between">
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">History</h1>
      </div>

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
                  {formattedDate}
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
