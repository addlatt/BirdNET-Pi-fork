import type { JSX } from 'preact';
import { useState, useEffect } from 'preact/hooks';
import type { HeatmapResponse } from '../types/api';

interface BirdActivityHeatmapProps {
  data: HeatmapResponse | null;
  loading?: boolean;
  onSpeciesClick?: (species: string) => void;
}

/**
 * Get color for heatmap cell based on count and max value.
 * Uses a green gradient with better contrast for low values.
 * Applies log scale to make low counts (1-3) more visible.
 */
function getCellColor(count: number, maxValue: number): string {
  if (count === 0) return 'transparent';

  // Use log scale for better distribution of colors
  // This makes count=1 much more visible against the background
  const logCount = Math.log(count + 1);
  const logMax = Math.log(maxValue + 1);
  const intensity = Math.min(logCount / logMax, 1);

  // Start with a more saturated base green for count=1 visibility
  // Base: rgb(74, 222, 128) - a medium-light green that's visible
  // Mid: rgb(34, 197, 94) - green-500
  // Dark: rgb(22, 101, 52) - green-800

  if (intensity < 0.5) {
    // Base green to medium green
    const t = intensity * 2;
    const r = Math.round(74 - t * (74 - 34));
    const g = Math.round(222 - t * (222 - 197));
    const b = Math.round(128 - t * (128 - 94));
    return `rgb(${r}, ${g}, ${b})`;
  } else {
    // Medium green to dark green
    const t = (intensity - 0.5) * 2;
    const r = Math.round(34 - t * (34 - 22));
    const g = Math.round(197 - t * (197 - 101));
    const b = Math.round(94 - t * (94 - 52));
    return `rgb(${r}, ${g}, ${b})`;
  }
}

/**
 * Determine if text should be white or dark based on background intensity.
 */
function shouldUseWhiteText(count: number, maxValue: number): boolean {
  if (count === 0) return false;
  const logCount = Math.log(count + 1);
  const logMax = Math.log(maxValue + 1);
  const intensity = logCount / logMax;
  // Use white text when background is dark enough (intensity > 0.35)
  return intensity > 0.35;
}

/**
 * Format hour for display.
 * Full format: "12a", "1a", etc.
 * Compact format for mobile: just the number.
 */
function formatHour(hour: number, compact: boolean = false): string {
  if (compact) {
    return hour === 0 ? '0' : hour.toString();
  }
  if (hour === 0) return '12a';
  if (hour === 12) return '12p';
  return hour < 12 ? `${hour}a` : `${hour - 12}p`;
}

/**
 * Hook to detect if we're on a mobile-sized screen.
 */
function useIsMobile(): boolean {
  const [isMobile, setIsMobile] = useState(false);

  useEffect(() => {
    const checkMobile = () => setIsMobile(window.innerWidth < 640);
    checkMobile();
    window.addEventListener('resize', checkMobile);
    return () => window.removeEventListener('resize', checkMobile);
  }, []);

  return isMobile;
}

/**
 * Bird Activity Heatmap - Shows detection counts per species per hour for today.
 * Species are sorted by total detections (most active at top).
 * Responsive design with sticky species column and compact mobile layout.
 */
