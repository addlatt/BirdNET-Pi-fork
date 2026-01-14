import { useState, useEffect, useRef, useCallback } from 'preact/hooks';
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
  /** Allow fullscreen toggle (default: true) */
  allowFullscreen?: boolean;
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
  allowFullscreen = true,
  class: className,
}: SpectrogramProps): JSX.Element {
  const [imageLoaded, setImageLoaded] = useState(false);
  const [imageError, setImageError] = useState(false);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const timeLabels = generateTimeLabels(duration);
  const freqLabels = generateFreqLabels(maxFreqKHz);

  // Check if Fullscreen API is supported
  const fullscreenSupported = typeof document !== 'undefined' &&
    (document.fullscreenEnabled ||
     (document as any).webkitFullscreenEnabled ||
     (document as any).mozFullScreenEnabled ||
     (document as any).msFullscreenEnabled);

  // Toggle fullscreen mode
  const toggleFullscreen = useCallback(async () => {
    if (!containerRef.current || !fullscreenSupported) return;

    try {
      if (!isFullscreen) {
        // Enter fullscreen
        const elem = containerRef.current as any;
        if (elem.requestFullscreen) {
          await elem.requestFullscreen();
        } else if (elem.webkitRequestFullscreen) {
          await elem.webkitRequestFullscreen();
        } else if (elem.mozRequestFullScreen) {
          await elem.mozRequestFullScreen();
        } else if (elem.msRequestFullscreen) {
          await elem.msRequestFullscreen();
        }
      } else {
        // Exit fullscreen
        const doc = document as any;
        if (doc.exitFullscreen) {
          await doc.exitFullscreen();
        } else if (doc.webkitExitFullscreen) {
          await doc.webkitExitFullscreen();
        } else if (doc.mozCancelFullScreen) {
          await doc.mozCancelFullScreen();
        } else if (doc.msExitFullscreen) {
          await doc.msExitFullscreen();
        }
      }
    } catch (err) {
      console.error('Fullscreen toggle failed:', err);
    }
  }, [isFullscreen, fullscreenSupported]);

  // Listen for fullscreen changes (including ESC key)
  useEffect(() => {
    const handleFullscreenChange = () => {
      const doc = document as any;
      const fullscreenElement = doc.fullscreenElement ||
        doc.webkitFullscreenElement ||
        doc.mozFullScreenElement ||
        doc.msFullscreenElement;
      setIsFullscreen(fullscreenElement === containerRef.current);
    };

    document.addEventListener('fullscreenchange', handleFullscreenChange);
    document.addEventListener('webkitfullscreenchange', handleFullscreenChange);
    document.addEventListener('mozfullscreenchange', handleFullscreenChange);
    document.addEventListener('MSFullscreenChange', handleFullscreenChange);

    return () => {
      document.removeEventListener('fullscreenchange', handleFullscreenChange);
      document.removeEventListener('webkitfullscreenchange', handleFullscreenChange);
      document.removeEventListener('mozfullscreenchange', handleFullscreenChange);
      document.removeEventListener('MSFullscreenChange', handleFullscreenChange);
    };
  }, []);

  const handleClick = (e: JSX.TargetedMouseEvent<HTMLDivElement>) => {
    if (!onClick || !imageLoaded) return;

    const rect = e.currentTarget.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const percentage = x / rect.width;
    onClick(Math.max(0, Math.min(1, percentage)));
  };

  const hasProgress = typeof progressPercent === 'number' && progressPercent >= 0;
  const showFullscreenButton = allowFullscreen && fullscreenSupported && imageLoaded;

  // Check if parent passed height constraints via className
  const hasHeightConstraint = className?.includes('h-full') || className?.includes('max-h');

  return (
    <div
      ref={containerRef}
      class={`spectrogram-container ${className || ''} ${isFullscreen ? 'fixed inset-0 z-50 bg-gray-900 flex flex-col justify-center p-4' : ''} ${hasHeightConstraint && !isFullscreen ? 'flex flex-col' : ''}`}
    >
      {/* Title */}
      {title && (
        <div class={`text-sm font-medium mb-1 ${isFullscreen ? 'text-gray-200' : 'text-gray-700 dark:text-gray-300'}`}>
          {title}
        </div>
      )}

      <div class={`flex ${isFullscreen || hasHeightConstraint ? 'flex-1 min-h-0' : ''}`}>
        {/* Y-axis (Frequency) */}
        {showAxes && imageLoaded && (
          <div class={`flex flex-col justify-between text-xs pr-1 py-0.5 ${isFullscreen ? 'text-gray-300 text-sm pr-2' : 'text-gray-500 dark:text-gray-400'}`} style={{ minWidth: isFullscreen ? '32px' : '24px' }}>
            {freqLabels.slice().reverse().map((label, i) => (
              <span key={i} class="text-right leading-none">{label.label}</span>
            ))}
          </div>
        )}

        <div class={`flex-1 flex flex-col ${isFullscreen || hasHeightConstraint ? 'min-h-0' : ''}`}>
          {/* Main spectrogram area */}
          <div class={`flex ${isFullscreen || hasHeightConstraint ? 'flex-1 min-h-0' : ''}`}>
            {/* Spectrogram image with overlays */}
            <div
              class={`relative flex-1 ${onClick ? 'cursor-pointer' : ''} ${isFullscreen || hasHeightConstraint ? 'min-h-0' : ''}`}
              onClick={handleClick}
            >
              {/* The raw spectrogram image */}
              {!imageError ? (
                <img
                  src={src}
                  alt="Spectrogram"
                  class={`w-full block rounded ${isFullscreen || hasHeightConstraint ? 'h-full object-contain' : 'h-auto'}`}
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

              {/* Fullscreen toggle button */}
              {showFullscreenButton && (
                <button
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation();
                    toggleFullscreen();
                  }}
                  class={`absolute top-2 right-2 p-1.5 rounded bg-black/50 hover:bg-black/70 text-white transition-colors ${isFullscreen ? 'top-4 right-4 p-2' : ''}`}
                  title={isFullscreen ? 'Exit fullscreen (ESC)' : 'View fullscreen'}
                  aria-label={isFullscreen ? 'Exit fullscreen' : 'Enter fullscreen'}
                >
                  {isFullscreen ? (
                    // Collapse/minimize icon
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  ) : (
                    // Expand/fullscreen icon
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
                    </svg>
                  )}
                </button>
              )}
            </div>

            {/* Color legend (dBFS scale) */}
            {showLegend && imageLoaded && (
              <div class={`flex flex-col ${isFullscreen ? 'ml-4' : 'ml-2'}`} style={{ minWidth: isFullscreen ? '48px' : '32px' }}>
                <div
                  class="flex-1 rounded"
                  style={{
                    background: DBFS_GRADIENT,
                    minHeight: isFullscreen ? '100px' : '60px',
                    width: isFullscreen ? '16px' : '12px',
                  }}
                />
                <div class={`flex flex-col justify-between text-xs mt-0.5 ${isFullscreen ? 'text-gray-300' : 'text-gray-500 dark:text-gray-400'}`} style={{ height: '100%' }}>
                  <span class="leading-none">0</span>
                  <span class="leading-none">dB</span>
                </div>
              </div>
            )}
          </div>

          {/* X-axis (Time) */}
          {showAxes && imageLoaded && duration > 0 && (
            <div class={`flex justify-between text-xs mt-1 px-0.5 ${isFullscreen ? 'text-gray-300 text-sm mt-2' : 'text-gray-500 dark:text-gray-400'}`}>
              {timeLabels.map((label, i) => (
                <span key={i}>{label.label}</span>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Axis labels */}
      {showAxes && imageLoaded && (
        <div class={`flex justify-between text-xs mt-0.5 ${isFullscreen ? 'text-gray-400 ml-8 mt-2' : 'text-gray-400 dark:text-gray-500 ml-6'}`}>
          <span>Time (seconds)</span>
          <span class={isFullscreen ? 'mr-12' : 'mr-8'}>Freq (kHz)</span>
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
