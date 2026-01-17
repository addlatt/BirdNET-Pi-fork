import { useState, useEffect, useCallback } from 'preact/hooks';
import type { JSX } from 'preact';
import {
  fetchRecordingDates,
  fetchRecordingSpecies,
  fetchRecordingsByDate,
  fetchRecordingsBySpecies,
  deleteRecording,
  changeRecordingIdentification,
  toggleRecordingLock,
  toggleRecordingShift,
  fetchLabels,
} from '../hooks/useApi';
import type {
  RecordingSpecies,
  RecordingFile,
  ListRecordingSpeciesParams,
  ListRecordingFilesParams,
} from '../types/api';
import { EnhancedAudioPlayer } from '../components/EnhancedAudioPlayer';

/**
 * View type for the recordings browser
 */
type ViewType = 'choose' | 'by-species' | 'by-date' | 'species-files' | 'date-species';

/**
 * Sort option for species list
 */
type SortOption = 'alphabetical' | 'occurrences' | 'confidence' | 'date';

/**
 * Recordings browser page component.
 * Provides navigation hierarchy to browse bird recordings by species or date.
 */
export function Recordings(): JSX.Element {
  // Navigation state
  const [view, setView] = useState<ViewType>('choose');
  const [selectedDate, setSelectedDate] = useState<string | null>(null);
  const [selectedSpecies, setSelectedSpecies] = useState<string | null>(null);
  const [selectedComName, setSelectedComName] = useState<string | null>(null);

  // Data state
  const [dates, setDates] = useState<string[]>([]);
  const [species, setSpecies] = useState<RecordingSpecies[]>([]);
  const [files, setFiles] = useState<RecordingFile[]>([]);
  const [labels, setLabels] = useState<string[]>([]);

  // UI state
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [sortBy, setSortBy] = useState<SortOption>('occurrences');
  const [fileSort, setFileSort] = useState<'date' | 'confidence'>('date');
  const [onlyLocked, setOnlyLocked] = useState(false);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const perPage = 20;

  // Modal state for changing identification
  const [showChangeModal, setShowChangeModal] = useState(false);
  const [changeTarget, setChangeTarget] = useState<RecordingFile | null>(null);
  const [newSpeciesValue, setNewSpeciesValue] = useState('');
  const [searchFilter, setSearchFilter] = useState('');

  // Load labels for identification change modal
  useEffect(() => {
    async function loadLabels() {
      try {
        const data = await fetchLabels();
        setLabels(data.labels || []);
      } catch {
        // Ignore errors - labels are optional
      }
    }
    loadLabels();
  }, []);

  // Load dates
  const loadDates = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await fetchRecordingDates(365);
      setDates(data.dates || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load dates');
    } finally {
      setLoading(false);
    }
  }, []);

  // Load species list (all or by date)
  const loadSpecies = useCallback(async (date?: string) => {
    setLoading(true);
    setError(null);
    try {
      const params: ListRecordingSpeciesParams = { sort: sortBy };
      if (date) params.date = date;
      const data = date
        ? await fetchRecordingsByDate(date)
        : await fetchRecordingSpecies(params);
      setSpecies(data.species || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load species');
    } finally {
      setLoading(false);
    }
  }, [sortBy]);

  // Load files for a species
  const loadFiles = useCallback(async (sciName: string) => {
    setLoading(true);
    setError(null);
    try {
      const params: ListRecordingFilesParams = {
        sort: fileSort,
        only_locked: onlyLocked,
        page,
        limit: perPage,
      };
      if (selectedDate) params.date = selectedDate;
      const data = await fetchRecordingsBySpecies(sciName, params);
      setFiles(data.files || []);
      setTotal(data.total || 0);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load files');
    } finally {
      setLoading(false);
    }
  }, [selectedDate, fileSort, onlyLocked, page]);

  // Handle view changes
  const handleViewBySpecies = useCallback(() => {
    setView('by-species');
    setSelectedDate(null);
    loadSpecies();
  }, [loadSpecies]);

  const handleViewByDate = useCallback(() => {
    setView('by-date');
    setSelectedSpecies(null);
    loadDates();
  }, [loadDates]);

  const handleSelectDate = useCallback((date: string) => {
    setSelectedDate(date);
    setView('date-species');
    loadSpecies(date);
  }, [loadSpecies]);

  const handleSelectSpecies = useCallback((sciName: string, comName: string) => {
    setSelectedSpecies(sciName);
    setSelectedComName(comName);
    setView('species-files');
    setPage(1);
    loadFiles(sciName);
  }, [loadFiles]);

  const handleBack = useCallback(() => {
    if (view === 'species-files') {
      if (selectedDate) {
        setView('date-species');
        setSelectedSpecies(null);
        setSelectedComName(null);
        setFiles([]);
      } else {
        setView('by-species');
        setSelectedSpecies(null);
        setSelectedComName(null);
        setFiles([]);
      }
    } else if (view === 'date-species') {
      setView('by-date');
      setSelectedDate(null);
      setSpecies([]);
    } else if (view === 'by-species' || view === 'by-date') {
      setView('choose');
      setSpecies([]);
      setDates([]);
    }
  }, [view, selectedDate]);

  // Reload files when sort/filter changes
  useEffect(() => {
    if (view === 'species-files' && selectedSpecies) {
      loadFiles(selectedSpecies);
    }
  }, [view, selectedSpecies, fileSort, onlyLocked, page, loadFiles]);

  // Reload species when sort changes
  useEffect(() => {
    if (view === 'by-species') {
      loadSpecies();
    } else if (view === 'date-species' && selectedDate) {
      loadSpecies(selectedDate);
    }
  }, [sortBy]);

  // Handle file actions
  const handleDelete = useCallback(async (file: RecordingFile) => {
    if (!confirm(`Delete recording of ${file.com_name}?`)) return;
    try {
      await deleteRecording(file.date, file.sci_name, file.file_name);
      setFiles((prev) => prev.filter((f) => f.file_name !== file.file_name));
      setTotal((prev) => Math.max(0, prev - 1));
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Delete failed');
    }
  }, []);

  const handleToggleLock = useCallback(async (file: RecordingFile) => {
    try {
      const result = await toggleRecordingLock(file.date, file.sci_name, file.file_name);
      setFiles((prev) =>
        prev.map((f) =>
          f.file_name === file.file_name ? { ...f, is_locked: result.is_locked } : f
        )
      );
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Toggle lock failed');
    }
  }, []);

  const handleToggleShift = useCallback(async (file: RecordingFile) => {
    try {
      const result = await toggleRecordingShift(file.date, file.sci_name, file.file_name);
      setFiles((prev) =>
        prev.map((f) =>
          f.file_name === file.file_name
            ? { ...f, is_shifted: result.is_shifted, shifted_url: result.shifted_url }
            : f
        )
      );
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Toggle shift failed');
    }
  }, []);

  const handleChangeIdentification = useCallback((file: RecordingFile) => {
    setChangeTarget(file);
    setNewSpeciesValue('');
    setSearchFilter('');
    setShowChangeModal(true);
  }, []);

  const handleConfirmChange = useCallback(async () => {
    if (!changeTarget || !newSpeciesValue) return;
    try {
      await changeRecordingIdentification(
        changeTarget.date,
        changeTarget.sci_name,
        changeTarget.file_name,
        newSpeciesValue
      );
      // Remove from current list since it's now a different species
      setFiles((prev) => prev.filter((f) => f.file_name !== changeTarget.file_name));
      setTotal((prev) => Math.max(0, prev - 1));
      setShowChangeModal(false);
      setChangeTarget(null);
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Change identification failed');
    }
  }, [changeTarget, newSpeciesValue]);

  const totalPages = Math.ceil(total / perPage);

  // Filter labels for the modal
  const filteredLabels = searchFilter
    ? labels.filter((l) => l.toLowerCase().includes(searchFilter.toLowerCase()))
    : labels.slice(0, 50);

  // Format date for display
  const formatDate = (dateStr: string): string => {
    try {
      return new Date(dateStr + 'T00:00:00').toLocaleDateString('en-US', {
        weekday: 'short',
        year: 'numeric',
        month: 'short',
        day: 'numeric',
      });
    } catch {
      return dateStr;
    }
  };

  return (
    <div class="space-y-6">
      {/* Header with navigation */}
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-4">
          {view !== 'choose' && (
            <button
              onClick={handleBack}
              class="p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
              aria-label="Go back"
            >
              <svg class="w-5 h-5 text-gray-600 dark:text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
              </svg>
            </button>
          )}
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">
            {view === 'choose' && 'Recordings'}
            {view === 'by-species' && 'Browse by Species'}
            {view === 'by-date' && 'Browse by Date'}
            {view === 'date-species' && `${formatDate(selectedDate!)}`}
            {view === 'species-files' && selectedComName}
          </h1>
        </div>

        {/* Sort controls for species view */}
        {(view === 'by-species' || view === 'date-species') && (
          <select
            value={sortBy}
            onChange={(e) => setSortBy((e.target as HTMLSelectElement).value as SortOption)}
            class="px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm"
          >
            <option value="occurrences">Most Recordings</option>
            <option value="alphabetical">Alphabetical</option>
            <option value="confidence">Highest Confidence</option>
            <option value="date">Most Recent</option>
          </select>
        )}

        {/* Sort/filter controls for files view */}
        {view === 'species-files' && (
          <div class="flex items-center gap-3">
            <label class="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
              <input
                type="checkbox"
                checked={onlyLocked}
                onChange={(e) => {
                  setOnlyLocked((e.target as HTMLInputElement).checked);
                  setPage(1);
                }}
                class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              />
              Locked only
            </label>
            <select
              value={fileSort}
              onChange={(e) => {
                setFileSort((e.target as HTMLSelectElement).value as 'date' | 'confidence');
                setPage(1);
              }}
              class="px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white text-sm"
            >
              <option value="date">Newest First</option>
              <option value="confidence">Highest Confidence</option>
            </select>
          </div>
        )}
      </div>

      {/* Error message */}
      {error && (
        <div class="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
          <p class="text-red-600 dark:text-red-400">{error}</p>
        </div>
      )}

      {/* Loading spinner */}
      {loading && (
        <div class="flex items-center justify-center h-64">
          <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
        </div>
      )}

      {/* Choose view */}
      {!loading && view === 'choose' && (
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-6">
          <button
            onClick={handleViewBySpecies}
            class="card p-8 text-left hover:shadow-lg transition-shadow"
          >
            <div class="flex items-center gap-4">
              <div class="p-4 rounded-full bg-primary-100 dark:bg-primary-900">
                <svg class="w-8 h-8 text-primary-600 dark:text-primary-400" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
                </svg>
              </div>
              <div>
                <h2 class="text-xl font-semibold text-gray-900 dark:text-white">By Species</h2>
                <p class="text-gray-500 dark:text-gray-400 mt-1">Browse all recordings organized by bird species</p>
              </div>
            </div>
          </button>

          <button
            onClick={handleViewByDate}
            class="card p-8 text-left hover:shadow-lg transition-shadow"
          >
            <div class="flex items-center gap-4">
              <div class="p-4 rounded-full bg-blue-100 dark:bg-blue-900">
                <svg class="w-8 h-8 text-blue-600 dark:text-blue-400" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M19 3h-1V1h-2v2H8V1H6v2H5c-1.11 0-1.99.9-1.99 2L3 19c0 1.1.89 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H5V8h14v11zM9 10H7v2h2v-2zm4 0h-2v2h2v-2zm4 0h-2v2h2v-2zm-8 4H7v2h2v-2zm4 0h-2v2h2v-2zm4 0h-2v2h2v-2z"/>
                </svg>
              </div>
              <div>
                <h2 class="text-xl font-semibold text-gray-900 dark:text-white">By Date</h2>
                <p class="text-gray-500 dark:text-gray-400 mt-1">Browse recordings by date they were captured</p>
              </div>
            </div>
          </button>
        </div>
      )}

      {/* Date list view */}
      {!loading && view === 'by-date' && (
        <div class="card divide-y divide-gray-200 dark:divide-gray-700">
          {dates.length === 0 ? (
            <div class="p-8 text-center text-gray-500 dark:text-gray-400">
              No recordings found
            </div>
          ) : (
            dates.map((date) => (
              <button
                key={date}
                onClick={() => handleSelectDate(date)}
                class="w-full p-4 text-left hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors flex items-center justify-between"
              >
                <span class="font-medium text-gray-900 dark:text-white">{formatDate(date)}</span>
                <svg class="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
                </svg>
              </button>
            ))
          )}
        </div>
      )}

      {/* Species list view */}
      {!loading && (view === 'by-species' || view === 'date-species') && (
        <div class="card divide-y divide-gray-200 dark:divide-gray-700">
          {species.length === 0 ? (
            <div class="p-8 text-center text-gray-500 dark:text-gray-400">
              No species found
            </div>
          ) : (
            species.map((sp) => (
              <button
                key={sp.sci_name}
                onClick={() => handleSelectSpecies(sp.sci_name, sp.com_name)}
                class="w-full p-4 text-left hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
              >
                <div class="flex items-center justify-between">
                  <div>
                    <span class="font-medium text-gray-900 dark:text-white">{sp.com_name}</span>
                    <span class="text-gray-500 dark:text-gray-400 text-sm ml-2">({sp.sci_name})</span>
                  </div>
                  <div class="flex items-center gap-4">
                    <span class="text-sm text-gray-500 dark:text-gray-400">
                      {sp.detection_count} recording{sp.detection_count !== 1 ? 's' : ''}
                    </span>
                    <span class="px-2 py-1 text-xs rounded-full bg-primary-100 dark:bg-primary-900 text-primary-700 dark:text-primary-300">
                      {Math.round(sp.max_confidence * 100)}%
                    </span>
                    <svg class="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
                    </svg>
                  </div>
                </div>
              </button>
            ))
          )}
        </div>
      )}

      {/* Files view */}
      {!loading && view === 'species-files' && (
        <>
          {/* Summary */}
          <div class="text-sm text-gray-500 dark:text-gray-400">
            {total} recording{total !== 1 ? 's' : ''}
            {selectedDate && ` on ${formatDate(selectedDate)}`}
            {onlyLocked && ' (locked only)'}
          </div>

          {/* Files grid */}
          {files.length === 0 ? (
            <div class="card p-8 text-center text-gray-500 dark:text-gray-400">
              No recordings found
            </div>
          ) : (
            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {files.map((file) => (
                <div key={file.file_name} class="card overflow-hidden">
                  <EnhancedAudioPlayer
                    audioUrl={file.audio_url}
                    spectrogramUrl={file.spectrogram_url}
                    title={file.com_name}
                    date={file.date}
                    time={file.time}
                    confidence={file.confidence}
                    isLocked={file.is_locked}
                    isShifted={file.is_shifted}
                    shiftedUrl={file.shifted_url}
                    showActions={true}
                    onDelete={() => handleDelete(file)}
                    onToggleLock={() => handleToggleLock(file)}
                    onToggleShift={() => handleToggleShift(file)}
                    onChangeIdentification={() => handleChangeIdentification(file)}
                  />
                </div>
              ))}
            </div>
          )}

          {/* Pagination */}
          {totalPages > 1 && (
            <div class="flex items-center justify-between">
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

      {/* Change identification modal */}
      {showChangeModal && changeTarget && (
        <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div class="bg-white dark:bg-gray-800 rounded-xl shadow-xl max-w-lg w-full max-h-[80vh] flex flex-col">
            <div class="p-4 border-b border-gray-200 dark:border-gray-700">
              <h3 class="text-lg font-semibold text-gray-900 dark:text-white">Change Identification</h3>
              <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
                Current: {changeTarget.com_name}
              </p>
            </div>

            <div class="p-4">
              <input
                type="text"
                placeholder="Search species..."
                value={searchFilter}
                onInput={(e) => setSearchFilter((e.target as HTMLInputElement).value)}
                class="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
              />
            </div>

            <div class="flex-1 overflow-y-auto px-4">
              <div class="space-y-1">
                {filteredLabels.map((label) => (
                  <button
                    key={label}
                    onClick={() => setNewSpeciesValue(label)}
                    class={`w-full text-left px-3 py-2 rounded-lg text-sm ${
                      newSpeciesValue === label
                        ? 'bg-primary-100 dark:bg-primary-900 text-primary-700 dark:text-primary-300'
                        : 'hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-900 dark:text-white'
                    }`}
                  >
                    {label}
                  </button>
                ))}
              </div>
            </div>

            <div class="p-4 border-t border-gray-200 dark:border-gray-700 flex gap-3">
              <button
                onClick={() => setShowChangeModal(false)}
                class="btn btn-secondary flex-1"
              >
                Cancel
              </button>
              <button
                onClick={handleConfirmChange}
                disabled={!newSpeciesValue}
                class="btn btn-primary flex-1"
              >
                Change
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
