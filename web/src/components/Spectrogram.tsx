import { useState } from 'preact/hooks';
import type { JSX } from 'preact';

/**
 * Spectrogram component props
 */
export interface SpectrogramProps {
  /** URL to the raw spectrogram PNG */
  src: string;
  /** Audio duration in seconds (for X-axis) */
  duration: number;
  /** Optional title overlay */
  title?: string;
  /** Optional click handler for seeking - receives percentage (0-1) */
  onClick?: (percentage: number) => void;
  /** Optional playback progress (0-100) */
  progressPercent?: number;
  /** Maximum frequency in kHz (default: 12 for bird audio) */
  maxFreqKHz?: number;
  /** Show the color legend (default: true) */
  showLegend?: boolean;
  /** Show axes (default: true) */
  showAxes?: boolean;
  /** Additional CSS class for the container */
  class?: string;
}

/**
 * Generate time axis labels based on duration.
 */
function generateTimeLabels(duration: number): { value: number; label: string }[] {
  if (duration <= 0) return [];

  // For short clips (< 10s), show every second
  // For longer clips, show fewer labels
  const interval = duration <= 10 ? 1 : Math.ceil(duration / 6);
  const labels: { value: number; label: string }[] = [];

  for (let t = 0; t <= duration; t += interval) {
    const secs = Math.floor(t);
    labels.push({ value: t, label: `${secs}s` });
  }

  return labels;
}

/**
 * Generate frequency axis labels (0 to maxFreq kHz).
 */
function generateFreqLabels(maxFreqKHz: number): { value: number; label: string }[] {
  // Show labels at 0, 3, 6, 9, 12 kHz for 12kHz max
  const step = maxFreqKHz <= 6 ? 1 : maxFreqKHz <= 12 ? 3 : 6;
  const labels: { value: number; label: string }[] = [];

  for (let f = 0; f <= maxFreqKHz; f += step) {
    labels.push({ value: f, label: `${f}` });
  }

  return labels;
}

/**
 * Color stops for the dBFS gradient legend.
 * Sox default spectrogram uses a colormap from dark (low energy) to bright (high energy).
 */
const DBFS_GRADIENT = 'linear-gradient(to top, #1a0a2e, #2d1b4e, #4a2c7a, #7b3fa0, #a855f7, #d97706, #fbbf24, #fef3c7)';

/**
 * Reusable Spectrogram component that displays a raw spectrogram image
 * with proper axes overlaid (time, frequency, and dBFS color legend).
 */
export function Spectrogram({
  src,
  duration,
  title,
  onClick,
  progressPercent,
  maxFreqKHz = 12,
  showLegend = true,
  showAxes = true,
  class: className,
}: SpectrogramProps): JSX.Element {
  const [imageLoaded, setImageLoaded] = useState(false);
  const [imageError, setImageError] = useState(false);

  const timeLabels = generateTimeLabels(duration);
  const freqLabels = generateFreqLabels(maxFreqKHz);

  const handleClick = (e: JSX.TargetedMouseEvent<HTMLDivElement>) => {
    if (!onClick || !imageLoaded) return;

    const rect = e.currentTarget.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const percentage = x / rect.width;
    onClick(Math.max(0, Math.min(1, percentage)));
  };

  const hasProgress = typeof progressPercent === 'number' && progressPercent >= 0;

  return (
    <div class={`spectrogram-container ${className || ''}`}>
      {/* Title */}
      {title && (
        <div class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
          {title}
        </div>
      )}

      <div class="flex">
        {/* Y-axis (Frequency) */}
        {showAxes && imageLoaded && (
          <div class="flex flex-col justify-between text-xs text-gray-500 dark:text-gray-400 pr-1 py-0.5" style={{ minWidth: '24px' }}>
            {freqLabels.slice().reverse().map((label, i) => (
              <span key={i} class="text-right leading-none">{label.label}</span>
            ))}
          </div>
        )}

        <div class="flex-1 flex flex-col">
          {/* Main spectrogram area */}
          <div class="flex">
            {/* Spectrogram image with overlays */}
            <div
              class={`relative flex-1 ${onClick ? 'cursor-pointer' : ''}`}
              onClick={handleClick}
            >
              {/* The raw spectrogram image */}
              {!imageError ? (
                <img
                  src={src}
                  alt="Spectrogram"
                  class="w-full h-auto block rounded"
                  onLoad={() => setImageLoaded(true)}
                  onError={() => {
                    setImageError(true);
                    setImageLoaded(false);
                  }}
                />
              ) : (
                <div class="w-full h-32 bg-gray-200 dark:bg-gray-700 rounded flex items-center justify-center">
                  <span class="text-sm text-gray-500 dark:text-gray-400">
                    Spectrogram unavailable
                  </span>
                </div>
              )}

              {/* Progress overlay */}
              {hasProgress && imageLoaded && (
                <>
                  {/* Playback progress fill */}
                  <div
                    class="absolute top-0 left-0 h-full bg-primary-500/20 pointer-events-none rounded-l"
                    style={{ width: `${progressPercent}%` }}
                  />
                  {/* Playhead line */}
                  <div
                    class="absolute top-0 h-full w-0.5 bg-primary-600 pointer-events-none"
                    style={{ left: `${progressPercent}%` }}
                  />
                </>
              )}
            </div>

            {/* Color legend (dBFS scale) */}
            {showLegend && imageLoaded && (
              <div class="flex flex-col ml-2" style={{ minWidth: '32px' }}>
                <div
                  class="flex-1 rounded"
                  style={{
                    background: DBFS_GRADIENT,
                    minHeight: '60px',
                    width: '12px',
                  }}
                />
                <div class="flex flex-col justify-between text-xs text-gray-500 dark:text-gray-400 mt-0.5" style={{ height: '100%' }}>
                  <span class="leading-none">0</span>
                  <span class="leading-none">dB</span>
                </div>
              </div>
            )}
          </div>

          {/* X-axis (Time) */}
          {showAxes && imageLoaded && duration > 0 && (
            <div class="flex justify-between text-xs text-gray-500 dark:text-gray-400 mt-1 px-0.5">
              {timeLabels.map((label, i) => (
                <span key={i}>{label.label}</span>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Axis labels */}
      {showAxes && imageLoaded && (
        <div class="flex justify-between text-xs text-gray-400 dark:text-gray-500 mt-0.5 ml-6">
          <span>Time (seconds)</span>
          <span class="mr-8">Freq (kHz)</span>
        </div>
      )}
    </div>
  );
}

/**
 * Compact spectrogram without axes - useful for thumbnails or cards
 */
export function SpectrogramThumbnail({
  src,
  onClick,
  progressPercent,
  class: className,
}: Pick<SpectrogramProps, 'src' | 'onClick' | 'progressPercent' | 'class'>): JSX.Element {
  return (
    <Spectrogram
      src={src}
      duration={0}
      showAxes={false}
      showLegend={false}
      onClick={onClick}
      progressPercent={progressPercent}
      class={className}
    />
  );
}
