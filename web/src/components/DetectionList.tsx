import type { JSX } from 'preact';
import { useState, useCallback } from 'preact/hooks';
import type { Detection } from '../types/api';
import { deleteDetection } from '../hooks/useApi';
import { SpeciesMiniChart } from './SpeciesMiniChart';
import { BirdImage } from './BirdImage';

/**
 * DetectionList props
 */
interface DetectionListProps {
  /** Array of detections to display */
  detections: Detection[];
  /** Callback when a detection is deleted */
  onDelete?: (detection: Detection) => void;
}

/**
 * DetectionItem props
 */
interface DetectionItemProps {
  /** Single detection to display */
  detection: Detection;
  /** Callback when deletion is complete */
  onDelete?: (detection: Detection) => void;
}

/**
 * List of detection items.
 */
export function DetectionList({ detections, onDelete }: DetectionListProps): JSX.Element {
  if (!detections || detections.length === 0) {
    return (
      <div class="p-8 text-center text-gray-500 dark:text-gray-400">
        No detections yet
      </div>
    );
  }

  return (
    <div class="divide-y divide-gray-200 dark:divide-gray-700">
      {detections.map((detection) => (
        <DetectionItem
          key={`${detection.date}-${detection.time}-${detection.sci_name}`}
          detection={detection}
          onDelete={onDelete}
        />
      ))}
    </div>
  );
}

/**
 * Single detection item with info links, delete, and chart features.
 */
