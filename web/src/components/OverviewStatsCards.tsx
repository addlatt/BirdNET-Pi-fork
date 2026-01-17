import type { JSX } from 'preact';
import type { StatsResponse } from '../types/api';

/**
 * OverviewStatsCards props
 */
interface OverviewStatsCardsProps {
  /** Stats data from API */
  stats: StatsResponse;
}

/**
 * Today-focused stats cards for the Overview page.
 * Only uses fields that are populated without extra API params:
 * - detections_today, species_today, detections_last_hour
 * - total_detections, total_species
 * - top_species (optional)
 */
export function OverviewStatsCards({ stats }: OverviewStatsCardsProps): JSX.Element {
  const topSpecies = stats.top_species?.[0];

  return (
    <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-3">
      {/* Detections Today */}
      <StatCard
        label="Detections Today"
        value={stats.detections_today || 0}
        subtext={stats.detections_today === 1 ? 'detection' : 'detections'}
        color="green"
      />

      {/* Species Today */}
      <StatCard
        label="Species Today"
        value={stats.species_today || 0}
        subtext={`of ${stats.total_species || 0} total`}
        color="amber"
      />

      {/* Last Hour */}
      <StatCard
        label="Last Hour"
        value={stats.detections_last_hour || 0}
        subtext={stats.detections_last_hour === 1 ? 'detection' : 'detections'}
        color="blue"
      />

      {/* All-Time Detections */}
      <StatCard
        label="All-Time"
        value={formatNumber(stats.total_detections || 0)}
        subtext="total detections"
        color="indigo"
      />

      {/* Top Species This Week */}
      {topSpecies ? (
        <StatCard
          label="Top This Week"
          value={topSpecies.com_name}
          subtext={`${topSpecies.detection_count} detections`}
          color="teal"
          smallValue
        />
      ) : (
        <StatCard
          label="Total Species"
          value={stats.total_species || 0}
          subtext="species recorded"
          color="teal"
        />
      )}
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
  smallValue?: boolean;
}

function StatCard({ label, value, subtext, color, smallValue }: StatCardProps): JSX.Element {
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
      <p class={`font-bold text-gray-900 dark:text-white mt-0.5 ${smallValue ? 'text-sm truncate' : 'text-xl'}`}>
        {value}
      </p>
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