export function BirdActivityHeatmap({
  data,
  loading = false,
  onSpeciesClick,
}: BirdActivityHeatmapProps): JSX.Element {
  const isMobile = useIsMobile();

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

  // Calculate row totals and create sorted indices (most detections first)
  const rowTotals = data.data.map((row) => row.reduce((a, b) => a + b, 0));
  const sortedIndices = rowTotals
    .map((_, idx) => idx)
    .sort((a, b) => rowTotals[b] - rowTotals[a]);

  // Mobile: truncate species names more aggressively
  const maxNameLength = isMobile ? 12 : 20;

  return (
    <div class="card">
      <div class="p-3 sm:p-4 border-b border-gray-200 dark:border-gray-700">
        <div class="flex items-center justify-between flex-wrap gap-2">
          <div>
            <h2 class="text-base sm:text-lg font-semibold text-gray-900 dark:text-white">Today's Bird Activity</h2>
            <p class="text-xs sm:text-sm text-gray-500 dark:text-gray-400">
              {data.total_detections} detection{data.total_detections !== 1 ? 's' : ''} across {data.species.length} species
            </p>
          </div>
          {/* Legend - positioned in header on mobile for visibility */}
          <div class="flex items-center gap-1.5 text-xs text-gray-500 dark:text-gray-400">
            <span class="hidden sm:inline">Less</span>
            <span class="sm:hidden">0</span>
            <div class="flex gap-0.5">
              {[0, 1, 2, 4, maxValue].map((count, idx) => (
                <div
                  key={idx}
                  class="w-3 h-3 sm:w-4 sm:h-4 rounded-sm border border-gray-300 dark:border-gray-600"
                  style={{
                    backgroundColor: count === 0 ? '#f3f4f6' : getCellColor(count, maxValue),
                  }}
                  title={count === 0 ? '0' : count.toString()}
                />
              ))}
            </div>
            <span class="hidden sm:inline">More</span>
            <span class="sm:hidden">{maxValue}</span>
          </div>
        </div>
      </div>

      {/* Scrollable table container */}
      <div class="relative">
        <div class="overflow-x-auto p-2 sm:p-4">
          <table class="w-full border-collapse text-xs">
            <thead>
              <tr>
                <th class="text-left p-1 pr-2 sm:pr-3 font-medium text-gray-600 dark:text-gray-400 sticky left-0 bg-white dark:bg-gray-800 z-10 min-w-[80px] sm:min-w-[120px]">
                  Species
                </th>
                {Array.from({ length: 24 }, (_, hour) => (
                  <th
                    key={hour}
                    class="p-0.5 sm:p-1 text-center font-medium text-gray-500 dark:text-gray-400"
                    style={{ minWidth: isMobile ? '20px' : '28px' }}
                  >
                    {formatHour(hour, isMobile)}
                  </th>
                ))}
                <th class="p-1 pl-2 sm:pl-3 text-right font-medium text-gray-600 dark:text-gray-400 sticky right-0 bg-white dark:bg-gray-800 z-10">
                  Total
                </th>
              </tr>
            </thead>
            <tbody>
              {sortedIndices.map((speciesIdx) => {
                const species = data.species[speciesIdx];
                const rowData = data.data[speciesIdx] || [];
                const rowTotal = rowTotals[speciesIdx];

                return (
                  <tr
                    key={species}
                    class={onSpeciesClick ? 'cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700/50' : ''}
                    onClick={() => onSpeciesClick?.(species)}
                  >
                    <td class="p-1 pr-2 sm:pr-3 text-gray-900 dark:text-white font-medium truncate sticky left-0 bg-white dark:bg-gray-800 z-10">
                      <span
                        title={species}
                        class="block truncate"
                        style={{ maxWidth: isMobile ? '80px' : '150px' }}
                      >
                        {species.length > maxNameLength ? species.substring(0, maxNameLength) + '…' : species}
                      </span>
                    </td>
                    {Array.from({ length: 24 }, (_, hour) => {
                      const count = rowData[hour] || 0;
                      const bgColor = getCellColor(count, maxValue);
                      const useWhite = shouldUseWhiteText(count, maxValue);

                      return (
                        <td
                          key={hour}
                          class="p-0 text-center"
                        >
                          <div
                            class="flex items-center justify-center text-xs font-medium rounded-sm"
                            style={{
                              height: isMobile ? '24px' : '28px',
                              minWidth: isMobile ? '18px' : '26px',
                              margin: '0 1px',
                              backgroundColor: bgColor,
                              color: count > 0 ? (useWhite ? '#fff' : '#1f2937') : 'transparent',
                            }}
                          >
                            {count > 0 ? count : ''}
                          </div>
                        </td>
                      );
                    })}
                    <td class="p-1 pl-2 sm:pl-3 text-right font-semibold text-primary-600 dark:text-primary-400 sticky right-0 bg-white dark:bg-gray-800 z-10">
                      {rowTotal}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
