import type { JSX } from 'preact';
import { useState, useCallback } from 'preact/hooks';

/**
 * SearchFilters props
 */
interface SearchFiltersProps {
  /** Current search term */
  search: string;
  /** Current minimum confidence (0-100) */
  minConfidence: number;
  /** Callback when search changes */
  onSearchChange: (search: string) => void;
  /** Callback when confidence filter changes */
  onConfidenceChange: (confidence: number) => void;
  /** Whether filters are currently being applied */
  loading?: boolean;
}

/**
 * Confidence filter options (percentage values)
 */
const CONFIDENCE_OPTIONS = [
  { value: 0, label: 'All' },
  { value: 50, label: '50%+' },
  { value: 70, label: '70%+' },
  { value: 80, label: '80%+' },
  { value: 90, label: '90%+' },
];

/**
 * SearchFilters - Search bar and confidence filter for detections.
 */
export function SearchFilters({
  search,
  minConfidence,
  onSearchChange,
  onConfidenceChange,
  loading = false,
}: SearchFiltersProps): JSX.Element {
  const [localSearch, setLocalSearch] = useState(search);

  // Debounced search submit
  const handleSearchSubmit = useCallback(
    (e: Event) => {
      e.preventDefault();
      onSearchChange(localSearch);
    },
    [localSearch, onSearchChange]
  );

  // Handle Enter key
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === 'Enter') {
        onSearchChange(localSearch);
      }
    },
    [localSearch, onSearchChange]
  );

  // Clear search
  const handleClear = useCallback(() => {
    setLocalSearch('');
    onSearchChange('');
  }, [onSearchChange]);

  return (
    <div class="flex flex-col sm:flex-row gap-4 mb-6">
      {/* Search Input */}
      <form onSubmit={handleSearchSubmit} class="flex-1 relative">
        <div class="relative">
          <input
            type="text"
            placeholder="Search species, time, file..."
            value={localSearch}
            onInput={(e) => setLocalSearch((e.target as HTMLInputElement).value)}
            onKeyDown={handleKeyDown}
            disabled={loading}
            class="w-full pl-10 pr-10 py-2 border border-gray-300 dark:border-gray-600 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                   placeholder-gray-500 dark:placeholder-gray-400
                   focus:ring-2 focus:ring-primary-500 focus:border-transparent
                   disabled:opacity-50 disabled:cursor-not-allowed"
          />
          {/* Search Icon */}
          <svg
            class="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
            />
          </svg>
          {/* Clear Button */}
          {localSearch && (
            <button
              type="button"
              onClick={handleClear}
              class="absolute right-3 top-1/2 transform -translate-y-1/2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
            >
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          )}
        </div>
      </form>

      {/* Confidence Filter */}
      <div class="flex items-center gap-2">
        <label class="text-sm text-gray-500 dark:text-gray-400 whitespace-nowrap">
          Min Confidence:
        </label>
        <select
          value={minConfidence}
          onChange={(e) => onConfidenceChange(Number((e.target as HTMLSelectElement).value))}
          disabled={loading}
          class="py-2 px-3 border border-gray-300 dark:border-gray-600 rounded-lg
                 bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                 focus:ring-2 focus:ring-primary-500 focus:border-transparent
                 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {CONFIDENCE_OPTIONS.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>
      </div>
    </div>
  );
}
