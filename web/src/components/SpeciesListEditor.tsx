import type { JSX } from 'preact';
import { useState, useEffect, useMemo, useCallback } from 'preact/hooks';
import type { SpeciesListType } from '../types/api';
import { fetchLabels, updateSpeciesList } from '../hooks/useApi';

/**
 * SpeciesListEditor props
 */
interface SpeciesListEditorProps {
  /** The list type being edited */
  listType: SpeciesListType;
  /** Current species in the list */
  currentList: string[];
  /** Callback when the modal is closed */
  onClose: () => void;
  /** Callback when the list is saved */
  onSave: () => void;
}

/**
 * List info for display
 */
const listInfo: Record<SpeciesListType, { title: string; description: string; warning?: string }> = {
  include: {
    title: 'Include Species List',
    description: 'Species that will ONLY be recognized by the system.',
    warning: 'Warning: If this list contains ANY species, the system will ONLY recognize those species. Keep this list EMPTY unless you are ONLY interested in detecting specific species.',
  },
  excluded: {
    title: 'Excluded Species List',
    description: 'Species that will be excluded from detection.',
  },
  whitelisted: {
    title: 'Whitelisted Species List',
    description: 'Species that will be detected even if below the Species Occurrence Frequency Threshold.',
  },
  confirmed: {
    title: 'Confirmed Species List',
    description: 'Species that have been confirmed to exist in your area.',
  },
};

/**
 * Modal for editing include/exclude/whitelist species lists with a dual-list UI.
 */
