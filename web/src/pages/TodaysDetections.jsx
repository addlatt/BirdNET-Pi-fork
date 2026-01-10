import { useState, useEffect } from 'preact/hooks';
import { useWebSocket } from '../hooks/useWebSocket';
import { fetchDetections } from '../hooks/useApi';
import { DetectionList } from '../components/DetectionList';

export function TodaysDetections() {
  const [detections, setDetections] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const perPage = 20;

  // WebSocket connection
  const wsUrl = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws`;
  const { isConnected, subscribe } = useWebSocket(wsUrl);

  // Load detections
  useEffect(() => {
    async function loadData() {
      try {
        setLoading(true);
        const today = new Date().toISOString().split('T')[0];
        const data = await fetchDetections({
          date: today,
          page,
          per_page: perPage,
        });
        setDetections(data.detections || []);
        setTotal(data.total || 0);
        setError(null);
      } catch (err) {
        setError(err.message);
      } finally {
        setLoading(false);
      }
    }
    loadData();
  }, [page]);

  // Subscribe to real-time detection updates
  useEffect(() => {
    const unsubscribe = subscribe('detection', (payload) => {
      // Add new detection to the top of the list
      setDetections((prev) => [payload, ...prev]);
      setTotal((prev) => prev + 1);
    });

    return unsubscribe;
  }, [subscribe]);

  const totalPages = Math.ceil(total / perPage);

  return (
    <div class="space-y-6">
      <div class="flex items-center justify-between">
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Today's Detections</h1>
        <span class={`flex items-center text-sm ${isConnected ? 'text-green-600' : 'text-red-600'}`}>
          <span class={`w-2 h-2 rounded-full mr-2 ${isConnected ? 'bg-green-600' : 'bg-red-600'}`}></span>
          {isConnected ? 'Live' : 'Offline'}
        </span>
      </div>

      <div class="card">
        {loading ? (
          <div class="flex items-center justify-center h-64">
            <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
          </div>
        ) : error ? (
          <div class="p-6 text-center">
            <p class="text-red-600 dark:text-red-400">Error: {error}</p>
            <button
              class="btn btn-primary mt-4"
              onClick={() => window.location.reload()}
            >
              Retry
            </button>
          </div>
        ) : (
          <>
            <div class="p-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
              <span class="text-sm text-gray-500 dark:text-gray-400">
                {total} detection{total !== 1 ? 's' : ''} today
              </span>
            </div>

            <DetectionList detections={detections} />

            {/* Pagination */}
            {totalPages > 1 && (
              <div class="p-4 border-t border-gray-200 dark:border-gray-700 flex items-center justify-between">
                <button
                  class="btn btn-secondary"
                  disabled={page === 1}
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                >
                  Previous
                </button>
                <span class="text-sm text-gray-500 dark:text-gray-400">
                  Page {page} of {totalPages}
                </span>
                <button
                  class="btn btn-secondary"
                  disabled={page >= totalPages}
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                >
                  Next
                </button>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}
