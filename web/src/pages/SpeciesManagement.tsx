import { useState, useEffect, useCallback } from 'preact/hooks';
import type { JSX } from 'preact';
import {
  fetchAllSpecies,
  fetchSpeciesLists,
  fetchSpeciesCount,
  deleteAllSpeciesDetections,
  addToSpeciesList,
  removeFromSpeciesList,
} from '../hooks/useApi';
import type { Species, SpeciesListsResponse, SpeciesListType } from '../types/api';
import { SpeciesTable } from '../components/SpeciesTable';
import { SpeciesListEditor } from '../components/SpeciesListEditor';

/**
 * Species Management page component.
 * Displays all detected species with management controls.
 */
export function SpeciesManagement(): JSX.Element {
  const [species, setSpecies] = useState<Species[]>([]);
  const [speciesLists, setSpeciesLists] = useState<SpeciesListsResponse>({
    confirmed: [],
    excluded: [],
    whitelisted: [],
    include: [],
  });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [toggleLoading, setToggleLoading] = useState<string | null>(null);
  const [listEditorOpen, setListEditorOpen] = useState<SpeciesListType | null>(null);

  // Delete confirmation state
  const [deleteConfirm, setDeleteConfirm] = useState<{
    sciName: string;
    comName: string;
    detectionCount: number;
    fileCount: number;
  } | null>(null);
  const [deleting, setDeleting] = useState(false);

  // Load data
  const loadData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const [speciesData, listsData] = await Promise.all([
        fetchAllSpecies(),
        fetchSpeciesLists(),
      ]);
      setSpecies(speciesData.species || []);
      setSpeciesLists(listsData);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load data');
    } finally {
      setLoading(false);
    }
  }, []);

  // Load on mount
  useEffect(() => {
    loadData();
  }, [loadData]);

  // Handle toggle list (confirmed/excluded/whitelisted)
  const handleToggleList = useCallback(
    async (listType: 'confirmed' | 'excluded' | 'whitelisted', sciName: string, action: 'add' | 'remove') => {
      try {
        setToggleLoading(sciName);
        if (action === 'add') {
          await addToSpeciesList(listType, sciName);
        } else {
          await removeFromSpeciesList(listType, sciName);
        }
        // Refresh lists
        const listsData = await fetchSpeciesLists();
        setSpeciesLists(listsData);
      } catch (err) {
        console.error('Failed to toggle species:', err);
      } finally {
        setToggleLoading(null);
      }
    },
    []
  );

  // Handle delete species - show confirmation
  const handleDeleteSpecies = useCallback(async (sciName: string, comName: string) => {
    try {
      const counts = await fetchSpeciesCount(sciName);
      setDeleteConfirm({
        sciName,
        comName,
        detectionCount: counts.detection_count,
        fileCount: counts.file_count,
      });
    } catch (err) {
      console.error('Failed to get species count:', err);
      alert('Failed to get species count. Please try again.');
    }
  }, []);

  // Confirm delete
  const confirmDelete = useCallback(async () => {
    if (!deleteConfirm) return;
    try {
      setDeleting(true);
      const result = await deleteAllSpeciesDetections(deleteConfirm.sciName);
      alert(
        `Deleted ${result.detections_deleted} detection(s) and ${result.files_deleted} file(s) for ${deleteConfirm.comName}`
      );
      setDeleteConfirm(null);
      // Refresh data
      loadData();
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to delete species');
    } finally {
      setDeleting(false);
    }
  }, [deleteConfirm, loadData]);

  // Handle list editor save
  const handleListEditorSave = useCallback(async () => {
    setListEditorOpen(null);
    // Refresh lists
    try {
      const listsData = await fetchSpeciesLists();
      setSpeciesLists(listsData);
    } catch (err) {
      console.error('Failed to refresh lists:', err);
    }
  }, []);

  return (
    <div class="space-y-6">
      {/* Header */}
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Species Management</h1>
          <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
            Manage detected species and configure include/exclude lists
          </p>
        </div>
      </div>

      {/* List Management Cards */}
      <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
        {/* Include List */}
        <div class="card p-4">
          <div class="flex items-center justify-between">
            <div>
              <h3 class="font-medium text-gray-900 dark:text-white">Include List</h3>
              <p class="text-sm text-gray-500 dark:text-gray-400">
                {speciesLists.include.length} species
              </p>
            </div>
            <button
              onClick={() => setListEditorOpen('include')}
              class="text-sm text-primary-600 hover:text-primary-700 dark:text-primary-400"
            >
              Edit
            </button>
          </div>
          {speciesLists.include.length > 0 && (
            <div class="mt-2 p-2 bg-yellow-50 dark:bg-yellow-900/30 rounded text-xs text-yellow-800 dark:text-yellow-200">
              Active: Only these species will be detected
            </div>
          )}
        </div>

        {/* Exclude List */}
        <div class="card p-4">
          <div class="flex items-center justify-between">
            <div>
              <h3 class="font-medium text-gray-900 dark:text-white">Exclude List</h3>
              <p class="text-sm text-gray-500 dark:text-gray-400">
                {speciesLists.excluded.length} species
              </p>
            </div>
            <button
              onClick={() => setListEditorOpen('excluded')}
              class="text-sm text-primary-600 hover:text-primary-700 dark:text-primary-400"
            >
              Edit
            </button>
          </div>
        </div>

        {/* Whitelist */}
        <div class="card p-4">
          <div class="flex items-center justify-between">
            <div>
              <h3 class="font-medium text-gray-900 dark:text-white">Whitelist</h3>
              <p class="text-sm text-gray-500 dark:text-gray-400">
                {speciesLists.whitelisted.length} species
              </p>
            </div>
            <button
              onClick={() => setListEditorOpen('whitelisted')}
              class="text-sm text-primary-600 hover:text-primary-700 dark:text-primary-400"
            >
              Edit
            </button>
          </div>
        </div>

        {/* Confirmed */}
        <div class="card p-4">
          <div class="flex items-center justify-between">
            <div>
              <h3 class="font-medium text-gray-900 dark:text-white">Confirmed</h3>
              <p class="text-sm text-gray-500 dark:text-gray-400">
                {speciesLists.confirmed.length} species
              </p>
            </div>
            <button
              onClick={() => setListEditorOpen('confirmed')}
              class="text-sm text-primary-600 hover:text-primary-700 dark:text-primary-400"
            >
              Edit
            </button>
          </div>
        </div>
      </div>

      {/* Species Table Card */}
      <div class="card">
        <div class="p-4 border-b border-gray-200 dark:border-gray-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Detected Species</h2>
          <p class="text-sm text-gray-500 dark:text-gray-400">
            All species detected by the system. Click column headers to sort.
          </p>
        </div>

        <div class="p-4">
          {loading ? (
            <div class="flex items-center justify-center h-64">
              <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
            </div>
          ) : error ? (
            <div class="text-center py-8">
              <p class="text-red-600 dark:text-red-400 mb-4">{error}</p>
              <button onClick={loadData} class="btn btn-primary">
                Retry
              </button>
            </div>
          ) : (
            <SpeciesTable
              species={species}
              speciesLists={speciesLists}
              onToggleList={handleToggleList}
              onDeleteSpecies={handleDeleteSpecies}
              toggleLoading={toggleLoading}
            />
          )}
        </div>
      </div>

      {/* List Editor Modal */}
      {listEditorOpen && (
        <SpeciesListEditor
          listType={listEditorOpen}
          currentList={speciesLists[listEditorOpen]}
          onClose={() => setListEditorOpen(null)}
          onSave={handleListEditorSave}
        />
      )}

      {/* Delete Confirmation Modal */}
      {deleteConfirm && (
        <div
          class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
          onClick={(e) => {
            if ((e.target as HTMLElement).classList.contains('fixed')) {
              setDeleteConfirm(null);
            }
          }}
        >
          <div class="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-md w-full mx-4 p-6">
            <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-2">
              Delete {deleteConfirm.comName}?
            </h3>
            <p class="text-gray-600 dark:text-gray-400 mb-4">
              This will permanently delete:
            </p>
            <ul class="list-disc list-inside text-gray-600 dark:text-gray-400 mb-4 space-y-1">
              <li>{deleteConfirm.detectionCount} detection(s) from the database</li>
              <li>{deleteConfirm.fileCount} audio and spectrogram file(s)</li>
            </ul>
            <p class="text-red-600 dark:text-red-400 text-sm mb-4">
              This action cannot be undone.
            </p>
            <div class="flex justify-end gap-2">
              <button
                onClick={() => setDeleteConfirm(null)}
                disabled={deleting}
                class="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 rounded
                       hover:bg-gray-200 dark:hover:bg-gray-600 disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                onClick={confirmDelete}
                disabled={deleting}
                class="px-4 py-2 text-sm font-medium text-white bg-red-600 rounded
                       hover:bg-red-700 disabled:opacity-50"
              >
                {deleting ? 'Deleting...' : 'Delete'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
