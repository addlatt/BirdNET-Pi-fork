import type { JSX } from 'preact';
import { useState, useEffect, useMemo, useCallback } from 'preact/hooks';
import type { Species, SpeciesListsResponse } from '../types/api';
import { SpeciesMiniChart } from './SpeciesMiniChart';

/**
 * Sort column type
 */
type SortColumn = 'com_name' | 'sci_name' | 'detection_count' | 'max_confidence' | 'last_seen';

/**
 * Sort direction type
 */
type SortDirection = 'asc' | 'desc';

/**
 * SpeciesTable props
 */
interface SpeciesTableProps {
  /** List of species to display */
  species: Species[];
  /** Species lists for confirmed/excluded/whitelisted status */
  speciesLists: SpeciesListsResponse;
  /** Callback to toggle a species in a list */
  onToggleList: (listType: 'confirmed' | 'excluded' | 'whitelisted', sciName: string, action: 'add' | 'remove') => void;
  /** Callback to delete a species */
  onDeleteSpecies: (sciName: string, comName: string) => void;
  /** Loading state for list toggles */
  toggleLoading: string | null;
}

/**
 * Table component for species management with sorting, filtering, and toggle icons.
 */
export function SpeciesTable({
  species,
  speciesLists,
  onToggleList,
  onDeleteSpecies,
  toggleLoading,
}: SpeciesTableProps): JSX.Element {
  // Filter state - persist in localStorage
  const [filter, setFilter] = useState(() => {
    try {
      return localStorage.getItem('speciesFilter') || '';
    } catch {
      return '';
    }
  });

  // Sort state - persist in localStorage
  const [sortColumn, setSortColumn] = useState<SortColumn>(() => {
    try {
      return (localStorage.getItem('speciesSortCol') as SortColumn) || 'detection_count';
    } catch {
      return 'detection_count';
    }
  });

  const [sortDirection, setSortDirection] = useState<SortDirection>(() => {
    try {
      return (localStorage.getItem('speciesSortDir') as SortDirection) || 'desc';
    } catch {
      return 'desc';
    }
  });

  // Mini chart state
  const [chartSpecies, setChartSpecies] = useState<{ sciName: string; comName: string } | null>(null);

  // Persist filter to localStorage
  useEffect(() => {
    try {
      localStorage.setItem('speciesFilter', filter);
    } catch {
      // Ignore localStorage errors
    }
  }, [filter]);

  // Persist sort to localStorage
  useEffect(() => {
    try {
      localStorage.setItem('speciesSortCol', sortColumn);
      localStorage.setItem('speciesSortDir', sortDirection);
    } catch {
      // Ignore localStorage errors
    }
  }, [sortColumn, sortDirection]);

  // Handle column header click for sorting
  const handleSort = useCallback((column: SortColumn) => {
    setSortColumn((prev) => {
      if (prev === column) {
        setSortDirection((dir) => (dir === 'asc' ? 'desc' : 'asc'));
        return column;
      }
      setSortDirection('desc');
      return column;
    });
  }, []);

  // Check if species is in a list
  const isInList = useCallback(
    (sciName: string, listType: 'confirmed' | 'excluded' | 'whitelisted'): boolean => {
      const list = speciesLists[listType];
      if (!list) return false;
      // Check both sci_name and sci_name_com_name format
      return list.some((item) => item === sciName || item.startsWith(sciName + '_'));
    },
    [speciesLists]
  );

  // Filter and sort species
  const filteredAndSorted = useMemo(() => {
    const lowerFilter = filter.toLowerCase();

    // Filter
    let result = species.filter((s) => {
      if (!filter) return true;
      return (
        s.com_name.toLowerCase().includes(lowerFilter) ||
        s.sci_name.toLowerCase().includes(lowerFilter)
      );
    });

    // Sort
    result = [...result].sort((a, b) => {
      let aVal: string | number;
      let bVal: string | number;

      switch (sortColumn) {
        case 'com_name':
          aVal = a.com_name.toLowerCase();
          bVal = b.com_name.toLowerCase();
          break;
        case 'sci_name':
          aVal = a.sci_name.toLowerCase();
          bVal = b.sci_name.toLowerCase();
          break;
        case 'detection_count':
          aVal = a.detection_count;
          bVal = b.detection_count;
          break;
        case 'max_confidence':
          aVal = a.max_confidence;
          bVal = b.max_confidence;
          break;
        case 'last_seen':
          aVal = a.last_seen || '';
          bVal = b.last_seen || '';
          break;
        default:
          return 0;
      }

      if (aVal < bVal) return sortDirection === 'asc' ? -1 : 1;
      if (aVal > bVal) return sortDirection === 'asc' ? 1 : -1;
      return 0;
    });

    return result;
  }, [species, filter, sortColumn, sortDirection]);

  // Sort indicator component
  const SortIndicator = ({ column }: { column: SortColumn }) => {
    if (sortColumn !== column) return null;
    return (
      <span class="ml-1">
        {sortDirection === 'asc' ? (
          <svg class="w-4 h-4 inline" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 15l7-7 7 7" />
          </svg>
        ) : (
          <svg class="w-4 h-4 inline" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
          </svg>
        )}
      </span>
    );
  };

  // Toggle icon component
  const ToggleIcon = ({
    isActive,
    onClick,
    isLoading,
    title,
  }: {
    isActive: boolean;
    onClick: () => void;
    isLoading: boolean;
    title: string;
  }) => {
    if (isLoading) {
      return (
        <div class="w-4 h-4 animate-spin rounded-full border-2 border-gray-300 border-t-primary-500" />
      );
    }
    return (
      <button
        onClick={onClick}
        title={title}
        class="w-4 h-4 flex items-center justify-center transition-colors"
      >
        {isActive ? (
          <svg class="w-4 h-4 text-green-600" fill="currentColor" viewBox="0 0 20 20">
            <path
              fill-rule="evenodd"
              d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
              clip-rule="evenodd"
            />
          </svg>
        ) : (
          <span class="w-3 h-3 border border-gray-400 rounded-full hover:border-gray-600 dark:border-gray-500 dark:hover:border-gray-300" />
        )}
      </button>
    );
  };

  return (
    <div class="space-y-4">
      {/* Filter input */}
      <div class="flex items-center gap-4">
        <input
          type="text"
          placeholder="Filter species... (name, scientific)"
          value={filter}
          onInput={(e) => setFilter((e.target as HTMLInputElement).value)}
          class="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg
                 bg-white dark:bg-gray-800 text-gray-900 dark:text-white
                 focus:outline-none focus:ring-2 focus:ring-primary-500"
        />
        <span class="text-sm text-gray-500 dark:text-gray-400">
          {filteredAndSorted.length} / {species.length}
        </span>
      </div>

      {/* Table */}
      <div class="overflow-x-auto rounded-lg border border-gray-200 dark:border-gray-700">
        <table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
          <thead class="bg-gray-50 dark:bg-gray-800">
            <tr>
              <th
                class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-700"
                onClick={() => handleSort('com_name')}
              >
                Common Name
                <SortIndicator column="com_name" />
              </th>
              <th
                class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-700"
                onClick={() => handleSort('sci_name')}
              >
                Scientific Name
                <SortIndicator column="sci_name" />
              </th>
              <th class="px-4 py-3 text-center text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Stats
              </th>
              <th
                class="px-4 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-700"
                onClick={() => handleSort('detection_count')}
              >
                Count
                <SortIndicator column="detection_count" />
              </th>
              <th
                class="px-4 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-700"
                onClick={() => handleSort('max_confidence')}
              >
                Max Conf
                <SortIndicator column="max_confidence" />
              </th>
              <th
                class="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-700"
                onClick={() => handleSort('last_seen')}
              >
                Last Seen
                <SortIndicator column="last_seen" />
              </th>
              <th class="px-4 py-3 text-center text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Confirmed
              </th>
              <th class="px-4 py-3 text-center text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Excluded
              </th>
              <th class="px-4 py-3 text-center text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Whitelisted
              </th>
              <th class="px-4 py-3 text-center text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Delete
              </th>
            </tr>
          </thead>
          <tbody class="bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-800">
            {filteredAndSorted.length === 0 ? (
              <tr>
                <td colSpan={10} class="px-4 py-8 text-center text-gray-500 dark:text-gray-400">
                  {filter ? 'No species match the filter' : 'No species detected'}
                </td>
              </tr>
            ) : (
              filteredAndSorted.map((s) => {
                const isConfirmed = isInList(s.sci_name, 'confirmed');
                const isExcluded = isInList(s.sci_name, 'excluded');
                const isWhitelisted = isInList(s.sci_name, 'whitelisted');
                const isToggling = toggleLoading === s.sci_name;

                return (
                  <tr key={s.sci_name} class="hover:bg-gray-50 dark:hover:bg-gray-800/50">
                    <td class="px-4 py-3 whitespace-nowrap">
                      <a
                        href={`/app/detections?species=${encodeURIComponent(s.sci_name)}`}
                        class="text-gray-900 dark:text-white hover:text-primary-600 dark:hover:text-primary-400"
                      >
                        {s.com_name}
                      </a>
                    </td>
                    <td class="px-4 py-3 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400 italic">
                      {s.sci_name}
                    </td>
                    <td class="px-4 py-3 whitespace-nowrap text-center">
                      <button
                        onClick={() => setChartSpecies({ sciName: s.sci_name, comName: s.com_name })}
                        class="text-gray-400 hover:text-primary-600 dark:hover:text-primary-400 transition-colors"
                        title="View detection history"
                      >
                        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path
                            stroke-linecap="round"
                            stroke-linejoin="round"
                            stroke-width="2"
                            d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"
                          />
                        </svg>
                      </button>
                    </td>
                    <td class="px-4 py-3 whitespace-nowrap text-right text-sm text-gray-900 dark:text-white">
                      {s.detection_count}
                    </td>
                    <td class="px-4 py-3 whitespace-nowrap text-right text-sm text-gray-900 dark:text-white">
                      {(s.max_confidence * 100).toFixed(1)}%
                    </td>
                    <td class="px-4 py-3 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                      {s.last_seen || '-'}
                    </td>
                    <td class="px-4 py-3 whitespace-nowrap text-center">
                      <ToggleIcon
                        isActive={isConfirmed}
                        onClick={() => onToggleList('confirmed', s.sci_name, isConfirmed ? 'remove' : 'add')}
                        isLoading={isToggling}
                        title={isConfirmed ? 'Remove from confirmed' : 'Add to confirmed'}
                      />
                    </td>
                    <td class="px-4 py-3 whitespace-nowrap text-center">
                      <ToggleIcon
                        isActive={isExcluded}
                        onClick={() => onToggleList('excluded', s.sci_name, isExcluded ? 'remove' : 'add')}
                        isLoading={isToggling}
                        title={isExcluded ? 'Remove from excluded' : 'Add to excluded'}
                      />
                    </td>
                    <td class="px-4 py-3 whitespace-nowrap text-center">
                      <ToggleIcon
                        isActive={isWhitelisted}
                        onClick={() => onToggleList('whitelisted', s.sci_name, isWhitelisted ? 'remove' : 'add')}
                        isLoading={isToggling}
                        title={isWhitelisted ? 'Remove from whitelist' : 'Add to whitelist'}
                      />
                    </td>
                    <td class="px-4 py-3 whitespace-nowrap text-center">
                      <button
                        onClick={() => onDeleteSpecies(s.sci_name, s.com_name)}
                        class="text-gray-400 hover:text-red-600 dark:hover:text-red-400 transition-colors"
                        title="Delete all detections for this species"
                      >
                        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path
                            stroke-linecap="round"
                            stroke-linejoin="round"
                            stroke-width="2"
                            d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                          />
                        </svg>
                      </button>
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>

      {/* Mini Chart Modal */}
      {chartSpecies && (
        <SpeciesMiniChart
          sciName={chartSpecies.sciName}
          comName={chartSpecies.comName}
          onClose={() => setChartSpecies(null)}
        />
      )}
    </div>
  );
}
