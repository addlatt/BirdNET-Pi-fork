import type { JSX } from 'preact';
import type { HeatmapResponse } from '../types/api';

interface BirdActivityHeatmapProps {
  data: HeatmapResponse | null;
  loading?: boolean;
  onSpeciesClick?: (species: string) => void;
}

/**
 * Get color for heatmap cell based on count and max value.
 * Uses a green gradient from light to dark.
 */
function getCellColor(count: number, maxValue: number): string {
  if (count === 0) return 'transparent';
  const intensity = Math.min(count / maxValue, 1);
  // Gradient: light green (#dcfce7) -> medium green (#22c55e) -> dark green (#166534)
  if (intensity < 0.5) {
    // Light to medium
    const r = Math.round(220 - intensity * 2 * (220 - 34));
    const g = Math.round(252 - intensity * 2 * (252 - 197));
    const b = Math.round(231 - intensity * 2 * (231 - 94));
    return `rgb(${r}, ${g}, ${b})`;
  } else {
    // Medium to dark
    const t = (intensity - 0.5) * 2;
    const r = Math.round(34 - t * (34 - 22));
    const g = Math.round(197 - t * (197 - 101));
    const b = Math.round(94 - t * (94 - 52));
    return `rgb(${r}, ${g}, ${b})`;
  }
}

/**
 * Format hour for display (12-hour format with am/pm).
 */
function formatHour(hour: number): string {
  if (hour === 0) return '12a';
  if (hour === 12) return '12p';
  return hour < 12 ? `${hour}a` : `${hour - 12}p`;
}

/**
 * Bird Activity Heatmap - Shows detection counts per species per hour for today.
 * Simple CSS grid implementation for reliability.
 */
export function BirdActivityHeatmap({
  data,
  loading = false,
  onSpeciesClick,
}: BirdActivityHeatmapProps): JSX.Element {
  // Loading state
  if (loading) {
    return (
      <div class="card">
        <div class="p-4 border-b border-gray-200 dark:border-gray-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Today's Bird Activity</h2>
        </div>
        <div class="p-8 flex items-center justify-center">
          <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
        </div>
      </div>
    );
  }

  // Empty state
  if (!data || data.species.length === 0) {
    return (
      <div class="card">
        <div class="p-4 border-b border-gray-200 dark:border-gray-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Today's Bird Activity</h2>
        </div>
        <div class="p-8 text-center text-gray-500 dark:text-gray-400">
          <p>No detections today yet.</p>
          <p class="text-sm mt-1">The heatmap will appear when birds are detected.</p>
        </div>
      </div>
    );
  }

  // Calculate max value for color scaling
  const maxValue = Math.max(...data.data.flat(), 1);

  return (
    <div class="card">
      <div class="p-4 border-b border-gray-200 dark:border-gray-700">
        <div class="flex items-center justify-between">
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Today's Bird Activity</h2>
            <p class="text-sm text-gray-500 dark:text-gray-400">
              {data.total_detections} detection{data.total_detections !== 1 ? 's' : ''} across {data.species.length} species
            </p>
          </div>
        </div>
      </div>
      <div class="p-4 overflow-x-auto">
        <table class="w-full border-collapse text-xs">
          <thead>
            <tr>
              <th class="text-left p-1 pr-3 font-medium text-gray-600 dark:text-gray-400 sticky left-0 bg-white dark:bg-gray-800 z-10">
                Species
              </th>
              {Array.from({ length: 24 }, (_, hour) => (
                <th
                  key={hour}
                  class="p-1 text-center font-medium text-gray-500 dark:text-gray-400 min-w-[28px]"
                >
                  {formatHour(hour)}
                </th>
              ))}
              <th class="p-1 pl-3 text-right font-medium text-gray-600 dark:text-gray-400">
                Total
              </th>
            </tr>
          </thead>
          <tbody>
            {data.species.map((species, speciesIdx) => {
              const rowData = data.data[speciesIdx] || [];
              const rowTotal = rowData.reduce((a, b) => a + b, 0);

              return (
                <tr
                  key={species}
                  class={onSpeciesClick ? 'cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700/50' : ''}
                  onClick={() => onSpeciesClick?.(species)}
                >
                  <td class="p-1 pr-3 text-gray-900 dark:text-white font-medium truncate max-w-[150px] sticky left-0 bg-white dark:bg-gray-800 z-10">
                    <span title={species}>
                      {species.length > 20 ? species.substring(0, 20) + '...' : species}
                    </span>
                  </td>
                  {Array.from({ length: 24 }, (_, hour) => {
                    const count = rowData[hour] || 0;
                    const bgColor = getCellColor(count, maxValue);
                    const textColor = count > maxValue * 0.5 ? '#fff' : '#374151';

                    return (
                      <td
                        key={hour}
                        class="p-0 text-center"
                      >
                        <div
                          class="w-full h-7 flex items-center justify-center text-xs font-medium rounded-sm mx-px"
                          style={{
                            backgroundColor: bgColor,
                            color: count > 0 ? textColor : 'transparent',
                          }}
                        >
                          {count > 0 ? count : ''}
                        </div>
                      </td>
                    );
                  })}
                  <td class="p-1 pl-3 text-right font-semibold text-primary-600 dark:text-primary-400">
                    {rowTotal}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>

        {/* Legend */}
        <div class="mt-4 flex items-center justify-end gap-2 text-xs text-gray-500 dark:text-gray-400">
          <span>Less</span>
          <div class="flex gap-0.5">
            {[0, 0.25, 0.5, 0.75, 1].map((intensity) => (
              <div
                key={intensity}
                class="w-4 h-4 rounded-sm"
                style={{
                  backgroundColor: intensity === 0 ? '#f3f4f6' : getCellColor(intensity * maxValue, maxValue),
                }}
              />
            ))}
          </div>
          <span>More</span>
        </div>
      </div>
    </div>
  );
}
