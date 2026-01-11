import { useState, useEffect, useCallback, useRef } from 'preact/hooks';
import type { JSX } from 'preact';
import { useWebSocket } from '../hooks/useWebSocket';
import {
  fetchSpectrogramInfo,
  getSpectrogramImageUrl,
  fetchRecentDetections,
} from '../hooks/useApi';
import type {
  SpectrogramInfoResponse,
  RecentDetection,
  DetectionNotification,
} from '../types/api';

/**
 * Spectrogram/Live View page component.
 * Displays spectrogram image with auto-refresh and recent detections feed.
 */
export function Spectrogram(): JSX.Element {
  const [info, setInfo] = useState<SpectrogramInfoResponse | null>(null);
  const [imageUrl, setImageUrl] = useState<string>('');
  const [recentDetections, setRecentDetections] = useState<RecentDetection[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const audioRef = useRef<HTMLAudioElement>(null);
  const refreshIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // WebSocket connection
  const wsUrl = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws`;
  const { isConnected, subscribe } = useWebSocket(wsUrl);

  // Load initial data
  const loadData = useCallback(async () => {
    try {
      setLoading(true);
      const [infoData, detectionsData] = await Promise.all([
        fetchSpectrogramInfo(),
        fetchRecentDetections({ limit: 10 }),
      ]);
      setInfo(infoData);
      setRecentDetections(detectionsData.detections || []);
      setImageUrl(getSpectrogramImageUrl());
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  }, []);

  // Load data on mount
  useEffect(() => {
    loadData();
  }, [loadData]);

  // Set up auto-refresh for spectrogram image
  useEffect(() => {
    if (!info || !info.available) return;

    const refreshMs = (info.refresh_seconds || 3) * 1000;
    refreshIntervalRef.current = setInterval(() => {
      setImageUrl(getSpectrogramImageUrl());
    }, refreshMs);

    return () => {
      if (refreshIntervalRef.current) {
        clearInterval(refreshIntervalRef.current);
      }
    };
  }, [info]);

  // Subscribe to real-time detection updates
  useEffect(() => {
    const unsubscribe = subscribe<DetectionNotification>('detection', (payload) => {
      const detection: RecentDetection = {
        time: payload.time,
        com_name: payload.com_name,
        sci_name: payload.sci_name,
        confidence: payload.confidence,
        file_name: payload.file_name,
      };

      // Add new detection to the top of the list
      setRecentDetections((prev) => [detection, ...prev].slice(0, 10));
    });

    return unsubscribe;
  }, [subscribe]);

  // Handle audio play/pause
  const toggleAudio = useCallback(() => {
    if (!audioRef.current) return;

    if (isPlaying) {
      audioRef.current.pause();
    } else {
      audioRef.current.play();
    }
    setIsPlaying(!isPlaying);
  }, [isPlaying]);

  // Handle audio play event
  const handleAudioPlay = useCallback(() => setIsPlaying(true), []);
  const handleAudioPause = useCallback(() => setIsPlaying(false), []);

  // Format confidence as percentage
  const formatConfidence = (confidence: number): string => {
    return `${Math.round(confidence * 100)}%`;
  };

  if (loading) {
    return (
      <div class="flex items-center justify-center h-64">
        <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
      </div>
    );
  }

  if (error) {
    return (
      <div class="p-6 text-center">
        <p class="text-red-600 dark:text-red-400">Error: {error}</p>
        <button class="btn btn-primary mt-4" onClick={() => loadData()}>
          Retry
        </button>
      </div>
    );
  }

  return (
    <div class="space-y-6">
      {/* Header with title and connection status */}
      <div class="flex items-center justify-between">
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Live View</h1>
        <span class={`flex items-center text-sm ${isConnected ? 'text-green-600' : 'text-red-600'}`}>
          <span class={`w-2 h-2 rounded-full mr-2 ${isConnected ? 'bg-green-600' : 'bg-red-600'}`}></span>
          {isConnected ? 'Live' : 'Offline'}
        </span>
      </div>

      {/* Main content grid */}
      <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Spectrogram Display */}
        <div class="lg:col-span-2 card">
          <div class="p-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Spectrogram</h2>
            {info?.available && (
              <span class="text-xs text-gray-500 dark:text-gray-400">
                Auto-refresh: {info.refresh_seconds}s
              </span>
            )}
          </div>
          <div class="p-4">
            {info?.available ? (
              <img
                src={imageUrl}
                alt="Live Spectrogram"
                class="w-full h-auto rounded-lg shadow-sm bg-gray-100 dark:bg-gray-800"
                onError={(e) => {
                  // Show placeholder on error
                  (e.target as HTMLImageElement).style.display = 'none';
                }}
              />
            ) : (
              <div class="flex items-center justify-center h-64 bg-gray-100 dark:bg-gray-800 rounded-lg">
                <div class="text-center">
                  <svg class="mx-auto h-12 w-12 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3" />
                  </svg>
                  <p class="mt-2 text-gray-500 dark:text-gray-400">No spectrogram available</p>
                  <p class="text-sm text-gray-400 dark:text-gray-500">Waiting for audio analysis...</p>
                </div>
              </div>
            )}
          </div>

          {/* Audio Player */}
          {info?.livestream_url && (
            <div class="p-4 border-t border-gray-200 dark:border-gray-700">
              <div class="flex items-center space-x-4">
                <button
                  onClick={toggleAudio}
                  class="flex items-center justify-center w-10 h-10 rounded-full bg-primary-600 hover:bg-primary-700 text-white transition-colors"
                  aria-label={isPlaying ? 'Pause' : 'Play'}
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
                <div class="flex-1">
                  <p class="text-sm font-medium text-gray-900 dark:text-white">Live Audio Stream</p>
                  <p class="text-xs text-gray-500 dark:text-gray-400">
                    {isPlaying ? 'Playing...' : 'Click to listen'}
                  </p>
                </div>
              </div>
              <audio
                ref={audioRef}
                src={info.livestream_url}
                preload="none"
                onPlay={handleAudioPlay}
                onPause={handleAudioPause}
                class="hidden"
              />
            </div>
          )}
        </div>

        {/* Recent Detections Sidebar */}
        <div class="card">
          <div class="p-4 border-b border-gray-200 dark:border-gray-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Recent Detections</h2>
          </div>
          <div class="divide-y divide-gray-200 dark:divide-gray-700 max-h-[600px] overflow-y-auto">
            {recentDetections.length === 0 ? (
              <div class="p-4 text-center text-gray-500 dark:text-gray-400">
                No detections yet today
              </div>
            ) : (
              recentDetections.map((detection, index) => (
                <div
                  key={`${detection.time}-${detection.sci_name}-${index}`}
                  class="p-4 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
                >
                  <div class="flex items-start justify-between">
                    <div class="min-w-0 flex-1">
                      <p class="font-medium text-gray-900 dark:text-white truncate">
                        {detection.com_name}
                      </p>
                      <p class="text-sm text-gray-500 dark:text-gray-400 italic truncate">
                        {detection.sci_name}
                      </p>
                    </div>
                    <span class="ml-2 inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200">
                      {formatConfidence(detection.confidence)}
                    </span>
                  </div>
                  <p class="mt-1 text-xs text-gray-400 dark:text-gray-500">
                    {detection.time}
                  </p>
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
