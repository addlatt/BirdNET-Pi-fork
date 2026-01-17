import { useState, useRef, useEffect, useCallback } from 'preact/hooks';
import type { JSX } from 'preact';

/**
 * Gain values in dB mapped to linear multipliers
 */
const GAIN_OPTIONS: Record<string, number> = {
  'Off': 1,
  '6': 2,
  '12': 4,
  '18': 8,
  '24': 16,
  '30': 32,
};

const HIGHPASS_OPTIONS = ['Off', '250', '500', '1000', '1500'];
const LOWPASS_OPTIONS = ['Off', '2000', '4000', '8000'];

/**
 * Safe localStorage access
 */
function getPreference(key: string, fallback: string): string {
  try {
    return localStorage.getItem(key) || fallback;
  } catch {
    return fallback;
  }
}

function setPreference(key: string, value: string): void {
  try {
    localStorage.setItem(key, value);
  } catch {
    // Ignore storage errors
  }
}

/**
 * EnhancedAudioPlayer props
 */
interface EnhancedAudioPlayerProps {
  /** URL of the audio file */
  audioUrl: string;
  /** URL of the spectrogram image */
  spectrogramUrl?: string;
  /** Title to display (e.g., species name) */
  title?: string;
  /** Date string for display */
  date?: string;
  /** Time string for display */
  time?: string;
  /** Confidence score */
  confidence?: number;
  /** Whether the file is locked from purge */
  isLocked?: boolean;
  /** Whether a shifted version exists */
  isShifted?: boolean;
  /** URL of the shifted version */
  shiftedUrl?: string;
  /** Callback when delete is requested */
  onDelete?: () => void;
  /** Callback when change identification is requested */
  onChangeIdentification?: () => void;
  /** Callback when toggle lock is requested */
  onToggleLock?: () => void;
  /** Callback when toggle shift is requested */
  onToggleShift?: () => void;
  /** Show action buttons */
  showActions?: boolean;
}

/**
 * EnhancedAudioPlayer with Web Audio API support for gain/filter controls.
 */
