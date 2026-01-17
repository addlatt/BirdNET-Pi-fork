import { useState, useRef, useEffect, useCallback } from 'preact/hooks';
import type { JSX } from 'preact';
import { Spectrogram } from './Spectrogram';

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
  /** Allow fullscreen mode (default: true) */
  allowFullscreen?: boolean;
}

/**
 * AudioPlayer component with spectrogram display and time axis.
 */
export function AudioPlayer({ audioUrl, spectrogramUrl, title, allowFullscreen = true }: AudioPlayerProps): JSX.Element {
  const audioRef = useRef<HTMLAudioElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);
  const [error, setError] = useState<string | null>(null);
  const [isFullscreen, setIsFullscreen] = useState(false);

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

  // Listen for fullscreen changes
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

  useEffect(() => {
    // Reset state when audio changes
    setIsPlaying(false);
    setCurrentTime(0);
    setDuration(0);
    setError(null);
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
    seekToPercentage(percentage);
  };

  const seekToPercentage = (percentage: number) => {
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
  const showFullscreenButton = allowFullscreen && fullscreenSupported && spectrogramUrl;

  return (
    <div
      ref={containerRef}
      class={`bg-gray-100 dark:bg-gray-800 rounded-lg overflow-hidden ${isFullscreen ? 'fixed inset-0 z-50 flex flex-col' : ''}`}
    >
      {/* Spectrogram with axes and color legend */}
      {spectrogramUrl && (
        <div class={`relative ${isFullscreen ? 'flex-1 min-h-0 p-4 overflow-hidden' : 'px-3 pt-2'}`}>
          <Spectrogram
            src={spectrogramUrl}
            duration={duration}
            title={title}
            onClick={seekToPercentage}
            progressPercent={progressPercentage}
            allowFullscreen={false}
            showDetectionWindow={true}
            class={isFullscreen ? 'h-full max-h-full overflow-hidden' : ''}
          />
          {/* Fullscreen toggle button */}
          {showFullscreenButton && (
            <button
              type="button"
              onClick={toggleFullscreen}
              class={`absolute top-4 right-4 p-1.5 rounded bg-black/50 hover:bg-black/70 text-white transition-colors z-10 ${isFullscreen ? 'top-6 right-6 p-2' : ''}`}
              title={isFullscreen ? 'Exit fullscreen (ESC)' : 'View fullscreen'}
              aria-label={isFullscreen ? 'Exit fullscreen' : 'Enter fullscreen'}
            >
              {isFullscreen ? (
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
                </svg>
              ) : (
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
                </svg>
              )}
            </button>
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
      <div class={`p-3 ${isFullscreen ? 'bg-gray-900 border-t border-gray-700' : ''}`}>
        {error ? (
          <div class="text-red-500 text-sm text-center">{error}</div>
        ) : (
          <div class="flex items-center gap-3">
            {/* Play/Pause button */}
            <button
              onClick={togglePlay}
              class={`flex items-center justify-center rounded-full bg-primary-600 hover:bg-primary-700 text-white transition-colors ${isFullscreen ? 'w-14 h-14' : 'w-10 h-10'}`}
              title={isPlaying ? 'Pause' : 'Play'}
            >
              {isPlaying ? (
                <svg class={isFullscreen ? 'w-7 h-7' : 'w-5 h-5'} fill="currentColor" viewBox="0 0 24 24">
                  <path d="M6 4h4v16H6V4zm8 0h4v16h-4V4z" />
                </svg>
              ) : (
                <svg class={isFullscreen ? 'w-7 h-7' : 'w-5 h-5'} fill="currentColor" viewBox="0 0 24 24">
                  <path d="M8 5v14l11-7z" />
                </svg>
              )}
            </button>

            {/* Progress bar */}
            <div
              class={`flex-1 bg-gray-300 dark:bg-gray-600 rounded-full cursor-pointer ${isFullscreen ? 'h-3' : 'h-2'}`}
              onClick={handleSeek}
            >
              <div
                class="h-full bg-primary-600 rounded-full"
                style={{ width: `${progressPercentage}%` }}
              />
            </div>

            {/* Time display */}
            <div class={`text-gray-600 dark:text-gray-400 whitespace-nowrap ${isFullscreen ? 'text-base text-gray-300' : 'text-sm'}`}>
              {formatTime(currentTime)} / {formatTime(duration)}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
