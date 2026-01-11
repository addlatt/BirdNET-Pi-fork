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
}

/**
 * AudioPlayer component with spectrogram display.
 */
export function AudioPlayer({ audioUrl, spectrogramUrl }: AudioPlayerProps): JSX.Element {
  const audioRef = useRef<HTMLAudioElement>(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    // Reset state when audio changes
    setIsPlaying(false);
    setCurrentTime(0);
    setDuration(0);
    setError(null);
  }, [audioUrl]);

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

  return (
    <div class="bg-gray-100 dark:bg-gray-800 rounded-lg overflow-hidden">
      {/* Spectrogram */}
      {spectrogramUrl && (
        <div class="relative">
          <img
            src={spectrogramUrl}
            alt="Spectrogram"
            class="w-full h-auto"
            onError={(e) => {
              (e.target as HTMLImageElement).style.display = 'none';
            }}
          />
          {/* Progress overlay on spectrogram */}
          {duration > 0 && (
            <div
              class="absolute top-0 left-0 h-full bg-primary-500 opacity-20 pointer-events-none"
              style={{ width: `${progressPercentage}%` }}
            />
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
                class="h-full bg-primary-600 rounded-full transition-all"
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
