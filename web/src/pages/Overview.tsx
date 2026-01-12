import { useState, useEffect, useCallback } from 'preact/hooks';
import type { JSX } from 'preact';
import { useWebSocket } from '../hooks/useWebSocket';
import { fetchStats, fetchDetections, fetchHeatmapToday } from '../hooks/useApi';
import type { StatsResponse, Detection, DetectionNotification, HeatmapResponse } from '../types/api';
import { DetectionList } from '../components/DetectionList';
import { StatsCards } from '../components/StatsCards';
import { BirdActivityHeatmap } from '../components/BirdActivityHeatmap';

/**
 * Overview page component.
 */
export function Overview(): JSX.Element {
  const [stats, setStats] = useState<StatsResponse | null>(null);
  const [recentDetections, setRecentDetections] = useState<Detection[]>([]);
  const [heatmapData, setHeatmapData] = useState<HeatmapResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [heatmapLoading, setHeatmapLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // WebSocket connection
  const wsUrl = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws`;
  const { isConnected, subscribe } = useWebSocket(wsUrl);

  // Load initial data
  useEffect(() => {
    async function loadData() {
      try {
        setLoading(true);
        setHeatmapLoading(true);
        const [statsData, detectionsData, heatmap] = await Promise.all([
          fetchStats({ include_top_species: 'true', top_limit: 5 }),
          fetchDetections({ per_page: 10 }),
          fetchHeatmapToday(),
        ]);
        setStats(statsData);
        setRecentDetections(detectionsData.detections || []);
        setHeatmapData(heatmap);
        setError(null);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
      } finally {
        setLoading(false);
        setHeatmapLoading(false);
      }
    }
    loadData();
  }, []);

  // Update heatmap with new detection
  const updateHeatmapWithDetection = useCallback((detection: DetectionNotification) => {
    setHeatmapData((prev) => {
      if (!prev) return prev;

      // Extract hour from time (format: "HH:MM:SS")
      const hour = parseInt(detection.time.split(':')[0], 10);
      if (hour < 0 || hour > 23) return prev;

      // Find species index
      const speciesIdx = prev.species.indexOf(detection.com_name);

      if (speciesIdx >= 0) {
        // Species exists, increment count
        const newData = prev.data.map((row, idx) => {
          if (idx === speciesIdx) {
            const newRow = [...row];
            newRow[hour] = (newRow[hour] || 0) + 1;
            return newRow;
          }
          return row;
        });

        return {
          ...prev,
          data: newData,
          total_detections: prev.total_detections + 1,
        };
      } else {
        // New species, add new row
        const newRow = new Array(24).fill(0);
        newRow[hour] = 1;

        return {
          ...prev,
          species: [...prev.species, detection.com_name],
          data: [...prev.data, newRow],
          total_detections: prev.total_detections + 1,
        };
      }
    });
  }, []);

  // Subscribe to real-time detection updates
  useEffect(() => {
    const unsubscribe = subscribe<DetectionNotification>('detection', (payload) => {
      // Convert notification to Detection format
      const detection: Detection = {
        date: payload.date,
        time: payload.time,
        sci_name: payload.sci_name,
        com_name: payload.com_name,
        confidence: payload.confidence,
        file_name: payload.file_name,
      };

      // Add new detection to the top of the list
      setRecentDetections((prev) => [detection, ...prev.slice(0, 9)]);

      // Update stats
      setStats((prev) => {
        if (!prev) return prev;
        return {
          ...prev,
          detections_today: prev.detections_today + 1,
        };
      });

      // Update heatmap in real-time
      updateHeatmapWithDetection(payload);
    });

    return unsubscribe;
  }, [subscribe, updateHeatmapWithDetection]);

  if (loading) {
    return (
      <div class="flex items-center justify-center h-64">
        <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
      </div>
    );
  }

  if (error) {
    return (
      <div class="card p-6 text-center">
        <p class="text-red-600 dark:text-red-400">Error: {error}</p>
        <button
          class="btn btn-primary mt-4"
          onClick={() => window.location.reload()}
        >
          Retry
        </button>
      </div>
    );
  }

  return (
    <div class="space-y-6">
      {/* Connection Status */}
      <div class="flex items-center justify-end">
        <span class={`flex items-center text-sm ${isConnected ? 'text-green-600' : 'text-red-600'}`}>
          <span class={`w-2 h-2 rounded-full mr-2 ${isConnected ? 'bg-green-600' : 'bg-red-600'}`}></span>
          {isConnected ? 'Connected' : 'Disconnected'}
        </span>
      </div>

      {/* Stats Cards */}
      {stats && <StatsCards stats={stats} />}

      {/* Bird Activity Heatmap */}
      <BirdActivityHeatmap data={heatmapData} loading={heatmapLoading} />

      {/* Recent Detections */}
      <div class="card">
        <div class="p-4 border-b border-gray-200 dark:border-gray-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Recent Detections</h2>
        </div>
        <DetectionList detections={recentDetections} />
      </div>

      {/* Top Species */}
      {stats?.top_species && stats.top_species.length > 0 && (
        <div class="card">
          <div class="p-4 border-b border-gray-200 dark:border-gray-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Top Species (Last 7 Days)</h2>
          </div>
          <div class="p-4">
            <ul class="space-y-2">
              {stats.top_species.map((species, index) => (
                <li key={species.sci_name} class="flex items-center justify-between">
                  <span class="flex items-center">
                    <span class="w-6 text-gray-500 dark:text-gray-400">{index + 1}.</span>
                    <span class="font-medium text-gray-900 dark:text-white">{species.com_name}</span>
                    <span class="ml-2 text-sm text-gray-500 dark:text-gray-400 italic">{species.sci_name}</span>
                  </span>
                  <span class="text-primary-600 font-medium">{species.detection_count}</span>
                </li>
              ))}
            </ul>
          </div>
        </div>
      )}
    </div>
  );
}
