import { useState, useEffect, useMemo } from 'preact/hooks';
import type { JSX } from 'preact';
import { fetchStats, fetchSpecies } from '../hooks/useApi';
import type { StatsResponse, Species, ListSpeciesParams, HourlyStat, NewSpecies } from '../types/api';
import { StatsCards } from '../components/StatsCards';
import { SpeciesDetail } from '../components/SpeciesDetail';

/**
 * Sort option type
 */
interface SortOption {
  value: ListSpeciesParams['sort'];
  label: string;
}

const SORT_OPTIONS: SortOption[] = [
  { value: 'occurrences', label: 'Count' },
  { value: 'alphabetical', label: 'A-Z' },
  { value: 'confidence', label: 'Conf' },
  { value: 'date', label: 'Recent' },
];

/**
 * Activity time periods for bird watchers
 */
interface ActivityPeriod {
  label: string;
  hours: number[];
  count: number;
}

/**
 * Stats page component.
 */
export function Stats(): JSX.Element {
  const [stats, setStats] = useState<StatsResponse | null>(null);
  const [species, setSpecies] = useState<Species[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [days, setDays] = useState(7);
  const [sortBy, setSortBy] = useState<ListSpeciesParams['sort']>('occurrences');
  const [selectedSpecies, setSelectedSpecies] = useState<string | null>(null);

  // Load stats data
  useEffect(() => {
    async function loadStats() {
      try {
        const statsData = await fetchStats({
          days,
          include_daily: 'true',
          include_hourly: 'true',
          include_top_species: 'true',
          include_new_species: 'true',
          top_limit: 10,
        });
        setStats(statsData);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
      }
    }
    loadStats();
  }, [days]);

  // Load species data (separate effect for sort changes)
  useEffect(() => {
    async function loadSpecies() {
      try {
        setLoading(true);
        const speciesData = await fetchSpecies({ sort: sortBy });
        setSpecies(speciesData.species || []);
        setError(null);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
      } finally {
        setLoading(false);
      }
    }
    loadSpecies();
  }, [sortBy]);

  // Compute activity patterns from hourly data
  const activityPatterns = useMemo(() => {
    if (!stats?.hourly_distribution) return null;
    return computeActivityPatterns(stats.hourly_distribution);
  }, [stats?.hourly_distribution]);

  // Find peak hour
  const peakHour = useMemo(() => {
    if (!stats?.hourly_distribution || stats.hourly_distribution.length === 0) return null;
    const peak = stats.hourly_distribution.reduce((max, h) =>
      h.detection_count > max.detection_count ? h : max
    );
    return peak.detection_count > 0 ? peak : null;
  }, [stats?.hourly_distribution]);

  if (loading && !stats) {
    return (
      <div class="flex items-center justify-center h-64">
        <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
      </div>
    );
  }

  if (error && !stats) {
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
      <div class="flex items-center justify-between">
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Statistics</h1>
        <select
          class="input w-40"
          value={days}
          onChange={(e) => setDays(parseInt((e.target as HTMLSelectElement).value, 10))}
        >
          <option value={7}>Last 7 days</option>
          <option value={14}>Last 14 days</option>
          <option value={30}>Last 30 days</option>
          <option value={90}>Last 90 days</option>
        </select>
      </div>

      {/* New Species Today Alert */}
      {stats?.new_species_today && stats.new_species_today.length > 0 && (
        <NewSpeciesAlert
          newSpecies={stats.new_species_today}
          onSpeciesClick={setSelectedSpecies}
        />
      )}

      {/* Stats Cards */}
      {stats && <StatsCards stats={stats} />}

      {/* Activity Patterns + Top Species Row */}
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Activity Patterns */}
        {activityPatterns && (
          <div class="card">
            <div class="p-4 border-b border-gray-200 dark:border-gray-700">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Activity Patterns</h2>
              {peakHour && (
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  Peak: {formatHour(peakHour.hour)} ({peakHour.detection_count} detections)
                </p>
              )}
            </div>
            <div class="p-4">
              <div class="space-y-3">
                {activityPatterns.map((period) => (
                  <div key={period.label} class="flex items-center gap-3">
                    <span class="w-24 text-sm text-gray-600 dark:text-gray-400">{period.label}</span>
                    <div class="flex-1 h-4 bg-gray-100 dark:bg-gray-700 rounded-full overflow-hidden">
                      <div
                        class="h-full bg-primary-500 rounded-full transition-all"
                        style={{
                          width: `${getBarWidth(period.count, activityPatterns)}%`,
                          minWidth: period.count > 0 ? '4px' : '0'
                        }}
                      />
                    </div>
                    <span class="w-12 text-right text-sm font-medium text-gray-900 dark:text-white">
                      {period.count}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          </div>
        )}

        {/* Top Species */}
        {stats?.top_species && stats.top_species.length > 0 && (
          <div class="card">
            <div class="p-4 border-b border-gray-200 dark:border-gray-700">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Top Species</h2>
              <p class="text-sm text-gray-500 dark:text-gray-400">Last {days} days</p>
            </div>
            <div class="p-4">
              <div class="space-y-2">
                {stats.top_species.slice(0, 5).map((sp, idx) => {
                  const maxCount = stats.top_species![0].detection_count;
                  const widthPercent = maxCount > 0 ? (sp.detection_count / maxCount) * 100 : 0;
                  return (
                    <div
                      key={sp.sci_name}
                      class="flex items-center gap-3 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800 rounded p-1 -mx-1 transition-colors"
                      onClick={() => setSelectedSpecies(sp.com_name)}
                    >
                      <span class="w-5 text-sm text-gray-400">{idx + 1}.</span>
                      <span class="w-36 truncate text-sm font-medium text-gray-900 dark:text-white">
                        {sp.com_name}
                      </span>
                      <div class="flex-1 h-3 bg-gray-100 dark:bg-gray-700 rounded-full overflow-hidden">
                        <div
                          class="h-full bg-green-500 rounded-full"
                          style={{ width: `${widthPercent}%` }}
                        />
                      </div>
                      <span class="w-10 text-right text-sm font-medium text-gray-700 dark:text-gray-300">
                        {sp.detection_count}
                      </span>
                    </div>
                  );
                })}
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Hourly Distribution + Daily Stats Row */}
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Hourly Distribution with Labels */}
        {stats?.hourly_distribution && stats.hourly_distribution.length > 0 && (
          <div class="card">
            <div class="p-4 border-b border-gray-200 dark:border-gray-700">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Hourly Distribution</h2>
            </div>
            <div class="p-4">
              {/* Period labels with color indicators */}
              <div class="flex text-xs text-gray-500 dark:text-gray-400 mb-1">
                <span class="flex-1 text-center flex items-center justify-center gap-1">
                  <span class="w-2 h-2 rounded-sm bg-gray-400"></span>Night
                </span>
                <span class="flex-1 text-center flex items-center justify-center gap-1">
                  <span class="w-2 h-2 rounded-sm bg-orange-400"></span>Dawn
                </span>
                <span class="flex-1 text-center flex items-center justify-center gap-1">
                  <span class="w-2 h-2 rounded-sm bg-primary-500"></span>Morning
                </span>
                <span class="flex-1 text-center flex items-center justify-center gap-1">
                  <span class="w-2 h-2 rounded-sm bg-primary-400"></span>Afternoon
                </span>
                <span class="flex-1 text-center flex items-center justify-center gap-1">
                  <span class="w-2 h-2 rounded-sm bg-purple-400"></span>Evening
                </span>
              </div>
              {/* Chart */}
              <div class="flex items-end h-32 gap-0.5">
                {Array.from({ length: 24 }, (_, hour) => {
                  const data = stats.hourly_distribution!.find((h) => h.hour === hour);
                  const count = data?.detection_count || 0;
                  const maxCount = Math.max(...stats.hourly_distribution!.map((h) => h.detection_count));
                  const heightPercent = maxCount > 0 ? (count / maxCount) * 100 : 0;
                  const periodColor = getHourColor(hour);

                  return (
                    <div
                      key={hour}
                      class="flex-1 flex flex-col justify-end h-full"
                      title={`${formatHour(hour)}: ${count} detections`}
                    >
                      <div
                        class={`w-full rounded-t transition-all ${periodColor}`}
                        style={{ height: `${Math.max(heightPercent, count > 0 ? 4 : 0)}%` }}
                      />
                    </div>
                  );
                })}
              </div>
              {/* Hour labels */}
              <div class="flex justify-between mt-1 text-xs text-gray-500 dark:text-gray-400">
                <span>12am</span>
                <span>6am</span>
                <span>12pm</span>
                <span>6pm</span>
                <span>12am</span>
              </div>
            </div>
          </div>
        )}

        {/* Daily Stats - Show all days */}
        {stats?.daily_stats && stats.daily_stats.length > 0 && (
          <div class="card">
            <div class="p-4 border-b border-gray-200 dark:border-gray-700">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Daily Breakdown</h2>
            </div>
            <div class="p-4 overflow-y-auto max-h-64">
              <table class="w-full text-sm">
                <thead class="sticky top-0 bg-white dark:bg-gray-800">
                  <tr class="text-left text-gray-500 dark:text-gray-400">
                    <th class="pb-2">Date</th>
                    <th class="pb-2 text-right">Detections</th>
                    <th class="pb-2 text-right">Species</th>
                    <th class="pb-2 text-right">Avg Conf</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-gray-200 dark:divide-gray-700">
                  {stats.daily_stats.map((day) => (
                    <tr key={day.date} class="text-gray-900 dark:text-white">
                      <td class="py-2">{formatDate(day.date)}</td>
                      <td class="py-2 text-right">{day.detection_count}</td>
                      <td class="py-2 text-right">{day.species_count}</td>
                      <td class="py-2 text-right">{Math.round(day.avg_confidence * 100)}%</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </div>

      {/* All Species */}
      <div class="card">
        <div class="p-4 border-b border-gray-200 dark:border-gray-700">
          <div class="flex items-center justify-between flex-wrap gap-3">
            <div class="flex items-center gap-3">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">All Species</h2>
              <span class="text-sm text-gray-500 dark:text-gray-400">{species.length} species</span>
            </div>
            {/* Sort buttons */}
            <div class="flex gap-1">
              {SORT_OPTIONS.map((option) => (
                <button
                  key={option.value}
                  onClick={() => setSortBy(option.value)}
                  class={`px-3 py-1.5 text-sm rounded-md transition-colors ${
                    sortBy === option.value
                      ? 'bg-primary-600 text-white'
                      : 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-600'
                  }`}
                  title={`Sort by ${option.label}`}
                >
                  {option.label}
                </button>
              ))}
            </div>
          </div>
        </div>
        <div class="p-4 overflow-x-auto">
          {loading ? (
            <div class="flex items-center justify-center py-8">
              <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
            </div>
          ) : (
            <table class="w-full text-sm">
              <thead>
                <tr class="text-left text-gray-500 dark:text-gray-400">
                  <th class="pb-2">Common Name</th>
                  <th class="pb-2">Scientific Name</th>
                  <th class="pb-2 text-right">Detections</th>
                  <th class="pb-2 text-right">Max Conf</th>
                  {sortBy === 'date' && <th class="pb-2 text-right">Last Seen</th>}
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-200 dark:divide-gray-700">
                {species.map((s) => (
                  <tr
                    key={s.sci_name}
                    class="text-gray-900 dark:text-white hover:bg-gray-50 dark:hover:bg-gray-700 cursor-pointer transition-colors"
                    onClick={() => setSelectedSpecies(s.com_name)}
                  >
                    <td class="py-2 font-medium">{s.com_name}</td>
                    <td class="py-2 italic text-gray-500 dark:text-gray-400">{s.sci_name}</td>
                    <td class="py-2 text-right">{s.detection_count}</td>
                    <td class="py-2 text-right">{Math.round(s.max_confidence * 100)}%</td>
                    {sortBy === 'date' && (
                      <td class="py-2 text-right">{s.last_seen || '-'}</td>
                    )}
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>

      {/* Species Detail Modal */}
      {selectedSpecies && (
        <SpeciesDetail
          speciesName={selectedSpecies}
          onClose={() => setSelectedSpecies(null)}
        />
      )}
    </div>
  );
}

/**
 * Compute activity patterns from hourly data.
 */
function computeActivityPatterns(hourly: HourlyStat[]): ActivityPeriod[] {
  const periods: ActivityPeriod[] = [
    { label: 'Night', hours: [0, 1, 2, 3, 4], count: 0 },
    { label: 'Dawn', hours: [5, 6, 7], count: 0 },
    { label: 'Morning', hours: [8, 9, 10, 11], count: 0 },
    { label: 'Afternoon', hours: [12, 13, 14, 15, 16], count: 0 },
    { label: 'Evening', hours: [17, 18, 19, 20, 21, 22, 23], count: 0 },
  ];

  for (const h of hourly) {
    for (const period of periods) {
      if (period.hours.includes(h.hour)) {
        period.count += h.detection_count;
        break;
      }
    }
  }

  return periods;
}

/**
 * Get bar width percentage based on max value.
 */
function getBarWidth(count: number, periods: ActivityPeriod[]): number {
  const max = Math.max(...periods.map(p => p.count));
  return max > 0 ? (count / max) * 100 : 0;
}

/**
 * Format hour for display.
 */
function formatHour(hour: number): string {
  if (hour === 0) return '12am';
  if (hour === 12) return '12pm';
  return hour < 12 ? `${hour}am` : `${hour - 12}pm`;
}

/**
 * Get color class for hour based on time period.
 */
function getHourColor(hour: number): string {
  if (hour >= 0 && hour < 5) return 'bg-gray-400';      // Night
  if (hour >= 5 && hour < 8) return 'bg-orange-400';    // Dawn (prime bird time!)
  if (hour >= 8 && hour < 12) return 'bg-primary-500';  // Morning
  if (hour >= 12 && hour < 17) return 'bg-primary-400'; // Afternoon
  return 'bg-purple-400';                               // Evening
}

/**
 * Format date for display.
 */
function formatDate(dateStr: string): string {
  // Handle ISO date strings
  const date = new Date(dateStr);
  return date.toLocaleDateString('en-US', {
    weekday: 'short',
    month: 'short',
    day: 'numeric'
  });
}

// =============================================================================
// New Species Alert Component
// =============================================================================

interface NewSpeciesAlertProps {
  newSpecies: NewSpecies[];
  onSpeciesClick: (name: string) => void;
}

/**
 * Alert banner for species detected for the first time ever today.
 * This is the most exciting event for bird watchers!
 */
function NewSpeciesAlert({ newSpecies, onSpeciesClick }: NewSpeciesAlertProps): JSX.Element {
  if (newSpecies.length === 1) {
    const sp = newSpecies[0];
    return (
      <div
        class="card bg-gradient-to-r from-green-500 to-emerald-500 text-white p-4 cursor-pointer hover:from-green-600 hover:to-emerald-600 transition-colors"
        onClick={() => onSpeciesClick(sp.com_name)}
      >
        <div class="flex items-center gap-3">
          <div class="flex-shrink-0">
            <StarIcon class="w-8 h-8" />
          </div>
          <div class="flex-1">
            <p class="text-sm font-medium opacity-90">NEW SPECIES TODAY</p>
            <p class="text-xl font-bold">{sp.com_name}</p>
            <p class="text-sm opacity-90">
              First detected at {formatTime(sp.first_time)} ({Math.round(sp.max_confidence * 100)}% confidence)
            </p>
          </div>
          <div class="text-right">
            <p class="text-2xl font-bold">{sp.detection_count}</p>
            <p class="text-xs opacity-75">detection{sp.detection_count !== 1 ? 's' : ''}</p>
          </div>
        </div>
      </div>
    );
  }

  // Multiple new species
  return (
    <div class="card bg-gradient-to-r from-green-500 to-emerald-500 text-white p-4">
      <div class="flex items-start gap-3 mb-3">
        <StarIcon class="w-6 h-6 flex-shrink-0" />
        <div>
          <p class="font-bold text-lg">NEW SPECIES TODAY</p>
          <p class="text-sm opacity-90">{newSpecies.length} species detected for the first time ever!</p>
        </div>
      </div>
      <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-2">
        {newSpecies.map((sp) => (
          <button
            key={sp.sci_name}
            class="text-left bg-white/10 hover:bg-white/20 rounded-lg p-2 transition-colors"
            onClick={() => onSpeciesClick(sp.com_name)}
          >
            <p class="font-medium truncate">{sp.com_name}</p>
            <p class="text-xs opacity-75">
              {formatTime(sp.first_time)} - {Math.round(sp.max_confidence * 100)}%
            </p>
          </button>
        ))}
      </div>
    </div>
  );
}

/**
 * Format time string for display.
 */
function formatTime(timeStr: string): string {
  // timeStr is in format "HH:MM:SS"
  const parts = timeStr.split(':');
  if (parts.length >= 2) {
    const hour = parseInt(parts[0], 10);
    const minute = parts[1];
    const ampm = hour >= 12 ? 'PM' : 'AM';
    const hour12 = hour === 0 ? 12 : hour > 12 ? hour - 12 : hour;
    return `${hour12}:${minute} ${ampm}`;
  }
  return timeStr;
}

/**
 * Star icon for new species alert.
 */
function StarIcon({ class: className }: { class?: string }): JSX.Element {
  return (
    <svg class={className} viewBox="0 0 24 24" fill="currentColor">
      <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z" />
    </svg>
  );
}