export function SpeciesListEditor({
  listType,
  currentList,
  onClose,
  onSave,
}: SpeciesListEditorProps): JSX.Element {
  // State
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [allLabels, setAllLabels] = useState<string[]>([]);
  const [selectedList, setSelectedList] = useState<string[]>(currentList);
  const [searchAvailable, setSearchAvailable] = useState('');
  const [searchSelected, setSearchSelected] = useState('');
  const [selectedAvailable, setSelectedAvailable] = useState<Set<string>>(new Set());
  const [selectedItems, setSelectedItems] = useState<Set<string>>(new Set());

  const info = listInfo[listType];

  // Load labels
  useEffect(() => {
    async function loadLabels() {
      try {
        setLoading(true);
        setError(null);
        const data = await fetchLabels();
        setAllLabels(data.labels || []);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load labels');
      } finally {
        setLoading(false);
      }
    }
    loadLabels();
  }, []);

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

  // Handle click outside to close
  const handleBackdropClick = useCallback(
    (e: MouseEvent) => {
      if ((e.target as HTMLElement).classList.contains('modal-backdrop')) {
        onClose();
      }
    },
    [onClose]
  );

  // Filter available labels
  const availableLabels = useMemo(() => {
    const selectedSet = new Set(selectedList);
    const search = searchAvailable.toLowerCase();
    return allLabels.filter((label) => {
      if (selectedSet.has(label)) return false;
      if (!search) return true;
      return label.toLowerCase().includes(search);
    });
  }, [allLabels, selectedList, searchAvailable]);

  // Filter selected list
  const filteredSelectedList = useMemo(() => {
    const search = searchSelected.toLowerCase();
    if (!search) return selectedList;
    return selectedList.filter((label) => label.toLowerCase().includes(search));
  }, [selectedList, searchSelected]);

  // Add selected items to the list
  const handleAdd = useCallback(() => {
    if (selectedAvailable.size === 0) return;
    setSelectedList((prev) => [...prev, ...Array.from(selectedAvailable)].sort());
    setSelectedAvailable(new Set());
  }, [selectedAvailable]);

  // Remove selected items from the list
  const handleRemove = useCallback(() => {
    if (selectedItems.size === 0) return;
    setSelectedList((prev) => prev.filter((item) => !selectedItems.has(item)));
    setSelectedItems(new Set());
  }, [selectedItems]);

  // Toggle selection in available list
  const toggleAvailable = useCallback((label: string, e: MouseEvent) => {
    setSelectedAvailable((prev) => {
      const next = new Set(prev);
      if (e.ctrlKey || e.metaKey) {
        // Toggle single item
        if (next.has(label)) {
          next.delete(label);
        } else {
          next.add(label);
        }
      } else {
        // Replace selection
        if (next.has(label) && next.size === 1) {
          next.clear();
        } else {
          next.clear();
          next.add(label);
        }
      }
      return next;
    });
  }, []);

  // Toggle selection in selected list
  const toggleSelected = useCallback((label: string, e: MouseEvent) => {
    setSelectedItems((prev) => {
      const next = new Set(prev);
      if (e.ctrlKey || e.metaKey) {
        if (next.has(label)) {
          next.delete(label);
        } else {
          next.add(label);
        }
      } else {
        if (next.has(label) && next.size === 1) {
          next.clear();
        } else {
          next.clear();
          next.add(label);
        }
      }
      return next;
    });
  }, []);

  // Save the list
  const handleSave = useCallback(async () => {
    try {
      setSaving(true);
      setError(null);
      await updateSpeciesList(listType, selectedList);
      onSave();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save list');
    } finally {
      setSaving(false);
    }
  }, [listType, selectedList, onSave]);

  // Check if list has changes
  const hasChanges = useMemo(() => {
    if (selectedList.length !== currentList.length) return true;
    const sortedNew = [...selectedList].sort();
    const sortedOld = [...currentList].sort();
    return !sortedNew.every((v, i) => v === sortedOld[i]);
  }, [selectedList, currentList]);

  return (
    <div
      class="modal-backdrop fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      onClick={handleBackdropClick}
    >
      <div class="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-5xl w-full mx-4 max-h-[90vh] flex flex-col">
        {/* Header */}
        <div class="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700">
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{info.title}</h2>
            <p class="text-sm text-gray-500 dark:text-gray-400">{info.description}</p>
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

        {/* Warning */}
        {info.warning && (
          <div class="mx-4 mt-4 p-3 bg-yellow-50 dark:bg-yellow-900/30 border border-yellow-200 dark:border-yellow-700 rounded-lg">
            <p class="text-sm text-yellow-800 dark:text-yellow-200">{info.warning}</p>
          </div>
        )}

        {/* Error */}
        {error && (
          <div class="mx-4 mt-4 p-3 bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-700 rounded-lg">
            <p class="text-sm text-red-800 dark:text-red-200">{error}</p>
          </div>
        )}

        {/* Content */}
        <div class="flex-1 overflow-hidden p-4">
          {loading ? (
            <div class="flex items-center justify-center h-64">
              <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
            </div>
          ) : (
            <div class="grid grid-cols-[1fr,auto,1fr] gap-4 h-full">
              {/* Available Labels */}
              <div class="flex flex-col min-h-0">
                <label class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Available Species ({availableLabels.length})
                </label>
                <input
                  type="text"
                  placeholder="Search species..."
                  value={searchAvailable}
                  onInput={(e) => setSearchAvailable((e.target as HTMLInputElement).value)}
                  class="mb-2 px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded
                         bg-white dark:bg-gray-700 text-gray-900 dark:text-white
                         focus:outline-none focus:ring-2 focus:ring-primary-500"
                />
                <div class="flex-1 overflow-y-auto border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-900">
                  {availableLabels.slice(0, 500).map((label) => (
                    <div
                      key={label}
                      onClick={(e) => toggleAvailable(label, e)}
                      class={`px-3 py-1 text-sm cursor-pointer truncate ${
                        selectedAvailable.has(label)
                          ? 'bg-primary-100 dark:bg-primary-900 text-primary-900 dark:text-primary-100'
                          : 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800'
                      }`}
                    >
                      {label}
                    </div>
                  ))}
                  {availableLabels.length > 500 && (
                    <div class="px-3 py-2 text-sm text-gray-500 dark:text-gray-400 italic">
                      ... and {availableLabels.length - 500} more (use search to find)
                    </div>
                  )}
                </div>
              </div>

              {/* Buttons */}
              <div class="flex flex-col justify-center gap-2">
                <button
                  onClick={handleAdd}
                  disabled={selectedAvailable.size === 0}
                  class="px-4 py-2 text-sm font-medium bg-primary-600 text-white rounded
                         hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  Add &gt;&gt;
                </button>
                <button
                  onClick={handleRemove}
                  disabled={selectedItems.size === 0}
                  class="px-4 py-2 text-sm font-medium bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded
                         hover:bg-gray-300 dark:hover:bg-gray-600 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  &lt;&lt; Remove
                </button>
              </div>

              {/* Selected List */}
              <div class="flex flex-col min-h-0">
                <label class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  {info.title.replace(' List', '')} ({selectedList.length})
                </label>
                <input
                  type="text"
                  placeholder="Search selected..."
                  value={searchSelected}
                  onInput={(e) => setSearchSelected((e.target as HTMLInputElement).value)}
                  class="mb-2 px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded
                         bg-white dark:bg-gray-700 text-gray-900 dark:text-white
                         focus:outline-none focus:ring-2 focus:ring-primary-500"
                />
                <div class="flex-1 overflow-y-auto border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-900">
                  {filteredSelectedList.length === 0 ? (
                    <div class="px-3 py-8 text-sm text-gray-500 dark:text-gray-400 text-center italic">
                      No species in this list
                    </div>
                  ) : (
                    filteredSelectedList.map((label) => (
                      <div
                        key={label}
                        onClick={(e) => toggleSelected(label, e)}
                        class={`px-3 py-1 text-sm cursor-pointer truncate ${
                          selectedItems.has(label)
                            ? 'bg-primary-100 dark:bg-primary-900 text-primary-900 dark:text-primary-100'
                            : 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800'
                        }`}
                      >
                        {label}
                      </div>
                    ))
                  )}
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <div class="flex items-center justify-between p-4 border-t border-gray-200 dark:border-gray-700">
          <span class="text-sm text-gray-500 dark:text-gray-400">
            {hasChanges ? 'Unsaved changes' : 'No changes'}
          </span>
          <div class="flex gap-2">
            <button
              onClick={onClose}
              class="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 rounded
                     hover:bg-gray-200 dark:hover:bg-gray-600"
            >
              Cancel
            </button>
            <button
              onClick={handleSave}
              disabled={!hasChanges || saving}
              class="px-4 py-2 text-sm font-medium text-white bg-primary-600 rounded
                     hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {saving ? 'Saving...' : 'Save Changes'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
