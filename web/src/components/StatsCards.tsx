import type { JSX } from 'preact';
import type { StatsResponse } from '../types/api';

/**
 * StatsCards props
 */
interface StatsCardsProps {
  /** Stats data from API */
  stats: StatsResponse;
}

/**
 * Compact stats summary cards - no icons, higher information density.
 */
export function StatsCards({ stats }: StatsCardsProps): JSX.Element {
  // Compute weekly total from daily stats
  const weeklyTotal = stats.daily_stats?.reduce((sum, d) => sum + d.detection_count, 0) || 0;

  // Compute average confidence from daily stats
  const avgConfidence = stats.daily_stats && stats.daily_stats.length > 0
    ? stats.daily_stats.reduce((sum, d) => sum + d.avg_confidence, 0) / stats.daily_stats.length
    : 0;

  return (
    <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-3">
      {/* Last Hour */}
      <StatCard
        label="Last Hour"
        value={stats.detections_last_hour || 0}
        subtext={stats.detections_last_hour === 1 ? 'detection' : 'detections'}
        color="blue"
      />

      {/* Today */}
      <StatCard
        label="Today"
        value={stats.detections_today || 0}
        subtext={`${stats.species_today || 0} species`}
        color="green"
      />

      {/* This Week (based on selected period) */}
      <StatCard
        label="Period Total"
        value={formatNumber(weeklyTotal)}
        subtext="detections"
        color="purple"
      />

      {/* Species Today */}
      <StatCard
        label="Species Today"
        value={stats.species_today || 0}
        subtext={`of ${stats.total_species || 0} total`}
        color="amber"
      />

      {/* Total Species */}
      <StatCard
        label="All-Time Species"
        value={stats.total_species || 0}
        subtext={`${formatNumber(stats.total_detections || 0)} detections`}
        color="indigo"
      />

      {/* Average Confidence */}
      <StatCard
        label="Avg Confidence"
        value={avgConfidence > 0 ? `${Math.round(avgConfidence * 100)}%` : '-'}
        subtext="for period"
        color="teal"
      />
    </div>
  );
}

/**
 * Individual stat card component.
 */
interface StatCardProps {
  label: string;
  value: string | number;
  subtext: string;
  color: 'blue' | 'green' | 'purple' | 'amber' | 'indigo' | 'teal';
}

function StatCard({ label, value, subtext, color }: StatCardProps): JSX.Element {
  const colorClasses: Record<string, string> = {
    blue: 'border-l-blue-500',
    green: 'border-l-green-500',
    purple: 'border-l-purple-500',
    amber: 'border-l-amber-500',
    indigo: 'border-l-indigo-500',
    teal: 'border-l-teal-500',
  };

  return (
    <div class={`card p-3 border-l-4 ${colorClasses[color]}`}>
      <p class="text-xs text-gray-500 dark:text-gray-400 uppercase tracking-wide">{label}</p>
      <p class="text-xl font-bold text-gray-900 dark:text-white mt-0.5">{value}</p>
      <p class="text-xs text-gray-400 dark:text-gray-500">{subtext}</p>
    </div>
  );
}

/**
 * Format large numbers with K/M suffix.
 */
function formatNumber(num: number): string {
  if (num >= 1000000) {
    return (num / 1000000).toFixed(1) + 'M';
  }
  if (num >= 1000) {
    return (num / 1000).toFixed(1) + 'K';
  }
  return num.toString();
}
