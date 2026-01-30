import { useState, useEffect, useCallback } from 'preact/hooks';
import type { JSX } from 'preact';
import { useWebSocket } from '../hooks/useWebSocket';
import { fetchStats, fetchSpeciesRanking, fetchHeatmapToday } from '../hooks/useApi';
import type { StatsResponse, SpeciesRankingEntry, DetectionNotification, HeatmapResponse } from '../types/api';
import { SpeciesRankingList } from '../components/SpeciesRankingList';
import { OverviewStatsCards } from '../components/OverviewStatsCards';
import { BirdActivityHeatmap } from '../components/BirdActivityHeatmap';

type RankingPeriod = 'today' | 'week' | 'month' | 'all';

/**
 * Overview page component.
 */
export function Overview(): JSX.Element {
  const [stats, setStats] = useState<StatsResponse | null>(null);
  const [speciesRanking, setSpeciesRanking] = useState<SpeciesRankingEntry[]>([]);
  const [heatmapData, setHeatmapData] = useState<HeatmapResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [heatmapLoading, setHeatmapLoading] = useState(true);
  const [rankingLoading, setRankingLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [rankingPeriod, setRankingPeriod] = useState<RankingPeriod>('today');

  // WebSocket connection
  const wsUrl = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws`;
  const { isConnected, subscribe } = useWebSocket(wsUrl);

  // Load initial data
  useEffect(() => {
    async function loadData() {
      try {
        setLoading(true);
        setHeatmapLoading(true);
        const [statsData, rankingData, heatmap] = await Promise.all([
          fetchStats({ include_top_species: 'true', top_limit: 5 }),
          fetchSpeciesRanking({ period: 'today' }),
          fetchHeatmapToday(),
        ]);
        setStats(statsData);
        setSpeciesRanking(rankingData.species || []);
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

  // Load ranking when period changes
  const handlePeriodChange = useCallback(async (period: RankingPeriod) => {
    setRankingPeriod(period);
    setRankingLoading(true);
    try {
      const rankingData = await fetchSpeciesRanking({ period });
      setSpeciesRanking(rankingData.species || []);
    } catch (err) {
      console.error('Failed to fetch ranking:', err);
    } finally {
      setRankingLoading(false);
    }
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
      // Only update ranking in real-time if viewing today or all
      if (rankingPeriod === 'today' || rankingPeriod === 'all') {
        setSpeciesRanking((prev) => {
          const existingIndex = prev.findIndex((s) => s.sci_name === payload.sci_name);
          if (existingIndex >= 0) {
            // Update existing species: increment count and update latest detection
            const updated = [...prev];
            updated[existingIndex] = {
              ...updated[existingIndex],
              detection_count: updated[existingIndex].detection_count + 1,
              latest_date: payload.date.split('T')[0],
              latest_time: payload.time,
              latest_file: payload.file_name,
              latest_confidence: payload.confidence,
            };
            // Re-sort by detection count
            updated.sort((a, b) => b.detection_count - a.detection_count);
            return updated;
          } else {
            // New species - add to list
            const newEntry: SpeciesRankingEntry = {
              sci_name: payload.sci_name,
              com_name: payload.com_name,
              detection_count: 1,
              latest_date: payload.date.split('T')[0],
              latest_time: payload.time,
              latest_file: payload.file_name,
              latest_confidence: payload.confidence,
              best_date: payload.date.split('T')[0],
              best_time: payload.time,
              best_file: payload.file_name,
              best_confidence: payload.confidence,
            };
            // Add and sort by detection count
            return [...prev, newEntry].sort((a, b) => b.detection_count - a.detection_count);
          }
        });
      }

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
  }, [subscribe, updateHeatmapWithDetection, rankingPeriod]);

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

  const periodOptions: { value: RankingPeriod; label: string }[] = [
    { value: 'today', label: 'Today' },
    { value: 'week', label: 'Week' },
    { value: 'month', label: 'Month' },
    { value: 'all', label: 'All Time' },
  ];

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
      {stats && <OverviewStatsCards stats={stats} />}

      {/* Bird Activity Heatmap */}
      <BirdActivityHeatmap data={heatmapData} loading={heatmapLoading} />

      {/* Species Ranking */}
      <div class="card">
        <div class="p-4 border-b border-gray-200 dark:border-gray-700 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Species Ranking</h2>
          <div class="flex items-center gap-1">
            {periodOptions.map((option) => (
              <button
                key={option.value}
                onClick={() => handlePeriodChange(option.value)}
                disabled={rankingLoading}
                class={`px-3 py-1.5 text-sm font-medium rounded transition-colors ${
                  rankingPeriod === option.value
                    ? 'bg-primary-600 text-white'
                    : 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-600'
                } disabled:opacity-50`}
              >
                {option.label}
              </button>
            ))}
          </div>
        </div>
        {rankingLoading ? (
          <div class="flex items-center justify-center h-32">
            <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
          </div>
        ) : (
          <SpeciesRankingList species={speciesRanking} />
        )}
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
