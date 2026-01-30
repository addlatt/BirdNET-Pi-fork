import type { JSX } from 'preact';
import { useState, useEffect, useMemo, useCallback } from 'preact/hooks';
import { fetchLabels } from '../../hooks/useApi';

/**
 * NotificationSpeciesSelector props
 */
interface NotificationSpeciesSelectorProps {
  /** Label for the field */
  label: string;
  /** Current comma-separated species list */
  value: string;
  /** Callback when the list changes */
  onChange: (value: string) => void;
  /** Help text to display */
  helpText?: string;
  /** Type of list for different styling/warnings */
  listType: 'watchlist' | 'exclude';
}

/**
 * Component for selecting species for notification settings.
 * Provides a searchable dropdown with the ability to add/remove species.
 */
export function NotificationSpeciesSelector({
  label,
  value,
  onChange,
  helpText,
  listType,
}: NotificationSpeciesSelectorProps): JSX.Element {
  const [isOpen, setIsOpen] = useState(false);
  const [search, setSearch] = useState('');
  const [allLabels, setAllLabels] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Parse current value into array
  const selectedSpecies = useMemo(() => {
    if (!value || value.trim() === '') return [];
    return value.split(',').map(s => s.trim()).filter(s => s.length > 0);
  }, [value]);

  // Load labels when dropdown opens
  useEffect(() => {
    if (isOpen && allLabels.length === 0) {
      loadLabels();
    }
  }, [isOpen]);

  const loadLabels = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await fetchLabels();
      setAllLabels(data.labels || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load species');
    } finally {
      setLoading(false);
    }
  };

  // Filter available labels based on search and already selected
  const filteredLabels = useMemo(() => {
    const selectedSet = new Set(selectedSpecies);
    const searchLower = search.toLowerCase();
    return allLabels
      .filter(label => !selectedSet.has(label))
      .filter(label => !searchLower || label.toLowerCase().includes(searchLower))
      .slice(0, 100); // Limit for performance
  }, [allLabels, selectedSpecies, search]);

  // Add a species to the list
  const addSpecies = useCallback((species: string) => {
    const newList = [...selectedSpecies, species].sort();
    onChange(newList.join(','));
    setSearch('');
  }, [selectedSpecies, onChange]);

  // Remove a species from the list
  const removeSpecies = useCallback((species: string) => {
    const newList = selectedSpecies.filter(s => s !== species);
    onChange(newList.join(','));
  }, [selectedSpecies, onChange]);

  // Handle click outside to close dropdown
  useEffect(() => {
    if (!isOpen) return;

    const handleClickOutside = (e: MouseEvent) => {
      const target = e.target as HTMLElement;
      if (!target.closest('.notification-species-selector')) {
        setIsOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [isOpen]);

  return (
    <div class="mb-4 notification-species-selector">
      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
        {label}
      </label>

      {/* Selected species tags */}
      <div class="flex flex-wrap gap-2 mb-2 min-h-[32px]">
        {selectedSpecies.length === 0 ? (
          <span class="text-sm text-gray-500 dark:text-gray-400 italic">
            {listType === 'watchlist' ? 'No species selected (notifications for all species)' : 'No species excluded'}
          </span>
        ) : (
          selectedSpecies.map(species => (
            <span
              key={species}
              class={`inline-flex items-center gap-1 px-2 py-1 text-sm rounded-full ${
                listType === 'watchlist'
                  ? 'bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-200'
                  : 'bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-200'
              }`}
            >
              {species}
              <button
                type="button"
                onClick={() => removeSpecies(species)}
                class="ml-1 hover:opacity-70 focus:outline-none"
                aria-label={`Remove ${species}`}
              >
                <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </span>
          ))
        )}
      </div>

      {/* Dropdown trigger and search */}
      <div class="relative">
        <div class="flex gap-2">
          <input
            type="text"
            value={search}
            onInput={(e) => setSearch((e.target as HTMLInputElement).value)}
            onFocus={() => setIsOpen(true)}
            placeholder="Search species to add..."
            class="flex-1 py-2 px-3 border border-gray-300 dark:border-gray-600 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                   placeholder-gray-500 dark:placeholder-gray-400
                   focus:ring-2 focus:ring-primary-500 focus:border-transparent"
          />
          <button
            type="button"
            onClick={() => setIsOpen(!isOpen)}
            class="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg
                   bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300
                   hover:bg-gray-50 dark:hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-primary-500"
          >
            <svg class={`w-5 h-5 transition-transform ${isOpen ? 'rotate-180' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
            </svg>
          </button>
        </div>

        {/* Dropdown list */}
        {isOpen && (
          <div class="absolute z-50 mt-1 w-full max-h-60 overflow-y-auto
                      bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600
                      rounded-lg shadow-lg">
            {loading ? (
              <div class="p-4 text-center">
                <div class="animate-spin rounded-full h-6 w-6 border-b-2 border-primary-600 mx-auto"></div>
                <p class="text-sm text-gray-500 dark:text-gray-400 mt-2">Loading species...</p>
              </div>
            ) : error ? (
              <div class="p-4 text-center text-red-500 dark:text-red-400">
                <p class="text-sm">{error}</p>
                <button
                  onClick={loadLabels}
                  class="mt-2 text-sm text-primary-600 hover:underline"
                >
                  Retry
                </button>
              </div>
            ) : filteredLabels.length === 0 ? (
              <div class="p-4 text-center text-gray-500 dark:text-gray-400">
                <p class="text-sm">
                  {search ? 'No matching species found' : 'All species already selected'}
                </p>
              </div>
            ) : (
              filteredLabels.map(species => (
                <button
                  key={species}
                  type="button"
                  onClick={() => addSpecies(species)}
                  class="w-full px-4 py-2 text-left text-sm
                         text-gray-700 dark:text-gray-300
                         hover:bg-gray-100 dark:hover:bg-gray-700
                         focus:bg-gray-100 dark:focus:bg-gray-700 focus:outline-none"
                >
                  {species}
                </button>
              ))
            )}
          </div>
        )}
      </div>

      {helpText && (
        <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">{helpText}</p>
      )}
    </div>
  );
}
