import { useState, useEffect } from 'preact/hooks';
import { fetchStats, fetchSpecies } from '../hooks/useApi';
import { StatsCards } from '../components/StatsCards';

export function Stats() {
  const [stats, setStats] = useState(null);
  const [species, setSpecies] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [days, setDays] = useState(7);

  useEffect(() => {
    async function loadData() {
      try {
        setLoading(true);
        const [statsData, speciesData] = await Promise.all([
          fetchStats({
            days,
            include_daily: 'true',
            include_hourly: 'true',
            include_top_species: 'true',
            top_limit: 10,
          }),
          fetchSpecies(),
        ]);
        setStats(statsData);
        setSpecies(speciesData.species || []);
        setError(null);
      } catch (err) {
        setError(err.message);
      } finally {
        setLoading(false);
      }
    }
    loadData();
  }, [days]);

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
      <div class="flex items-center justify-between">
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Statistics</h1>
        <select
          class="input w-40"
          value={days}
          onChange={(e) => setDays(parseInt(e.target.value, 10))}
        >
          <option value={7}>Last 7 days</option>
          <option value={14}>Last 14 days</option>
          <option value={30}>Last 30 days</option>
          <option value={90}>Last 90 days</option>
        </select>
      </div>

      {/* Stats Cards */}
      {stats && <StatsCards stats={stats} />}

      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Daily Stats */}
        {stats?.daily_stats && stats.daily_stats.length > 0 && (
          <div class="card">
            <div class="p-4 border-b border-gray-200 dark:border-gray-700">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Daily Detections</h2>
            </div>
            <div class="p-4 overflow-x-auto">
              <table class="w-full text-sm">
                <thead>
                  <tr class="text-left text-gray-500 dark:text-gray-400">
                    <th class="pb-2">Date</th>
                    <th class="pb-2 text-right">Detections</th>
                    <th class="pb-2 text-right">Species</th>
                    <th class="pb-2 text-right">Avg Conf</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-gray-200 dark:divide-gray-700">
                  {stats.daily_stats.slice(0, 7).map((day) => (
                    <tr key={day.date} class="text-gray-900 dark:text-white">
                      <td class="py-2">{day.date}</td>
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

        {/* Hourly Distribution */}
        {stats?.hourly_distribution && stats.hourly_distribution.length > 0 && (
          <div class="card">
            <div class="p-4 border-b border-gray-200 dark:border-gray-700">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Hourly Distribution</h2>
            </div>
            <div class="p-4">
              <div class="flex items-end h-40 space-x-1">
                {Array.from({ length: 24 }, (_, hour) => {
                  const data = stats.hourly_distribution.find((h) => h.hour === hour);
                  const count = data?.detection_count || 0;
                  const maxCount = Math.max(...stats.hourly_distribution.map((h) => h.detection_count));
                  const height = maxCount > 0 ? (count / maxCount) * 100 : 0;

                  return (
                    <div
                      key={hour}
                      class="flex-1 flex flex-col items-center"
                      title={`${hour}:00 - ${count} detections`}
                    >
                      <div
                        class="w-full bg-primary-500 rounded-t"
                        style={{ height: `${height}%`, minHeight: count > 0 ? '4px' : '0' }}
                      />
                      {hour % 6 === 0 && (
                        <span class="text-xs text-gray-500 mt-1">{hour}</span>
                      )}
                    </div>
                  );
                })}
              </div>
              <p class="text-xs text-gray-500 dark:text-gray-400 mt-2 text-center">Hour of day</p>
            </div>
          </div>
        )}
      </div>

      {/* All Species */}
      <div class="card">
        <div class="p-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">All Species</h2>
          <span class="text-sm text-gray-500 dark:text-gray-400">{species.length} species</span>
        </div>
        <div class="p-4 overflow-x-auto">
          <table class="w-full text-sm">
            <thead>
              <tr class="text-left text-gray-500 dark:text-gray-400">
                <th class="pb-2">Common Name</th>
                <th class="pb-2">Scientific Name</th>
                <th class="pb-2 text-right">Detections</th>
                <th class="pb-2 text-right">Max Confidence</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 dark:divide-gray-700">
              {species.map((s) => (
                <tr key={s.sci_name} class="text-gray-900 dark:text-white">
                  <td class="py-2 font-medium">{s.com_name}</td>
                  <td class="py-2 italic text-gray-500 dark:text-gray-400">{s.sci_name}</td>
                  <td class="py-2 text-right">{s.detection_count}</td>
                  <td class="py-2 text-right">{Math.round(s.max_confidence * 100)}%</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
