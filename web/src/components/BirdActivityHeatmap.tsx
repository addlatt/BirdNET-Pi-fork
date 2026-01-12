import { useEffect, useRef, useState } from 'preact/hooks';
import type { JSX } from 'preact';
import type { HeatmapResponse } from '../types/api';
import {
  SciChartSurface,
  NumericAxis,
  HeatmapColorMap,
  UniformHeatmapDataSeries,
  UniformHeatmapRenderableSeries,
  EAxisAlignment,
  NumberRange,
  TextAnnotation,
  ECoordinateMode,
  EHorizontalAnchorPoint,
  EVerticalAnchorPoint,
} from 'scichart';

// Set the SciChart community license (free with watermark)
SciChartSurface.setRuntimeLicenseKey('');

// Configure SciChart to load WASM from CDN (avoids local file serving issues)
SciChartSurface.configure({
  wasmUrl: 'https://cdn.jsdelivr.net/npm/scichart@4.0.923/_wasm/scichart2d.wasm',
  dataUrl: 'https://cdn.jsdelivr.net/npm/scichart@4.0.923/_wasm/scichart2d.data',
});

interface BirdActivityHeatmapProps {
  data: HeatmapResponse | null;
  loading?: boolean;
  onSpeciesClick?: (species: string) => void;
}

/**
 * Bird Activity Heatmap - Shows detection counts per species per hour for today.
 * Uses SciChart Community Edition.
 */
export function BirdActivityHeatmap({
  data,
  loading = false,
  onSpeciesClick,
}: BirdActivityHeatmapProps): JSX.Element {
  const chartRef = useRef<HTMLDivElement>(null);
  const sciChartSurfaceRef = useRef<SciChartSurface | null>(null);
  const [initError, setInitError] = useState<string | null>(null);

  // Initialize SciChart
  useEffect(() => {
    if (!chartRef.current || !data || data.species.length === 0) return;

    let isSubscribed = true;

    const initChart = async () => {
      try {
        // Clean up previous chart if exists
        if (sciChartSurfaceRef.current) {
          sciChartSurfaceRef.current.delete();
          sciChartSurfaceRef.current = null;
        }

        // Create the chart surface
        const { sciChartSurface, wasmContext } = await SciChartSurface.create(
          chartRef.current!,
          {
            theme: {
              axisBandsFill: 'transparent',
              gridBackgroundBrush: 'transparent',
              gridBorderBrush: 'transparent',
              loadingAnimationBackground: 'transparent',
              sciChartBackground: 'transparent',
            },
          }
        );

        if (!isSubscribed) {
          sciChartSurface.delete();
          return;
        }

        sciChartSurfaceRef.current = sciChartSurface;

        // Configure X-axis (Hours 0-23)
        const xAxis = new NumericAxis(wasmContext, {
          axisTitle: 'Hour of Day',
          axisTitleStyle: { fontSize: 12, color: '#666' },
          labelStyle: { fontSize: 10, color: '#666' },
          visibleRange: new NumberRange(-0.5, 23.5),
          labelProvider: {
            formatLabel: (dataValue: number) => {
              const hour = Math.round(dataValue);
              if (hour === 0) return '12a';
              if (hour === 12) return '12p';
              return hour < 12 ? `${hour}a` : `${hour - 12}p`;
            },
          },
        });
        sciChartSurface.xAxes.add(xAxis);

        // Configure Y-axis (Species)
        const yAxis = new NumericAxis(wasmContext, {
          axisAlignment: EAxisAlignment.Left,
          axisTitleStyle: { fontSize: 12, color: '#666' },
          labelStyle: { fontSize: 10, color: '#666' },
          visibleRange: new NumberRange(-0.5, data.species.length - 0.5),
          labelProvider: {
            formatLabel: (dataValue: number) => {
              const idx = Math.round(dataValue);
              if (idx >= 0 && idx < data.species.length) {
                const name = data.species[idx];
                return name.length > 15 ? name.substring(0, 15) + '...' : name;
              }
              return '';
            },
          },
        });
        sciChartSurface.yAxes.add(yAxis);

        // Create heatmap data series
        const zValues = data.data;
        const maxValue = Math.max(...zValues.flat(), 1);

        const heatmapDataSeries = new UniformHeatmapDataSeries(wasmContext, {
          xStart: 0,
          xStep: 1,
          yStart: 0,
          yStep: 1,
          zValues,
        });

        // Create color map: white -> light green -> dark green
        const colorMap = new HeatmapColorMap({
          minimum: 0,
          maximum: maxValue,
          gradientStops: [
            { offset: 0, color: '#f0f9f0' },
            { offset: 0.2, color: '#90EE90' },
            { offset: 0.5, color: '#32CD32' },
            { offset: 1, color: '#006400' },
          ],
        });

        // Create heatmap series
        const heatmapSeries = new UniformHeatmapRenderableSeries(wasmContext, {
          dataSeries: heatmapDataSeries,
          colorMap,
          useLinearTextureFiltering: false,
          fillValuesOutOfRange: true,
        });

        sciChartSurface.renderableSeries.add(heatmapSeries);

        // Add text annotations for cell values
        for (let speciesIdx = 0; speciesIdx < data.species.length; speciesIdx++) {
          for (let hour = 0; hour < 24; hour++) {
            const count = data.data[speciesIdx]?.[hour] || 0;
            if (count > 0) {
              const annotation = new TextAnnotation({
                x1: hour,
                y1: speciesIdx,
                text: count.toString(),
                fontSize: 10,
                fontWeight: 'bold',
                textColor: count > maxValue * 0.5 ? '#ffffff' : '#333333',
                xCoordinateMode: ECoordinateMode.DataValue,
                yCoordinateMode: ECoordinateMode.DataValue,
                horizontalAnchorPoint: EHorizontalAnchorPoint.Center,
                verticalAnchorPoint: EVerticalAnchorPoint.Center,
              });
              sciChartSurface.annotations.add(annotation);
            }
          }
        }

        setInitError(null);
      } catch (err) {
        console.error('Failed to initialize SciChart:', err);
        setInitError(err instanceof Error ? err.message : 'Failed to load chart');
      }
    };

    initChart();

    return () => {
      isSubscribed = false;
      if (sciChartSurfaceRef.current) {
        sciChartSurfaceRef.current.delete();
        sciChartSurfaceRef.current = null;
      }
    };
  }, [data]);

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

  // Error state
  if (initError) {
    return (
      <div class="card">
        <div class="p-4 border-b border-gray-200 dark:border-gray-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Today's Bird Activity</h2>
        </div>
        <div class="p-4">
          <p class="text-red-600 dark:text-red-400 text-sm">Error loading chart: {initError}</p>
          {/* Fallback to simple table view */}
          <FallbackTable data={data} onSpeciesClick={onSpeciesClick} />
        </div>
      </div>
    );
  }

  // Chart height based on number of species
  const chartHeight = Math.max(200, data.species.length * 28 + 60);

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
      <div class="p-4">
        <div
          ref={chartRef}
          style={{ width: '100%', height: `${chartHeight}px` }}
        />
      </div>
    </div>
  );
}