function DetectionItem({ detection, onDelete }: DetectionItemProps): JSX.Element {
  const [showChart, setShowChart] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [showSpectrogram, setShowSpectrogram] = useState(false);

  const confidencePercent = Math.round(detection.confidence * 100);
  const confidenceColor = getConfidenceColor(confidencePercent);

  // Build species directory name (com_name with spaces as underscores)
  const speciesDir = detection.com_name.replace(/ /g, '_');

  // Extract just the date portion (YYYY-MM-DD) from ISO date string
  const dateOnly = detection.date.split('T')[0];

  // Build file paths for audio and spectrogram
  // Structure: /By_Date/[date]/[species_dir]/[filename]
  const audioPath = `/By_Date/${dateOnly}/${speciesDir}/${detection.file_name}`;
  const spectrogramPath = `${audioPath}.png`;

  // Build external info URLs
  const wikipediaUrl = `https://en.wikipedia.org/wiki/${encodeURIComponent(detection.sci_name.replace(/ /g, '_'))}`;
  const allAboutBirdsUrl = `https://www.allaboutbirds.org/guide/${encodeURIComponent(detection.com_name.replace(/ /g, '_'))}`;
  const eBirdUrl = `https://ebird.org/species/${encodeURIComponent(detection.sci_name.toLowerCase().replace(/ /g, ''))}`;

  // Handle delete confirmation
  const handleDeleteClick = useCallback(() => {
    setShowDeleteConfirm(true);
  }, []);

  // Handle actual deletion
  const handleConfirmDelete = useCallback(async () => {
    try {
      setDeleting(true);
      await deleteDetection(detection.date, detection.time, detection.sci_name);
      setShowDeleteConfirm(false);
      onDelete?.(detection);
    } catch (err) {
      console.error('Failed to delete detection:', err);
      alert(`Failed to delete: ${err instanceof Error ? err.message : 'Unknown error'}`);
    } finally {
      setDeleting(false);
    }
  }, [detection, onDelete]);

  // Cancel delete
  const handleCancelDelete = useCallback(() => {
    setShowDeleteConfirm(false);
  }, []);

  return (
    <>
      <div class="p-4 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors">
        <div class="flex items-start gap-4">
          {/* Bird Image */}
          <BirdImage
            sciName={detection.sci_name}
            comName={detection.com_name}
            size="medium"
          />

          {/* Species Info */}
          <div class="flex-1 min-w-0">
            <div class="flex items-center flex-wrap gap-2">
              <h3 class="font-medium text-gray-900 dark:text-white">
                {detection.com_name}
              </h3>
              <span class="text-sm text-gray-500 dark:text-gray-400 italic">
                {detection.sci_name}
              </span>
            </div>

            {/* Date, Time, Audio Link */}
            <div class="flex items-center mt-1 text-sm text-gray-500 dark:text-gray-400 flex-wrap gap-x-2">
              <span>{dateOnly}</span>
              <span>-</span>
              <span>{detection.time}</span>
              {detection.file_name && (
                <>
                  <span>-</span>
                  <a
                    href={audioPath}
                    class="text-primary-600 hover:underline"
                    title="Play audio"
                  >
                    Play
                  </a>
                  <span>-</span>
                  <button
                    onClick={() => setShowSpectrogram(!showSpectrogram)}
                    class="text-primary-600 hover:underline"
                    title="Show spectrogram"
                  >
                    {showSpectrogram ? 'Hide' : 'Show'} Spectrogram
                  </button>
                </>
              )}
            </div>

            {/* Spectrogram Image */}
            {showSpectrogram && detection.file_name && (
              <div class="mt-2">
                <img
                  src={spectrogramPath}
                  alt={`Spectrogram for ${detection.com_name}`}
                  class="max-w-full rounded border border-gray-200 dark:border-gray-700"
                  loading="lazy"
                />
              </div>
            )}

            {/* Info Links */}
            <div class="flex items-center mt-2 gap-2">
              <a
                href={wikipediaUrl}
                target="_blank"
                rel="noopener noreferrer"
                class="inline-flex items-center px-2 py-1 text-xs font-medium rounded
                       bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300
                       hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors"
                title="View on Wikipedia"
              >
                <WikiIcon class="w-3 h-3 mr-1" />
                Wiki
              </a>
              <a
                href={allAboutBirdsUrl}
                target="_blank"
                rel="noopener noreferrer"
                class="inline-flex items-center px-2 py-1 text-xs font-medium rounded
                       bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300
                       hover:bg-blue-200 dark:hover:bg-blue-900/50 transition-colors"
                title="View on All About Birds"
              >
                <BirdIcon class="w-3 h-3 mr-1" />
                AllAboutBirds
              </a>
              <a
                href={eBirdUrl}
                target="_blank"
                rel="noopener noreferrer"
                class="inline-flex items-center px-2 py-1 text-xs font-medium rounded
                       bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300
                       hover:bg-green-200 dark:hover:bg-green-900/50 transition-colors"
                title="View on eBird"
              >
                <EBirdIcon class="w-3 h-3 mr-1" />
                eBird
              </a>
              <button
                onClick={() => setShowChart(true)}
                class="inline-flex items-center px-2 py-1 text-xs font-medium rounded
                       bg-purple-100 dark:bg-purple-900/30 text-purple-700 dark:text-purple-300
                       hover:bg-purple-200 dark:hover:bg-purple-900/50 transition-colors"
                title="View detection history"
              >
                <ChartIcon class="w-3 h-3 mr-1" />
                History
              </button>
            </div>
          </div>

          {/* Confidence + Actions */}
          <div class="flex items-center gap-4">
            {/* Confidence Bar */}
            <div class="flex items-center">
              <div class="w-20 h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
                <div
                  class={`h-full ${confidenceColor}`}
                  style={{ width: `${confidencePercent}%` }}
                />
              </div>
              <span class="ml-2 text-sm font-medium text-gray-700 dark:text-gray-300 w-12 text-right">
                {confidencePercent}%
              </span>
            </div>

            {/* Delete Button */}
            {onDelete && (
              <button
                onClick={handleDeleteClick}
                disabled={deleting}
                class="p-2 text-gray-400 hover:text-red-600 dark:hover:text-red-400
                       disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                title="Delete detection"
              >
                <TrashIcon class="w-5 h-5" />
              </button>
            )}
          </div>
        </div>

        {/* Delete Confirmation */}
        {showDeleteConfirm && (
          <div class="mt-3 p-3 bg-red-50 dark:bg-red-900/20 rounded-lg border border-red-200 dark:border-red-800">
            <p class="text-sm text-red-700 dark:text-red-300 mb-2">
              Delete this detection? This action cannot be undone.
            </p>
            <div class="flex gap-2">
              <button
                onClick={handleConfirmDelete}
                disabled={deleting}
                class="px-3 py-1 text-sm font-medium text-white bg-red-600 rounded
                       hover:bg-red-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {deleting ? 'Deleting...' : 'Delete'}
              </button>
              <button
                onClick={handleCancelDelete}
                disabled={deleting}
                class="px-3 py-1 text-sm font-medium text-gray-700 dark:text-gray-300
                       bg-gray-200 dark:bg-gray-700 rounded hover:bg-gray-300 dark:hover:bg-gray-600"
              >
                Cancel
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Species Mini-Chart Modal */}
      {showChart && (
        <SpeciesMiniChart
          sciName={detection.sci_name}
          comName={detection.com_name}
          onClose={() => setShowChart(false)}
        />
      )}
    </>
  );
}

/**
 * Get color class based on confidence percentage.
 */
function getConfidenceColor(percent: number): string {
  if (percent >= 90) return 'bg-green-500';
  if (percent >= 70) return 'bg-green-400';
  if (percent >= 50) return 'bg-yellow-400';
  return 'bg-orange-400';
}

// =============================================================================
// Icon Components
// =============================================================================

function WikiIcon({ class: className }: { class?: string }): JSX.Element {
  return (
    <svg class={className} viewBox="0 0 24 24" fill="currentColor">
      <path d="M12.09 13.119c-.936 1.932-2.217 4.548-2.853 5.728-.616 1.074-1.127.931-1.532.029-1.406-3.321-4.293-9.144-5.651-12.409-.251-.601-.441-.987-.619-1.139-.181-.15-.554-.24-1.122-.271C.103 5.033 0 4.982 0 4.898v-.455l.052-.045c.924-.005 5.401 0 5.401 0l.051.045v.434c0 .119-.075.176-.225.176l-.564.031c-.485.029-.727.164-.727.436 0 .135.053.33.166.601 1.082 2.646 4.818 10.521 4.818 10.521l.136.046 2.411-4.81-.482-1.067-1.658-3.264s-.318-.654-.428-.872c-.728-1.443-.712-1.518-1.447-1.617-.207-.023-.313-.05-.313-.149v-.468l.06-.045h4.292l.113.037v.451c0 .105-.076.15-.227.15l-.308.047c-.792.061-.661.381-.136 1.422l1.582 3.252 1.758-3.504c.293-.64.233-.801.111-.947-.07-.084-.305-.22-.812-.24l-.201-.021c-.052 0-.098-.015-.145-.051-.045-.031-.067-.076-.067-.129v-.427l.061-.045c1.247-.008 4.043 0 4.043 0l.059.045v.436c0 .121-.059.178-.193.178-.646.03-.782.095-1.023.439-.12.186-.375.589-.646 1.039l-2.301 4.273-.065.135 2.792 5.712.17.048 4.396-10.438c.154-.422.129-.722-.064-.895-.197-.172-.346-.273-.857-.295l-.42-.016c-.061 0-.105-.014-.152-.045-.043-.029-.072-.075-.072-.119v-.436l.059-.045h4.961l.041.045v.437c0 .119-.074.18-.209.18-.648.03-1.127.18-1.443.421-.314.255-.557.616-.736 1.067 0 0-4.043 9.258-5.426 12.339-.525 1.007-1.053.917-1.503-.031-.571-1.171-1.773-3.786-2.646-5.71l.053-.036z"/>
    </svg>
  );
}

function BirdIcon({ class: className }: { class?: string }): JSX.Element {
  return (
    <svg class={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path d="M16 7c0-2.21-1.79-4-4-4S8 4.79 8 7c0 .34.04.67.12 1H3l3 6v7h12v-7l3-6h-5.12c.08-.33.12-.66.12-1z" />
      <circle cx="12" cy="7" r="1" fill="currentColor" />
    </svg>
  );
}

function EBirdIcon({ class: className }: { class?: string }): JSX.Element {
  return (
    <svg class={className} viewBox="0 0 24 24" fill="currentColor">
      <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"/>
    </svg>
  );
}

function ChartIcon({ class: className }: { class?: string }): JSX.Element {
  return (
    <svg class={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path d="M3 3v18h18" />
      <path d="M18.7 8l-5.1 5.2-2.8-2.7L7 14.3" />
    </svg>
  );
}

function TrashIcon({ class: className }: { class?: string }): JSX.Element {
  return (
    <svg class={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
    </svg>
  );
}
