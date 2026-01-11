import type { JSX } from 'preact';
import { useState, useEffect } from 'preact/hooks';
import { fetchStats } from '../hooks/useApi';
import type { StatsResponse } from '../types/api';

/**
 * StatsHeader - Displays summary statistics at the top of the Today's Detections page.
 * Shows total detections, today's count, last hour count, and species counts.
 */
export function StatsHeader(): JSX.Element {
  const [stats, setStats] = useState<StatsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function loadStats() {
      try {
        setLoading(true);
        const data = await fetchStats();
        setStats(data);
        setError(null);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load stats');
      } finally {
        setLoading(false);
      }
    }
    loadStats();
  }, []);

  if (loading) {
    return (
      <div class="grid grid-cols-2 md:grid-cols-5 gap-4 mb-6">
        {[...Array(5)].map((_, i) => (
          <div key={i} class="bg-white dark:bg-gray-800 rounded-lg p-4 animate-pulse">
            <div class="h-4 bg-gray-200 dark:bg-gray-700 rounded w-20 mb-2"></div>
            <div class="h-8 bg-gray-200 dark:bg-gray-700 rounded w-16"></div>
          </div>
        ))}
      </div>
    );
  }

  if (error || !stats) {
    return (
      <div class="bg-red-50 dark:bg-red-900/20 rounded-lg p-4 mb-6 text-red-600 dark:text-red-400">
        {error || 'Failed to load statistics'}
      </div>
    );
  }

  const statItems = [
    { label: 'Total', value: stats.total_detections, color: 'text-gray-900 dark:text-white' },
    { label: 'Today', value: stats.detections_today, color: 'text-blue-600 dark:text-blue-400' },
    { label: 'Last Hour', value: stats.detections_last_hour, color: 'text-green-600 dark:text-green-400' },
    { label: 'Species Total', value: stats.total_species, color: 'text-purple-600 dark:text-purple-400' },
    { label: 'Species Today', value: stats.species_today, color: 'text-orange-600 dark:text-orange-400' },
  ];

  return (
    <div class="grid grid-cols-2 md:grid-cols-5 gap-4 mb-6">
      {statItems.map((item) => (
        <div
          key={item.label}
          class="bg-white dark:bg-gray-800 rounded-lg p-4 shadow-sm border border-gray-200 dark:border-gray-700"
        >
          <div class="text-sm text-gray-500 dark:text-gray-400">{item.label}</div>
          <div class={`text-2xl font-bold ${item.color}`}>
            {item.value.toLocaleString()}
          </div>
        </div>
      ))}
    </div>
  );
}
