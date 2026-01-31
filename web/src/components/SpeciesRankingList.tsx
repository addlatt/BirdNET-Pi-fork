import type { JSX } from 'preact';
import { useState } from 'preact/hooks';
import type { SpeciesRankingEntry } from '../types/api';
import { BirdImage } from './BirdImage';
import { AudioPlayer } from './AudioPlayer';

/**
 * SpeciesRankingList props
 */
interface SpeciesRankingListProps {
  /** Array of species ranking entries to display */
  species: SpeciesRankingEntry[];
}

/**
 * SpeciesRankingItem props
 */
interface SpeciesRankingItemProps {
  /** Species ranking entry to display */
  entry: SpeciesRankingEntry;
  /** Rank/position in the list */
  rank: number;
  /** Currently expanded type (null, 'latest', or 'best') */
  expandedType: 'latest' | 'best' | null;
  /** Callback when expansion changes */
  onExpand: (type: 'latest' | 'best' | null) => void;
}

/**
 * List of species ranked by detection frequency.
 */
export function SpeciesRankingList({ species }: SpeciesRankingListProps): JSX.Element {
  // Track which species/type is expanded: "sciName:type" or null
  const [expandedKey, setExpandedKey] = useState<string | null>(null);

  if (!species || species.length === 0) {
    return (
      <div class="p-8 text-center text-gray-500 dark:text-gray-400">
        No species detected yet
      </div>
    );
  }

  const handleExpand = (sciName: string, type: 'latest' | 'best' | null) => {
    if (type === null) {
      setExpandedKey(null);
    } else {
      const newKey = `${sciName}:${type}`;
      setExpandedKey(expandedKey === newKey ? null : newKey);
    }
  };

  return (
    <div class="divide-y divide-gray-200 dark:divide-gray-700">
      {species.map((entry, index) => {
        const currentKey = expandedKey;
        const latestKey = `${entry.sci_name}:latest`;
        const bestKey = `${entry.sci_name}:best`;
        const expandedType = currentKey === latestKey ? 'latest' : currentKey === bestKey ? 'best' : null;

        return (
          <SpeciesRankingItem
            key={entry.sci_name}
            entry={entry}
            rank={index + 1}
            expandedType={expandedType}
            onExpand={(type) => handleExpand(entry.sci_name, type)}
          />
        );
      })}
    </div>
  );
}

/**
 * Single species ranking item with expandable Latest/Best spectrograms.
 */
