import { useState, useEffect } from 'preact/hooks';
import type { JSX } from 'preact';
import { fetchSpeciesDetail } from '../hooks/useApi';
import type { SpeciesDetail as SpeciesDetailType } from '../types/api';
import { AudioPlayer } from './AudioPlayer';

/**
 * SpeciesDetail modal props
 */
interface SpeciesDetailProps {
  /** Name of the species to display */
  speciesName: string;
  /** Callback when modal is closed */
  onClose: () => void;
}

/**
 * SpeciesDetail modal component.
 */
export function SpeciesDetail({ speciesName, onClose }: SpeciesDetailProps): JSX.Element {
  const [detail, setDetail] = useState<SpeciesDetailType | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!speciesName) return;

    async function loadDetail() {
      try {
        setLoading(true);
        setError(null);
        const data = await fetchSpeciesDetail(speciesName);
        setDetail(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
      } finally {
        setLoading(false);
      }
    }

    loadDetail();
  }, [speciesName]);

  // Handle escape key
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [onClose]);

  // Handle click outside
  const handleBackdropClick = (e: JSX.TargetedMouseEvent<HTMLDivElement>) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  return (
    <div
      class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4"
      onClick={handleBackdropClick}
    >
      <div class="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-2xl w-full max-h-[90vh] overflow-y-auto">
        {/* Header */}
        <div class="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700">
          <h2 class="text-xl font-semibold text-gray-900 dark:text-white">
            {speciesName}
          </h2>
          <button
            onClick={onClose}
            class="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
            title="Close"
          >
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Content */}
        <div class="p-4">
          {loading && (
            <div class="flex items-center justify-center py-8">
              <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
            </div>
          )}

          {error && (
            <div class="text-center py-8">
              <p class="text-red-600 dark:text-red-400">{error}</p>
              <button
                class="btn btn-primary mt-4"
                onClick={() => window.location.reload()}
              >
                Retry
              </button>
            </div>
          )}

          {detail && !loading && (
            <div class="space-y-4">
              {/* Species info */}
              <div class="grid grid-cols-2 gap-4">
                <div>
                  <div class="text-sm text-gray-500 dark:text-gray-400">Scientific Name</div>
                  <div class="text-gray-900 dark:text-white italic">{detail.sci_name}</div>
                </div>
                <div>
                  <div class="text-sm text-gray-500 dark:text-gray-400">Occurrences</div>
                  <div class="text-gray-900 dark:text-white">{detail.detection_count}</div>
                </div>
                <div>
                  <div class="text-sm text-gray-500 dark:text-gray-400">Max Confidence</div>
                  <div class="text-gray-900 dark:text-white">
                    {Math.round(detail.max_confidence * 100)}%
                  </div>
                </div>
                <div>
                  <div class="text-sm text-gray-500 dark:text-gray-400">Best Recording</div>
                  <div class="text-gray-900 dark:text-white">
                    {detail.best_date} {detail.best_time}
                  </div>
                </div>
              </div>

              {/* Audio player with spectrogram */}
              <div>
                <div class="text-sm text-gray-500 dark:text-gray-400 mb-2">Best Recording</div>
                <AudioPlayer
                  audioUrl={detail.audio_url}
                  spectrogramUrl={detail.spectrogram_url}
                />
              </div>

              {/* External links */}
              <div class="flex gap-3 pt-2">
                <a
                  href={`https://wikipedia.org/wiki/${encodeURIComponent(detail.sci_name)}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  class="text-primary-600 hover:text-primary-800 dark:text-primary-400 dark:hover:text-primary-300 text-sm flex items-center gap-1"
                >
                  <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
                    <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"/>
                  </svg>
                  Wikipedia
                </a>
                <a
                  href={`https://ebird.org/species/${encodeURIComponent(detail.sci_name.toLowerCase().replace(' ', ''))}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  class="text-primary-600 hover:text-primary-800 dark:text-primary-400 dark:hover:text-primary-300 text-sm flex items-center gap-1"
                >
                  <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
                    <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"/>
                  </svg>
                  eBird
                </a>
                <a
                  href={`https://www.allaboutbirds.org/guide/${encodeURIComponent(detail.com_name.replace(/ /g, '_'))}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  class="text-primary-600 hover:text-primary-800 dark:text-primary-400 dark:hover:text-primary-300 text-sm flex items-center gap-1"
                >
                  <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
                    <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"/>
                  </svg>
                  All About Birds
                </a>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