export function EnhancedAudioPlayer({
  audioUrl,
  spectrogramUrl,
  title,
  date,
  time,
  confidence,
  isLocked = false,
  isShifted = false,
  shiftedUrl,
  onDelete,
  onChangeIdentification,
  onToggleLock,
  onToggleShift,
  showActions = false,
}: EnhancedAudioPlayerProps): JSX.Element {
  const audioRef = useRef<HTMLAudioElement>(null);
  const audioContextRef = useRef<AudioContext | null>(null);
  const sourceNodeRef = useRef<MediaElementAudioSourceNode | null>(null);
  const gainNodeRef = useRef<GainNode | null>(null);
  const highpassNodeRef = useRef<BiquadFilterNode | null>(null);
  const lowpassNodeRef = useRef<BiquadFilterNode | null>(null);

  const [isPlaying, setIsPlaying] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showMenu, setShowMenu] = useState(false);
  const [showHover, setShowHover] = useState(false);
  const [useShifted, setUseShifted] = useState(false);

  // Audio processing preferences
  const [activeGain, setActiveGain] = useState(getPreference('customAudioPlayerGain', 'Off'));
  const [activeHighpass, setActiveHighpass] = useState(getPreference('customAudioPlayerFilterHigh', 'Off'));
  const [activeLowpass, setActiveLowpass] = useState(getPreference('customAudioPlayerFilterLow', 'Off'));

  // Current audio source
  const currentAudioUrl = useShifted && shiftedUrl ? shiftedUrl : audioUrl;

  // Reset state when audio changes
  useEffect(() => {
    setIsPlaying(false);
    setCurrentTime(0);
    setDuration(0);
    setError(null);
    setIsLoading(false);
    // Disconnect existing audio context nodes when source changes
    if (sourceNodeRef.current) {
      sourceNodeRef.current.disconnect();
      sourceNodeRef.current = null;
    }
  }, [currentAudioUrl]);

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

  /**
   * Initialize or get Web Audio API context
   */
  const initAudioContext = useCallback(async () => {
    if (!audioRef.current) return;

    if (!audioContextRef.current) {
      audioContextRef.current = new (window.AudioContext || (window as any).webkitAudioContext)();
    }

    if (!sourceNodeRef.current) {
      sourceNodeRef.current = audioContextRef.current.createMediaElementSource(audioRef.current);
      gainNodeRef.current = audioContextRef.current.createGain();
      gainNodeRef.current.gain.value = 1;
      sourceNodeRef.current.connect(gainNodeRef.current).connect(audioContextRef.current.destination);
    }

    if (audioContextRef.current.state === 'suspended') {
      await audioContextRef.current.resume();
    }
  }, []);

  /**
   * Rebuild the audio processing chain
   */
  const rebuildAudioChain = useCallback(() => {
    if (!audioContextRef.current || !sourceNodeRef.current || !gainNodeRef.current) return;

    sourceNodeRef.current.disconnect();
    gainNodeRef.current.disconnect();
    if (highpassNodeRef.current) highpassNodeRef.current.disconnect();
    if (lowpassNodeRef.current) lowpassNodeRef.current.disconnect();

    let currentChain: AudioNode = sourceNodeRef.current;

    if (highpassNodeRef.current) {
      currentChain.connect(highpassNodeRef.current);
      currentChain = highpassNodeRef.current;
    }

    if (lowpassNodeRef.current) {
      currentChain.connect(lowpassNodeRef.current);
      currentChain = lowpassNodeRef.current;
    }

    currentChain.connect(gainNodeRef.current).connect(audioContextRef.current.destination);
  }, []);

  /**
   * Set gain level
   */
  const handleSetGain = useCallback(async (value: string) => {
    setActiveGain(value);
    setPreference('customAudioPlayerGain', value);

    if (value !== 'Off') {
      await initAudioContext();
      if (gainNodeRef.current) {
        gainNodeRef.current.gain.value = GAIN_OPTIONS[value] || 1;
      }
    } else if (gainNodeRef.current) {
      gainNodeRef.current.gain.value = 1;
    }
  }, [initAudioContext]);

  /**
   * Set highpass filter
   */
  const handleSetHighpass = useCallback(async (value: string) => {
    setActiveHighpass(value);
    setPreference('customAudioPlayerFilterHigh', value);

    if (value !== 'Off') {
      await initAudioContext();
      if (!highpassNodeRef.current && audioContextRef.current) {
        highpassNodeRef.current = audioContextRef.current.createBiquadFilter();
        highpassNodeRef.current.type = 'highpass';
      }
      if (highpassNodeRef.current) {
        highpassNodeRef.current.frequency.value = parseFloat(value);
      }
    } else if (highpassNodeRef.current) {
      highpassNodeRef.current.disconnect();
      highpassNodeRef.current = null;
    }

    rebuildAudioChain();
  }, [initAudioContext, rebuildAudioChain]);

  /**
   * Set lowpass filter
   */
  const handleSetLowpass = useCallback(async (value: string) => {
    setActiveLowpass(value);
    setPreference('customAudioPlayerFilterLow', value);

    if (value !== 'Off') {
      await initAudioContext();
      if (!lowpassNodeRef.current && audioContextRef.current) {
        lowpassNodeRef.current = audioContextRef.current.createBiquadFilter();
        lowpassNodeRef.current.type = 'lowpass';
      }
      if (lowpassNodeRef.current) {
        lowpassNodeRef.current.frequency.value = parseFloat(value);
      }
    } else if (lowpassNodeRef.current) {
      lowpassNodeRef.current.disconnect();
      lowpassNodeRef.current = null;
    }

    rebuildAudioChain();
  }, [initAudioContext, rebuildAudioChain]);

  // Apply initial audio settings when context is created
  useEffect(() => {
    if (activeGain !== 'Off' && gainNodeRef.current) {
      gainNodeRef.current.gain.value = GAIN_OPTIONS[activeGain] || 1;
    }
  }, [activeGain]);

  const handlePlay = async () => {
    if (audioRef.current) {
      setIsLoading(true);
      await initAudioContext();
      // Apply stored preferences
      if (activeGain !== 'Off' && gainNodeRef.current) {
        gainNodeRef.current.gain.value = GAIN_OPTIONS[activeGain] || 1;
      }
      audioRef.current.play().catch((err) => {
        setError('Failed to play audio');
        console.error('Audio play error:', err);
      }).finally(() => {
        setIsLoading(false);
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

  const handleDownload = async () => {
    setShowMenu(false);
    try {
      setIsLoading(true);
      const response = await fetch(currentAudioUrl);
      const blob = await response.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = currentAudioUrl.split('/').pop() || 'audio_file';
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch {
      setError('Failed to download audio');
    } finally {
      setIsLoading(false);
    }
  };

  const handleShowInfo = async () => {
    setShowMenu(false);
    if (audioRef.current) {
      const dur = audioRef.current.duration ? `${audioRef.current.duration.toFixed(2)} s` : 'Unknown';
      const format = currentAudioUrl.split('.').pop()?.toUpperCase() || 'Unknown';
      alert(`Duration: ${dur}\nFormat: ${format}`);
    }
  };

  const formatTime = (t: number): string => {
    if (!t || isNaN(t)) return '0:00';
    const minutes = Math.floor(t / 60);
    const seconds = Math.floor(t % 60);
    return `${minutes}:${seconds.toString().padStart(2, '0')}`;
  };

  const formatFrequency = (hz: string): string => {
    if (hz === 'Off') return 'Off';
    const num = parseInt(hz, 10);
    if (num >= 1000) return `${num / 1000}k`;
    return hz;
  };

  const progressPercentage = duration > 0 ? (currentTime / duration) * 100 : 0;

  return (
    <div
      class="relative rounded-xl overflow-hidden bg-gray-900"
      onMouseEnter={() => setShowHover(true)}
      onMouseLeave={() => !showMenu && setShowHover(false)}
    >
      {/* Spectrogram */}
      {spectrogramUrl ? (
        <div class="relative">
          <img
            src={spectrogramUrl}
            alt="Spectrogram"
            class="w-full h-auto block"
            onError={() => setError('Spectrogram unavailable')}
          />
          {/* Progress overlay */}
          {duration > 0 && (
            <>
              <div
                class="absolute top-0 left-0 h-full bg-black/40 pointer-events-none"
                style={{ width: `${progressPercentage}%` }}
              />
              <div
                class="absolute top-0 h-full w-0.5 bg-white pointer-events-none"
                style={{ left: `${progressPercentage}%` }}
              />
            </>
          )}
        </div>
      ) : (
        <div class="w-full h-32 bg-gray-800 flex items-center justify-center">
          <span class="text-gray-400 text-sm">No spectrogram</span>
        </div>
      )}

      {/* Hidden audio element */}
      <audio
        ref={audioRef}
        src={currentAudioUrl}
        preload="metadata"
        onPlay={() => setIsPlaying(true)}
        onPause={() => setIsPlaying(false)}
        onEnded={() => setIsPlaying(false)}
        onLoadedMetadata={() => {
          if (audioRef.current) {
            setDuration(audioRef.current.duration);
          }
        }}
        onWaiting={() => setIsLoading(true)}
        onCanPlay={() => setIsLoading(false)}
        onError={() => setError('Failed to load audio')}
      />

      {/* Loading spinner */}
      {isLoading && (
        <div class="absolute inset-0 flex items-center justify-center bg-black/50">
          <div class="w-10 h-10 border-4 border-white/30 border-t-white rounded-full animate-spin" />
        </div>
      )}

      {/* Error message */}
      {error && (
        <div class="absolute inset-0 flex items-center justify-center bg-black/50">
          <div class="bg-red-600 text-white px-4 py-2 rounded-lg text-sm">{error}</div>
        </div>
      )}

      {/* Center play button - shown when paused and hovering */}
      {(showHover || !isPlaying) && !isLoading && !error && (
        <button
          onClick={togglePlay}
          class="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-16 h-16 rounded-full bg-black/50 hover:bg-black/70 flex items-center justify-center transition-colors"
        >
          {isPlaying ? (
            <svg class="w-8 h-8 text-white" fill="currentColor" viewBox="0 0 24 24">
              <path d="M6 4h4v16H6V4zm8 0h4v16h-4V4z" />
            </svg>
          ) : (
            <svg class="w-8 h-8 text-white" fill="currentColor" viewBox="0 0 24 24">
              <path d="M8 5v14l11-7z" />
            </svg>
          )}
        </button>
      )}

      {/* Bottom controls overlay */}
      {(showHover || isPlaying) && (
        <div class="absolute bottom-0 left-0 right-0 bg-black/70 backdrop-blur-sm p-3">
          {/* Title row */}
          {(title || date || confidence !== undefined) && (
            <div class="flex items-center justify-between text-white text-sm mb-2">
              <span class="font-medium truncate">{title}</span>
              <div class="flex items-center gap-2 text-xs text-gray-300">
                {time && <span>{time}</span>}
                {date && <span>{date}</span>}
                {confidence !== undefined && (
                  <span class="bg-primary-600 px-1.5 py-0.5 rounded">{Math.round(confidence * 100)}%</span>
                )}
              </div>
            </div>
          )}

          {/* Progress bar and time */}
          <div class="flex items-center gap-3">
            <span class="text-white text-xs w-10">{formatTime(currentTime)}</span>
            <div
              class="flex-1 h-1 bg-white/30 rounded-full cursor-pointer"
              onClick={handleSeek}
            >
              <div
                class="h-full bg-white rounded-full"
                style={{ width: `${progressPercentage}%` }}
              />
            </div>
            <span class="text-white text-xs w-10 text-right">{formatTime(duration)}</span>

            {/* Menu button */}
            <button
              onClick={() => setShowMenu(!showMenu)}
              class="w-8 h-8 flex items-center justify-center rounded-full hover:bg-white/20"
            >
              <svg class="w-5 h-5 text-white" fill="currentColor" viewBox="0 0 24 24">
                <path d="M12 6a2 2 0 1 0 0-4 2 2 0 0 0 0 4zM12 14a2 2 0 1 0 0-4 2 2 0 0 0 0 4zM12 22a2 2 0 1 0 0-4 2 2 0 0 0 0 4z" />
              </svg>
            </button>
          </div>
        </div>
      )}

      {/* Menu dropdown */}
      {showMenu && (
        <div class="absolute bottom-16 right-2 bg-black/90 backdrop-blur-sm rounded-lg p-3 min-w-48 z-10">
          {/* Info and Download */}
          <button onClick={handleShowInfo} class="w-full text-left text-white text-sm py-2 px-3 hover:bg-white/10 rounded">
            Info
          </button>
          <button onClick={handleDownload} class="w-full text-left text-white text-sm py-2 px-3 hover:bg-white/10 rounded">
            Download
          </button>

          {/* Shifted toggle */}
          {isShifted && shiftedUrl && (
            <button
              onClick={() => setUseShifted(!useShifted)}
              class="w-full text-left text-white text-sm py-2 px-3 hover:bg-white/10 rounded flex items-center justify-between"
            >
              <span>Use shifted version</span>
              <span class={useShifted ? 'text-primary-400' : 'text-gray-500'}>{useShifted ? 'ON' : 'OFF'}</span>
            </button>
          )}

          {/* Gain control */}
          <div class="border-t border-white/20 mt-2 pt-2">
            <div class="text-gray-400 text-xs px-3 mb-1">Gain (dB)</div>
            <div class="flex flex-wrap gap-1 px-2">
              {Object.keys(GAIN_OPTIONS).map((opt) => (
                <button
                  key={opt}
                  onClick={() => handleSetGain(opt)}
                  class={`px-2 py-1 text-xs rounded ${activeGain === opt ? 'bg-primary-600 text-white' : 'text-white hover:bg-white/10'}`}
                >
                  {opt}
                </button>
              ))}
            </div>
          </div>

          {/* Highpass filter */}
          <div class="border-t border-white/20 mt-2 pt-2">
            <div class="text-gray-400 text-xs px-3 mb-1">Highpass (Hz)</div>
            <div class="flex flex-wrap gap-1 px-2">
              {HIGHPASS_OPTIONS.map((opt) => (
                <button
                  key={opt}
                  onClick={() => handleSetHighpass(opt)}
                  class={`px-2 py-1 text-xs rounded ${activeHighpass === opt ? 'bg-primary-600 text-white' : 'text-white hover:bg-white/10'}`}
                >
                  {formatFrequency(opt)}
                </button>
              ))}
            </div>
          </div>

          {/* Lowpass filter */}
          <div class="border-t border-white/20 mt-2 pt-2">
            <div class="text-gray-400 text-xs px-3 mb-1">Lowpass (Hz)</div>
            <div class="flex flex-wrap gap-1 px-2">
              {LOWPASS_OPTIONS.map((opt) => (
                <button
                  key={opt}
                  onClick={() => handleSetLowpass(opt)}
                  class={`px-2 py-1 text-xs rounded ${activeLowpass === opt ? 'bg-primary-600 text-white' : 'text-white hover:bg-white/10'}`}
                >
                  {formatFrequency(opt)}
                </button>
              ))}
            </div>
          </div>

          {/* Action buttons */}
          {showActions && (
            <div class="border-t border-white/20 mt-2 pt-2 space-y-1">
              {onToggleLock && (
                <button
                  onClick={() => { setShowMenu(false); onToggleLock(); }}
                  class="w-full text-left text-sm py-2 px-3 hover:bg-white/10 rounded flex items-center gap-2"
                >
                  <svg class={`w-4 h-4 ${isLocked ? 'text-yellow-400' : 'text-white'}`} fill="currentColor" viewBox="0 0 24 24">
                    {isLocked ? (
                      <path d="M18 8h-1V6c0-2.76-2.24-5-5-5S7 3.24 7 6v2H6c-1.1 0-2 .9-2 2v10c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V10c0-1.1-.9-2-2-2zm-6 9c-1.1 0-2-.9-2-2s.9-2 2-2 2 .9 2 2-.9 2-2 2zm3.1-9H8.9V6c0-1.71 1.39-3.1 3.1-3.1 1.71 0 3.1 1.39 3.1 3.1v2z" />
                    ) : (
                      <path d="M12 17c1.1 0 2-.9 2-2s-.9-2-2-2-2 .9-2 2 .9 2 2 2zm6-9h-1V6c0-2.76-2.24-5-5-5S7 3.24 7 6h1.9c0-1.71 1.39-3.1 3.1-3.1 1.71 0 3.1 1.39 3.1 3.1v2H6c-1.1 0-2 .9-2 2v10c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V10c0-1.1-.9-2-2-2z" />
                    )}
                  </svg>
                  <span class="text-white">{isLocked ? 'Unlock from purge' : 'Lock from purge'}</span>
                </button>
              )}
              {onToggleShift && (
                <button
                  onClick={() => { setShowMenu(false); onToggleShift(); }}
                  class="w-full text-left text-white text-sm py-2 px-3 hover:bg-white/10 rounded flex items-center gap-2"
                >
                  <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
                    <path d="M12 3v10.55c-.59-.34-1.27-.55-2-.55-2.21 0-4 1.79-4 4s1.79 4 4 4 4-1.79 4-4V7h4V3h-6z" />
                  </svg>
                  <span>{isShifted ? 'Remove shifted' : 'Create shifted'}</span>
                </button>
              )}
              {onChangeIdentification && (
                <button
                  onClick={() => { setShowMenu(false); onChangeIdentification(); }}
                  class="w-full text-left text-white text-sm py-2 px-3 hover:bg-white/10 rounded flex items-center gap-2"
                >
                  <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
                    <path d="M3 17.25V21h3.75L17.81 9.94l-3.75-3.75L3 17.25zM20.71 7.04c.39-.39.39-1.02 0-1.41l-2.34-2.34c-.39-.39-1.02-.39-1.41 0l-1.83 1.83 3.75 3.75 1.83-1.83z" />
                  </svg>
                  <span>Change identification</span>
                </button>
              )}
              {onDelete && (
                <button
                  onClick={() => { setShowMenu(false); onDelete(); }}
                  class="w-full text-left text-red-400 text-sm py-2 px-3 hover:bg-white/10 rounded flex items-center gap-2"
                >
                  <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
                    <path d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z" />
                  </svg>
                  <span>Delete</span>
                </button>
              )}
            </div>
          )}
        </div>
      )}

      {/* Lock indicator */}
      {isLocked && (
        <div class="absolute top-2 right-2 bg-yellow-500 text-white text-xs px-2 py-1 rounded-full flex items-center gap-1">
          <svg class="w-3 h-3" fill="currentColor" viewBox="0 0 24 24">
            <path d="M18 8h-1V6c0-2.76-2.24-5-5-5S7 3.24 7 6v2H6c-1.1 0-2 .9-2 2v10c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V10c0-1.1-.9-2-2-2z" />
          </svg>
          Locked
        </div>
      )}

      {/* Click to close menu when clicking outside */}
      {showMenu && (
        <div
          class="fixed inset-0 z-0"
          onClick={() => setShowMenu(false)}
        />
      )}
    </div>
  );
}
