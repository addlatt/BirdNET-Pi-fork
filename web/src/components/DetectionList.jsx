export function DetectionList({ detections }) {
  if (!detections || detections.length === 0) {
    return (
      <div class="p-8 text-center text-gray-500 dark:text-gray-400">
        No detections yet
      </div>
    );
  }

  return (
    <div class="divide-y divide-gray-200 dark:divide-gray-700">
      {detections.map((detection) => (
        <DetectionItem key={detection.id || `${detection.date}-${detection.time}`} detection={detection} />
      ))}
    </div>
  );
}

function DetectionItem({ detection }) {
  const confidencePercent = Math.round(detection.confidence * 100);
  const confidenceColor = getConfidenceColor(confidencePercent);

  return (
    <div class="p-4 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors">
      <div class="flex items-center justify-between">
        <div class="flex-1 min-w-0">
          <div class="flex items-center">
            <h3 class="font-medium text-gray-900 dark:text-white truncate">
              {detection.com_name}
            </h3>
            <span class="ml-2 text-sm text-gray-500 dark:text-gray-400 italic truncate">
              {detection.sci_name}
            </span>
          </div>
          <div class="flex items-center mt-1 text-sm text-gray-500 dark:text-gray-400">
            <span>{detection.date}</span>
            <span class="mx-2">-</span>
            <span>{detection.time}</span>
            {detection.file_name && (
              <>
                <span class="mx-2">-</span>
                <a
                  href={`/By_Date/${detection.date}/${detection.file_name}`}
                  class="text-primary-600 hover:underline truncate max-w-[150px]"
                  title={detection.file_name}
                >
                  Play
                </a>
              </>
            )}
          </div>
        </div>
        <div class="ml-4 flex-shrink-0">
          <div class="flex items-center">
            <div class="w-20 h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
              <div
                class={`h-full ${confidenceColor}`}
                style={{ width: `${confidencePercent}%` }}
              />
            </div>
            <span class="ml-2 text-sm font-medium text-gray-700 dark:text-gray-300 w-12 text-right">
              {confidencePercent}%
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}

function getConfidenceColor(percent) {
  if (percent >= 90) return 'bg-green-500';
  if (percent >= 70) return 'bg-green-400';
  if (percent >= 50) return 'bg-yellow-400';
  return 'bg-orange-400';
}
