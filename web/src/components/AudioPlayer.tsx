import { useState, useRef, useEffect } from 'preact/hooks';
import type { JSX } from 'preact';

/**
 * AudioPlayer props
 */
interface AudioPlayerProps {
  /** URL of the audio file */
  audioUrl: string;
  /** URL of the spectrogram image */
  spectrogramUrl?: string;
  /** Title to display above spectrogram (e.g., species name) */
  title?: string;
}

/**
 * Generate time axis labels based on duration.
 */
function generateTimeLabels(duration: number): string[] {
  if (duration <= 0) return [];

  // For short clips (< 10s), show every second
  // For longer clips, show fewer labels
  const interval = duration <= 10 ? 1 : Math.ceil(duration / 6);
  const labels: string[] = [];

  for (let t = 0; t <= duration; t += interval) {
    const secs = Math.floor(t);
    labels.push(`${secs}s`);
  }

  return labels;
}

/**
 * AudioPlayer component with spectrogram display and time axis.
 */
export function AudioPlayer({ audioUrl, spectrogramUrl, title }: AudioPlayerProps): JSX.Element {
  const audioRef = useRef<HTMLAudioElement>(null);
  const spectrogramRef = useRef<HTMLDivElement>(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);
  const [error, setError] = useState<string | null>(null);
  const [imageLoaded, setImageLoaded] = useState(false);

  useEffect(() => {
    // Reset state when audio changes
    setIsPlaying(false);
    setCurrentTime(0);
    setDuration(0);
    setError(null);
    setImageLoaded(false);
  }, [audioUrl]);

  // Smooth progress updates using requestAnimationFrame
  useEffect(() => {
    if (!isPlaying) return;

    let animationId: number;
    const updateProgress = () => {
      if (audioRef.current) {
        setCurrentTime(audioRef.current.currentTime);
      }
      animationId = requestAnimationFrame(updateProgress);
    };
    animationId = requestAnimationFrame(updateProgress);

    return () => cancelAnimationFrame(animationId);
  }, [isPlaying]);

  const handlePlay = () => {
    if (audioRef.current) {
      audioRef.current.play().catch((err) => {
        setError('Failed to play audio');
        console.error('Audio play error:', err);
      });
    }
  };

  const handlePause = () => {
    if (audioRef.current) {
      audioRef.current.pause();
    }
  };

  const togglePlay = () => {
    if (isPlaying) {
      handlePause();
    } else {
      handlePlay();
    }
  };

  const handleTimeUpdate = () => {
    if (audioRef.current) {
      setCurrentTime(audioRef.current.currentTime);
    }
  };

  const handleLoadedMetadata = () => {
    if (audioRef.current) {
      setDuration(audioRef.current.duration);
    }
  };

  const handleSeek = (e: JSX.TargetedMouseEvent<HTMLDivElement>) => {
    const rect = e.currentTarget.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const percentage = x / rect.width;
    const newTime = percentage * duration;
    if (audioRef.current) {
      audioRef.current.currentTime = newTime;
      setCurrentTime(newTime);
    }
  };

  const formatTime = (time: number): string => {
    if (!time || isNaN(time)) return '0:00';
    const minutes = Math.floor(time / 60);
    const seconds = Math.floor(time % 60);
    return `${minutes}:${seconds.toString().padStart(2, '0')}`;
  };

  const progressPercentage = duration > 0 ? (currentTime / duration) * 100 : 0;
  const timeLabels = generateTimeLabels(duration);

  return (
    <div class="bg-gray-100 dark:bg-gray-800 rounded-lg overflow-hidden">
      {/* Title */}
      {title && (
        <div class="px-3 pt-2 pb-1">
          <h3 class="text-sm font-medium text-gray-700 dark:text-gray-300">{title}</h3>
        </div>
      )}

      {/* Spectrogram with time axis */}
      {spectrogramUrl && (
        <div class="px-3">
          {/* Spectrogram image with progress overlay */}
          <div
            ref={spectrogramRef}
            class="relative cursor-pointer"
            onClick={handleSeek}
          >
            <img
              src={spectrogramUrl}
              alt="Spectrogram"
              class="w-full h-auto block"
              onLoad={() => setImageLoaded(true)}
              onError={(e) => {
                (e.target as HTMLImageElement).style.display = 'none';
                setImageLoaded(false);
              }}
            />
            {/* Progress overlay - perfectly aligned with raw spectrogram */}
            {duration > 0 && imageLoaded && (
              <>
                {/* Playback progress fill */}
                <div
                  class="absolute top-0 left-0 h-full bg-primary-500/20 pointer-events-none"
                  style={{ width: `${progressPercentage}%` }}
                />
                {/* Playhead line */}
                <div
                  class="absolute top-0 h-full w-0.5 bg-primary-600 pointer-events-none"
                  style={{ left: `${progressPercentage}%` }}
                />
              </>
            )}
          </div>

          {/* Time axis labels */}
          {duration > 0 && imageLoaded && (
            <div class="flex justify-between text-xs text-gray-500 dark:text-gray-400 mt-1 px-0.5">
              {timeLabels.map((label, i) => (
                <span key={i}>{label}</span>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Audio element (hidden) */}
      <audio
        ref={audioRef}
        src={audioUrl}
        onPlay={() => setIsPlaying(true)}
        onPause={() => setIsPlaying(false)}
        onEnded={() => setIsPlaying(false)}
        onTimeUpdate={handleTimeUpdate}
        onLoadedMetadata={handleLoadedMetadata}
        onError={() => setError('Failed to load audio')}
      />

      {/* Controls */}
      <div class="p-3">
        {error ? (
          <div class="text-red-500 text-sm text-center">{error}</div>
        ) : (
          <div class="flex items-center gap-3">
            {/* Play/Pause button */}
            <button
              onClick={togglePlay}
              class="w-10 h-10 flex items-center justify-center rounded-full bg-primary-600 hover:bg-primary-700 text-white transition-colors"
              title={isPlaying ? 'Pause' : 'Play'}
            >
              {isPlaying ? (
                <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M6 4h4v16H6V4zm8 0h4v16h-4V4z" />
                </svg>
              ) : (
                <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M8 5v14l11-7z" />
                </svg>
              )}
            </button>

            {/* Progress bar */}
            <div
              class="flex-1 h-2 bg-gray-300 dark:bg-gray-600 rounded-full cursor-pointer"
              onClick={handleSeek}
            >
              <div
                class="h-full bg-primary-600 rounded-full"
                style={{ width: `${progressPercentage}%` }}
              />
            </div>

            {/* Time display */}
            <div class="text-sm text-gray-600 dark:text-gray-400 whitespace-nowrap">
              {formatTime(currentTime)} / {formatTime(duration)}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
