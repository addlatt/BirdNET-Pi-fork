import type { JSX } from 'preact';
import { useMemo } from 'preact/hooks';

/**
 * DatePicker props
 */
interface DatePickerProps {
  /** Currently selected date (YYYY-MM-DD) */
  selectedDate: string;
  /** List of dates that have detections */
  availableDates: string[];
  /** Callback when date changes */
  onDateChange: (date: string) => void;
  /** Whether the component is loading */
  loading?: boolean;
}

/**
 * Date picker component for History page.
 * Shows date input with quick navigation to dates with detections.
 */
export function DatePicker({
  selectedDate,
  availableDates,
  onDateChange,
  loading = false,
}: DatePickerProps): JSX.Element {
  // Convert to Set for O(1) lookup
  const availableDateSet = useMemo(() => new Set(availableDates), [availableDates]);

  // Check if selected date has detections
  const hasDetections = availableDateSet.has(selectedDate);

  // Get today's date in local timezone
  const now = new Date();
  const today = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`;

  // Find previous and next dates with detections
  const { prevDate, nextDate } = useMemo(() => {
    if (availableDates.length === 0) {
      return { prevDate: null, nextDate: null };
    }

    // availableDates is sorted DESC (most recent first)
    const currentIndex = availableDates.indexOf(selectedDate);

    let prev: string | null = null;
    let next: string | null = null;

    if (currentIndex === -1) {
      // Selected date not in list, find closest dates
      for (let i = 0; i < availableDates.length; i++) {
        if (availableDates[i] < selectedDate && !prev) {
          prev = availableDates[i];
        }
        if (availableDates[i] > selectedDate) {
          next = availableDates[i];
        }
      }
    } else {
      // Current date is in list
      if (currentIndex < availableDates.length - 1) {
        prev = availableDates[currentIndex + 1]; // Earlier date (DESC order)
      }
      if (currentIndex > 0) {
        next = availableDates[currentIndex - 1]; // Later date (DESC order)
      }
    }

    return { prevDate: prev, nextDate: next };
  }, [availableDates, selectedDate]);

  // Handle date input change
  const handleDateChange = (e: Event) => {
    const target = e.target as HTMLInputElement;
    onDateChange(target.value);
  };

  // Jump to today
  const handleTodayClick = () => {
    onDateChange(today);
  };

  return (
    <div class="space-y-4">
      {/* Main date selector row */}
      <div class="flex flex-wrap items-center gap-3">
        {/* Previous date button */}
        <button
          onClick={() => prevDate && onDateChange(prevDate)}
          disabled={!prevDate || loading}
          class="p-2 rounded-lg border border-gray-200 dark:border-gray-700
                 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300
                 hover:bg-gray-50 dark:hover:bg-gray-700
                 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          title={prevDate ? `Previous: ${prevDate}` : 'No earlier detections'}
        >
          <ChevronLeftIcon class="w-5 h-5" />
        </button>

        {/* Date input */}
        <div class="flex-1 min-w-[200px]">
          <input
            type="date"
            value={selectedDate}
            onChange={handleDateChange}
            max={today}
            disabled={loading}
            class="w-full px-4 py-2 text-center text-lg font-medium
                   rounded-lg border border-gray-200 dark:border-gray-700
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                   focus:ring-2 focus:ring-primary-500 focus:border-transparent
                   disabled:opacity-50 disabled:cursor-not-allowed"
          />
        </div>

        {/* Next date button */}
        <button
          onClick={() => nextDate && onDateChange(nextDate)}
          disabled={!nextDate || loading}
          class="p-2 rounded-lg border border-gray-200 dark:border-gray-700
                 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300
                 hover:bg-gray-50 dark:hover:bg-gray-700
                 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          title={nextDate ? `Next: ${nextDate}` : 'No later detections'}
        >
          <ChevronRightIcon class="w-5 h-5" />
        </button>

        {/* Today button */}
        <button
          onClick={handleTodayClick}
          disabled={selectedDate === today || loading}
          class="px-4 py-2 rounded-lg border border-gray-200 dark:border-gray-700
                 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300
                 hover:bg-gray-50 dark:hover:bg-gray-700
                 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          Today
        </button>
      </div>

      {/* Status indicator */}
      <div class="flex items-center justify-between text-sm">
        <span class={`${hasDetections ? 'text-green-600 dark:text-green-400' : 'text-gray-500 dark:text-gray-400'}`}>
          {hasDetections ? (
            <span class="flex items-center">
              <CheckIcon class="w-4 h-4 mr-1" />
              Detections available for this date
            </span>
          ) : (
            <span class="flex items-center">
              <InfoIcon class="w-4 h-4 mr-1" />
              No detections on this date
            </span>
          )}
        </span>
        <span class="text-gray-500 dark:text-gray-400">
          {availableDates.length} day{availableDates.length !== 1 ? 's' : ''} with data
        </span>
      </div>

      {/* Recent dates quick access */}
      {availableDates.length > 0 && (
        <div class="flex flex-wrap gap-2">
          <span class="text-sm text-gray-500 dark:text-gray-400 py-1">Recent:</span>
          {availableDates.slice(0, 7).map((date) => (
            <button
              key={date}
              onClick={() => onDateChange(date)}
              disabled={loading}
              class={`px-3 py-1 text-sm rounded-full transition-colors
                ${date === selectedDate
                  ? 'bg-primary-600 text-white'
                  : 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-600'
                }
                disabled:opacity-50 disabled:cursor-not-allowed`}
            >
              {formatDateShort(date)}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

/**
 * Format date as short string (e.g., "Jan 11")
 */
function formatDateShort(dateStr: string): string {
  const date = new Date(dateStr + 'T00:00:00');
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
}

// =============================================================================
// Icon Components
// =============================================================================

function ChevronLeftIcon({ class: className }: { class?: string }): JSX.Element {
  return (
    <svg class={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7" />
    </svg>
  );
}

function ChevronRightIcon({ class: className }: { class?: string }): JSX.Element {
  return (
    <svg class={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
    </svg>
  );
}

function CheckIcon({ class: className }: { class?: string }): JSX.Element {
  return (
    <svg class={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
    </svg>
  );
}

function InfoIcon({ class: className }: { class?: string }): JSX.Element {
  return (
    <svg class={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <circle cx="12" cy="12" r="10" />
      <path stroke-linecap="round" stroke-linejoin="round" d="M12 16v-4m0-4h.01" />
    </svg>
  );
}
