import { useState, useEffect, useCallback, useRef } from 'preact/hooks';
import type { JSX } from 'preact';
import { useWebSocket } from '../hooks/useWebSocket';
import {
  fetchSpectrogramInfo,
  fetchRecentDetections,
} from '../hooks/useApi';
import type {
  SpectrogramInfoResponse,
  RecentDetection,
  DetectionNotification,
} from '../types/api';

interface DetectionOverlay {
  text: string;
  x: number;
  y: number;
  opacity: number;
  timestamp: number;
}

/**
 * Spectrogram/Live View page component.
 * Real-time audio spectrogram visualization using Web Audio API.
 */
export function Spectrogram(): JSX.Element {
  const [info, setInfo] = useState<SpectrogramInfoResponse | null>(null);
  const [recentDetections, setRecentDetections] = useState<RecentDetection[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [streamLoading, setStreamLoading] = useState(false);
  const [gain, setGain] = useState(100);
  const [compressionEnabled, setCompressionEnabled] = useState(false);

  // Refs for audio and canvas
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const audioRef = useRef<HTMLAudioElement>(null);
  const audioContextRef = useRef<AudioContext | null>(null);
  const analyserRef = useRef<AnalyserNode | null>(null);
  const gainNodeRef = useRef<GainNode | null>(null);
  const compressorRef = useRef<DynamicsCompressorNode | null>(null);
  const sourceRef = useRef<MediaElementAudioSourceNode | null>(null);
  const animationRef = useRef<number | null>(null);
  const detectionsOverlayRef = useRef<DetectionOverlay[]>([]);
  const fpsRef = useRef<number[]>([]);
  const avgFpsRef = useRef<number>(60);
  const lastFrameTimeRef = useRef<number>(0);
  const audioInitializedRef = useRef(false);

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

  // Initialize Web Audio API
  const initializeAudio = useCallback(() => {
    if (!audioRef.current || audioInitializedRef.current) return;

    try {
      const audioContext = new AudioContext();
      const analyser = audioContext.createAnalyser();
      analyser.fftSize = 2048;

      const source = audioContext.createMediaElementSource(audioRef.current);
      const gainNode = audioContext.createGain();
      gainNode.gain.value = gain / 100;

      const compressor = audioContext.createDynamicsCompressor();
      compressor.threshold.value = -50;
      compressor.knee.value = 40;
      compressor.ratio.value = 12;
      compressor.attack.value = 0;
      compressor.release.value = 0.25;

      // Default connection: source -> gain -> analyser -> destination
      source.connect(gainNode);
      gainNode.connect(analyser);
      analyser.connect(audioContext.destination);

      audioContextRef.current = audioContext;
      analyserRef.current = analyser;
      gainNodeRef.current = gainNode;
      compressorRef.current = compressor;
      sourceRef.current = source;
      audioInitializedRef.current = true;

      // Start the visualization loop
      startVisualization();
    } catch (err) {
      console.error('Failed to initialize audio:', err);
    }
  }, [gain]);

  // Canvas visualization loop
  const startVisualization = useCallback(() => {
    const canvas = canvasRef.current;
    const analyser = analyserRef.current;
    if (!canvas || !analyser) return;

    const ctx = canvas.getContext('2d', { willReadFrequently: true });
    if (!ctx) return;

    const dataArray = new Uint8Array(analyser.frequencyBinCount);
    const bufferLength = dataArray.length;

    // Set canvas size
    const resize = () => {
      canvas.width = canvas.clientWidth;
      canvas.height = canvas.clientHeight;
      // Fill with dark background
      ctx.fillStyle = 'hsl(280, 100%, 10%)';
      ctx.fillRect(0, 0, canvas.width, canvas.height);
    };
    resize();
    window.addEventListener('resize', resize);

    const W = canvas.width;
    const H = canvas.height;
    const h = H / bufferLength + 0.9;
    const x = W - 1;

    const loop = (time: number) => {
      // Calculate FPS
      if (lastFrameTimeRef.current) {
        const fps = Math.round(1000 / (time - lastFrameTimeRef.current));
        if (fps > 0 && fps < 200) {
          fpsRef.current.push(fps);
          if (fpsRef.current.length > 60) fpsRef.current.shift();
          avgFpsRef.current = fpsRef.current.reduce((a, b) => a + b, 0) / fpsRef.current.length;
        }
      }
      lastFrameTimeRef.current = time;

      // Shift existing image left by 1 pixel
      const imgData = ctx.getImageData(1, 0, W - 1, H);
      ctx.fillStyle = 'hsl(280, 100%, 10%)';
      ctx.fillRect(0, 0, W, H);
      ctx.putImageData(imgData, 0, 0);

      // Get frequency data
      analyser.getByteFrequencyData(dataArray);

      // Draw new frequency bars on right edge
      for (let i = 0; i < bufferLength; i++) {
        const rat = dataArray[i] / 255;
        const hue = Math.round((rat * 120) + 280) % 360;
        const sat = '100%';
        const lit = 10 + (70 * rat) + '%';
        ctx.beginPath();
        ctx.strokeStyle = `hsl(${hue}, ${sat}, ${lit})`;
        ctx.moveTo(x, H - (i * h));
        ctx.lineTo(x, H - (i * h + h));
        ctx.stroke();
      }

      // Draw detection overlays
      const now = Date.now();
      detectionsOverlayRef.current = detectionsOverlayRef.current.filter(d => now - d.timestamp < 10000);
      ctx.textAlign = 'center';
      ctx.font = '15px system-ui, sans-serif';
      for (const detection of detectionsOverlayRef.current) {
        // Move detection left as spectrogram scrolls
        detection.x -= 1;
        const opacity = Math.max(0.2, detection.opacity);
        ctx.fillStyle = `rgba(255, 255, 255, ${opacity})`;
        ctx.fillText(detection.text, detection.x, detection.y);
      }

      animationRef.current = requestAnimationFrame(loop);
    };

    animationRef.current = requestAnimationFrame(loop);

    return () => {
      window.removeEventListener('resize', resize);
      if (animationRef.current) {
        cancelAnimationFrame(animationRef.current);
      }
    };
  }, []);

  // Handle play/pause
  const toggleAudio = useCallback(async () => {
    if (!audioRef.current) return;

    if (isPlaying) {
      audioRef.current.pause();
      setIsPlaying(false);
      setStreamLoading(false);
    } else {
      // Show loading indicator while stream connects
      setStreamLoading(true);

      // Initialize audio on first play (required for user gesture)
      if (!audioInitializedRef.current) {
        initializeAudio();
      }
      // Resume audio context if suspended
      if (audioContextRef.current?.state === 'suspended') {
        await audioContextRef.current.resume();
      }
      try {
        await audioRef.current.play();
        setIsPlaying(true);
      } catch (err) {
        console.error('Failed to play audio:', err);
        setStreamLoading(false);
      }
    }
  }, [isPlaying, initializeAudio]);

  // Listen for audio playing event to hide loading spinner
  useEffect(() => {
    const audio = audioRef.current;
    if (!audio) return;

    const handlePlaying = () => {
      setStreamLoading(false);
    };

    const handleWaiting = () => {
      if (isPlaying) setStreamLoading(true);
    };

    const handleError = () => {
      setStreamLoading(false);
    };

    audio.addEventListener('playing', handlePlaying);
    audio.addEventListener('waiting', handleWaiting);
    audio.addEventListener('error', handleError);

    return () => {
      audio.removeEventListener('playing', handlePlaying);
      audio.removeEventListener('waiting', handleWaiting);
      audio.removeEventListener('error', handleError);
    };
  }, [isPlaying]);

  // Handle gain change
  const handleGainChange = useCallback((e: Event) => {
    const value = parseInt((e.target as HTMLInputElement).value);
    setGain(value);
    if (gainNodeRef.current && audioContextRef.current) {
      gainNodeRef.current.gain.setValueAtTime(value / 50, audioContextRef.current.currentTime);
    }
  }, []);

  // Handle compression toggle
  const handleCompressionToggle = useCallback(() => {
    if (!sourceRef.current || !gainNodeRef.current || !analyserRef.current ||
        !compressorRef.current || !audioContextRef.current) return;

    const newState = !compressionEnabled;
    setCompressionEnabled(newState);

    // Disconnect all
    sourceRef.current.disconnect();
    gainNodeRef.current.disconnect();
    compressorRef.current.disconnect();
    analyserRef.current.disconnect();

    if (newState) {
      // With compression: source -> compressor -> analyser -> gain -> destination
      sourceRef.current.connect(compressorRef.current);
      compressorRef.current.connect(analyserRef.current);
      analyserRef.current.connect(gainNodeRef.current);
      gainNodeRef.current.connect(audioContextRef.current.destination);
    } else {
      // Without compression: source -> gain -> analyser -> destination
      sourceRef.current.connect(gainNodeRef.current);
      gainNodeRef.current.connect(analyserRef.current);
      analyserRef.current.connect(audioContextRef.current.destination);
    }
  }, [compressionEnabled]);

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

      // Add to sidebar list
      setRecentDetections((prev) => [detection, ...prev].slice(0, 10));

      // Add overlay on canvas
      const canvas = canvasRef.current;
      if (canvas) {
        const yOffset = (detectionsOverlayRef.current.length % 4) * 20;
        detectionsOverlayRef.current.push({
          text: payload.com_name,
          x: canvas.width - 100,
          y: canvas.height * 0.5 + yOffset,
          opacity: payload.confidence,
          timestamp: Date.now(),
        });
      }
    });

    return unsubscribe;
  }, [subscribe]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (animationRef.current) {
        cancelAnimationFrame(animationRef.current);
      }
      if (audioContextRef.current) {
        audioContextRef.current.close();
      }
    };
  }, []);

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
    <div class="space-y-4">
      {/* Header with title and connection status */}
      <div class="flex items-center justify-between">
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Live View</h1>
        <span class={`flex items-center text-sm ${isConnected ? 'text-green-600' : 'text-red-600'}`}>
          <span class={`w-2 h-2 rounded-full mr-2 ${isConnected ? 'bg-green-600' : 'bg-red-600'}`}></span>
          {isConnected ? 'Live' : 'Offline'}
        </span>
      </div>

      {/* Main content grid */}
      <div class="grid grid-cols-1 lg:grid-cols-4 gap-4">
        {/* Spectrogram Canvas */}
        <div class="lg:col-span-3 card overflow-hidden">
          {/* Controls bar */}
          <div class="p-3 border-b border-gray-200 dark:border-gray-700 flex flex-wrap items-center gap-4">
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

            <div class="flex items-center gap-2">
              <label class="text-sm text-gray-600 dark:text-gray-400">Gain:</label>
              <input
                type="range"
                min="0"
                max="250"
                value={gain}
                onInput={handleGainChange}
                class="w-24"
              />
              <span class="text-sm text-gray-600 dark:text-gray-400 w-12">{gain}%</span>
            </div>

            <label class="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
              <input
                type="checkbox"
                checked={compressionEnabled}
                onChange={handleCompressionToggle}
                disabled={!audioInitializedRef.current}
              />
              Compression
            </label>
          </div>

          {/* Canvas container */}
          <div class="relative bg-gray-900" style={{ height: '60vh' }}>
            {/* Loading spinner overlay */}
            {streamLoading && (
              <div class="absolute inset-0 flex items-center justify-center bg-gray-900/80 z-10">
                <div class="text-center">
                  <div class="animate-spin rounded-full h-16 w-16 border-4 border-primary-600 border-t-transparent mx-auto mb-4"></div>
                  <p class="text-gray-400 text-lg">Connecting to stream...</p>
                </div>
              </div>
            )}
            {/* Play prompt overlay */}
            {!isPlaying && !streamLoading && (
              <div class="absolute inset-0 flex items-center justify-center bg-gray-900/80 z-10">
                <div class="text-center">
                  <svg class="mx-auto h-16 w-16 text-gray-400 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  <p class="text-gray-400 text-lg">Click play to start live spectrogram</p>
                </div>
              </div>
            )}
            <canvas
              ref={canvasRef}
              class="w-full h-full"
            />
          </div>

          {/* Hidden audio element */}
          <audio
            ref={audioRef}
            src={info?.livestream_url || 'http://localhost:8000/stream'}
            crossOrigin="anonymous"
            preload="none"
            class="hidden"
          />
        </div>

        {/* Recent Detections Sidebar */}
        <div class="card">
          <div class="p-3 border-b border-gray-200 dark:border-gray-700">
            <h2 class="font-semibold text-gray-900 dark:text-white">Recent Detections</h2>
          </div>
          <div class="divide-y divide-gray-200 dark:divide-gray-700 max-h-[60vh] overflow-y-auto">
            {recentDetections.length === 0 ? (
              <div class="p-4 text-center text-gray-500 dark:text-gray-400 text-sm">
                No detections yet today
              </div>
            ) : (
              recentDetections.map((detection, index) => (
                <div
                  key={`${detection.time}-${detection.sci_name}-${index}`}
                  class="p-3 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
                >
                  <div class="flex items-start justify-between gap-2">
                    <div class="min-w-0 flex-1">
                      <p class="font-medium text-gray-900 dark:text-white text-sm truncate">
                        {detection.com_name}
                      </p>
                      <p class="text-xs text-gray-500 dark:text-gray-400 italic truncate">
                        {detection.sci_name}
                      </p>
                    </div>
                    <span class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200">
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