/**
 * Fallback table view when SciChart fails to load.
 */
function FallbackTable({
  data,
  onSpeciesClick,
}: {
  data: HeatmapResponse;
  onSpeciesClick?: (species: string) => void;
}): JSX.Element {
  return (
    <div class="mt-4 overflow-x-auto">
      <table class="w-full text-xs">
        <thead>
          <tr>
            <th class="text-left p-1 sticky left-0 bg-white dark:bg-gray-800">Species</th>
            {data.hours.slice(0, 24).map((h) => (
              <th key={h} class="p-1 text-center w-6">
                {h}
              </th>
            ))}
            <th class="p-1 text-right">Total</th>
          </tr>
        </thead>
        <tbody>
          {data.species.map((species, idx) => {
            const rowTotal = data.data[idx]?.reduce((a, b) => a + b, 0) || 0;
            return (
              <tr
                key={species}
                class="hover:bg-gray-50 dark:hover:bg-gray-700 cursor-pointer"
                onClick={() => onSpeciesClick?.(species)}
              >
                <td class="p-1 truncate max-w-[120px] sticky left-0 bg-white dark:bg-gray-800">
                  {species}
                </td>
                {data.data[idx]?.map((count, hour) => (
                  <td
                    key={hour}
                    class="p-1 text-center"
                    style={{
                      backgroundColor: count > 0
                        ? `rgba(34, 139, 34, ${Math.min(count / 5, 1) * 0.7})`
                        : 'transparent',
                      color: count > 2 ? '#fff' : '#333',
                    }}
                  >
                    {count > 0 ? count : ''}
                  </td>
                ))}
                <td class="p-1 text-right font-medium">{rowTotal}</td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