function SpeciesRankingItem({ entry, rank, expandedType, onExpand }: SpeciesRankingItemProps): JSX.Element {
  // Build species directory name (com_name with spaces as underscores)
  const speciesDir = entry.com_name.replace(/ /g, '_');

  // Build file paths for latest detection
  const latestEncodedFileName = encodeURIComponent(entry.latest_file);
  const latestAudioPath = `/By_Date/${entry.latest_date}/${speciesDir}/${latestEncodedFileName}`;
  const latestSpectrogramPath = `/By_Date/${entry.latest_date}/${speciesDir}/${latestEncodedFileName}.png`;

  // Build file paths for best detection
  const bestEncodedFileName = encodeURIComponent(entry.best_file);
  const bestAudioPath = `/By_Date/${entry.best_date}/${speciesDir}/${bestEncodedFileName}`;
  const bestSpectrogramPath = `/By_Date/${entry.best_date}/${speciesDir}/${bestEncodedFileName}.png`;

  const latestConfidencePercent = Math.round(entry.latest_confidence * 100);
  const bestConfidencePercent = Math.round(entry.best_confidence * 100);

  return (
    <div class="p-3 sm:p-4 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors">
      {/* Mobile Layout */}
      <div class="sm:hidden">
        {/* Top row: rank, image, name, count */}
        <div class="flex items-center gap-2 mb-2">
          <span class="w-6 text-sm font-medium text-gray-500 dark:text-gray-400">{rank}.</span>
          <BirdImage
            sciName={entry.sci_name}
            comName={entry.com_name}
            size="small"
          />
          <div class="flex-1 min-w-0">
            <h3 class="font-medium text-gray-900 dark:text-white text-sm truncate">
              {entry.com_name}
            </h3>
            <p class="text-xs text-gray-500 dark:text-gray-400 italic truncate">
              {entry.sci_name}
            </p>
          </div>
          <span class="text-primary-600 font-bold text-lg">{entry.detection_count}</span>
        </div>

        {/* Action buttons */}
        <div class="flex items-center gap-2 ml-8">
          <button
            onClick={() => onExpand(expandedType === 'latest' ? null : 'latest')}
            class={`inline-flex items-center px-3 py-1.5 text-xs font-medium rounded touch-manipulation transition-colors ${
              expandedType === 'latest'
                ? 'bg-green-600 text-white'
                : 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300 hover:bg-green-200 dark:hover:bg-green-900/50'
            }`}
          >
            <ClockIcon class="w-3 h-3 mr-1" />
            Latest ({latestConfidencePercent}%)
          </button>
          <button
            onClick={() => onExpand(expandedType === 'best' ? null : 'best')}
            class={`inline-flex items-center px-3 py-1.5 text-xs font-medium rounded touch-manipulation transition-colors ${
              expandedType === 'best'
                ? 'bg-yellow-600 text-white'
                : 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-300 hover:bg-yellow-200 dark:hover:bg-yellow-900/50'
            }`}
          >
            <StarIcon class="w-3 h-3 mr-1" />
            Best ({bestConfidencePercent}%)
          </button>
        </div>

        {/* Expanded content */}
        {expandedType && (
          <div class="mt-3 ml-8">
            <div class="text-xs text-gray-500 dark:text-gray-400 mb-2">
              {expandedType === 'latest' ? (
                <span>{entry.latest_date} at {entry.latest_time}</span>
              ) : (
                <span>{entry.best_date} at {entry.best_time}</span>
              )}
            </div>
            <AudioPlayer
              audioUrl={expandedType === 'latest' ? latestAudioPath : bestAudioPath}
              spectrogramUrl={expandedType === 'latest' ? latestSpectrogramPath : bestSpectrogramPath}
              title={entry.com_name}
            />
          </div>
        )}
      </div>

      {/* Desktop Layout */}
      <div class="hidden sm:flex items-start gap-4">
        {/* Rank */}
        <span class="w-8 text-lg font-medium text-gray-500 dark:text-gray-400 text-right">{rank}.</span>

        {/* Bird Image */}
        <BirdImage
          sciName={entry.sci_name}
          comName={entry.com_name}
          size="medium"
        />

        {/* Species Info */}
        <div class="flex-1 min-w-0">
          <div class="flex items-center flex-wrap gap-2">
            <h3 class="font-medium text-gray-900 dark:text-white">
              {entry.com_name}
            </h3>
            <span class="text-sm text-gray-500 dark:text-gray-400 italic">
              {entry.sci_name}
            </span>
          </div>

          {/* Action buttons */}
          <div class="flex items-center mt-2 gap-2 flex-wrap">
            <button
              onClick={() => onExpand(expandedType === 'latest' ? null : 'latest')}
              class={`inline-flex items-center px-2 py-1 text-xs font-medium rounded transition-colors ${
                expandedType === 'latest'
                  ? 'bg-green-600 text-white'
                  : 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300 hover:bg-green-200 dark:hover:bg-green-900/50'
              }`}
              title={`Latest: ${entry.latest_date} ${entry.latest_time}`}
            >
              <ClockIcon class="w-3 h-3 mr-1" />
              Latest ({latestConfidencePercent}%)
            </button>
            <button
              onClick={() => onExpand(expandedType === 'best' ? null : 'best')}
              class={`inline-flex items-center px-2 py-1 text-xs font-medium rounded transition-colors ${
                expandedType === 'best'
                  ? 'bg-yellow-600 text-white'
                  : 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-300 hover:bg-yellow-200 dark:hover:bg-yellow-900/50'
              }`}
              title={`Best: ${entry.best_date} ${entry.best_time}`}
            >
              <StarIcon class="w-3 h-3 mr-1" />
              Best ({bestConfidencePercent}%)
            </button>
          </div>

          {/* Expanded content */}
          {expandedType && (
            <div class="mt-3 max-w-lg">
              <div class="text-sm text-gray-500 dark:text-gray-400 mb-2">
                {expandedType === 'latest' ? (
                  <span>{entry.latest_date} at {entry.latest_time}</span>
                ) : (
                  <span>{entry.best_date} at {entry.best_time}</span>
                )}
              </div>
              <AudioPlayer
                audioUrl={expandedType === 'latest' ? latestAudioPath : bestAudioPath}
                spectrogramUrl={expandedType === 'latest' ? latestSpectrogramPath : bestSpectrogramPath}
                title={entry.com_name}
              />
            </div>
          )}
        </div>

        {/* Detection Count */}
        <div class="flex items-center">
          <span class="text-primary-600 font-bold text-xl">{entry.detection_count}</span>
          <span class="ml-1 text-sm text-gray-500 dark:text-gray-400">detections</span>
        </div>
      </div>
    </div>
  );
}

// =============================================================================
// Icon Components
// =============================================================================

function ClockIcon({ class: className }: { class?: string }): JSX.Element {
  return (
    <svg class={className} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <circle cx="12" cy="12" r="10" />
      <path d="M12 6v6l4 2" />
    </svg>
  );
}

function StarIcon({ class: className }: { class?: string }): JSX.Element {
  return (
    <svg class={className} viewBox="0 0 24 24" fill="currentColor">
      <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z" />
    </svg>
  );
}
