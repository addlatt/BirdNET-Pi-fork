-- name: GetDetectionByCompositeKey :one
SELECT date, time, sci_name, com_name, confidence, lat, lon, cutoff, week, sens, overlap, file_name
FROM detections
WHERE date = ? AND time = ? AND sci_name = ?
LIMIT 1;

-- name: ListDetections :many
SELECT date, time, sci_name, com_name, confidence, lat, lon, cutoff, week, sens, overlap, file_name
FROM detections
ORDER BY date DESC, time DESC
LIMIT ? OFFSET ?;

-- name: ListDetectionsByDate :many
SELECT date, time, sci_name, com_name, confidence, lat, lon, cutoff, week, sens, overlap, file_name
FROM detections
WHERE date = ?
ORDER BY time DESC;

-- name: ListDetectionsByDatePaginated :many
SELECT date, time, sci_name, com_name, confidence, lat, lon, cutoff, week, sens, overlap, file_name
FROM detections
WHERE date = ?
ORDER BY time DESC
LIMIT ? OFFSET ?;

-- name: ListDetectionsByDateRange :many
SELECT date, time, sci_name, com_name, confidence, lat, lon, cutoff, week, sens, overlap, file_name
FROM detections
WHERE date >= ? AND date <= ?
ORDER BY date DESC, time DESC
LIMIT ? OFFSET ?;

-- name: ListDetectionsBySpecies :many
SELECT date, time, sci_name, com_name, confidence, lat, lon, cutoff, week, sens, overlap, file_name
FROM detections
WHERE sci_name = ? OR com_name = ?
ORDER BY date DESC, time DESC
LIMIT ? OFFSET ?;

-- name: CountDetections :one
SELECT COUNT(*) as count
FROM detections;

-- name: CountDetectionsByDate :one
SELECT COUNT(*) as count
FROM detections
WHERE date = ?;

-- name: CountDetectionsToday :one
SELECT COUNT(*) as count
FROM detections
WHERE date = DATE('now', 'localtime');

-- name: ListSpecies :many
SELECT DISTINCT sci_name, com_name, COUNT(*) as detection_count, MAX(confidence) as max_confidence
FROM detections
GROUP BY sci_name, com_name
ORDER BY detection_count DESC;

-- name: ListSpeciesToday :many
SELECT DISTINCT sci_name, com_name, COUNT(*) as detection_count, MAX(confidence) as max_confidence
FROM detections
WHERE date = DATE('now', 'localtime')
GROUP BY sci_name, com_name
ORDER BY detection_count DESC;

-- name: GetSpeciesStats :one
SELECT
    sci_name,
    com_name,
    COUNT(*) as total_detections,
    MAX(confidence) as max_confidence,
    AVG(confidence) as avg_confidence,
    MIN(date) as first_detection,
    MAX(date) as last_detection
FROM detections
WHERE sci_name = ?
GROUP BY sci_name, com_name;

-- name: GetDailyStats :many
SELECT
    date,
    COUNT(*) as detection_count,
    COUNT(DISTINCT sci_name) as species_count,
    AVG(confidence) as avg_confidence
FROM detections
WHERE date >= ?
GROUP BY date
ORDER BY date DESC;

-- name: GetHourlyDistribution :many
SELECT
    CAST(strftime('%H', time) AS INTEGER) as hour,
    COUNT(*) as detection_count
FROM detections
WHERE date >= ?
GROUP BY hour
ORDER BY hour;

-- name: GetTopSpecies :many
SELECT sci_name, com_name, COUNT(*) as detection_count
FROM detections
WHERE date >= ?
GROUP BY sci_name, com_name
ORDER BY detection_count DESC
LIMIT ?;

-- name: GetRecentDetections :many
SELECT date, time, sci_name, com_name, confidence, lat, lon, cutoff, week, sens, overlap, file_name
FROM detections
ORDER BY date DESC, time DESC
LIMIT ?;

-- name: GetTotalSpeciesCount :one
SELECT COUNT(DISTINCT sci_name) as count
FROM detections;

-- name: GetTotalSpeciesCountToday :one
SELECT COUNT(DISTINCT sci_name) as count
FROM detections
WHERE date = DATE('now', 'localtime');

-- name: GetDetectionDates :many
SELECT DISTINCT date
FROM detections
ORDER BY date DESC
LIMIT ? OFFSET ?;

-- name: GetBestDetectionForSpecies :one
SELECT date, time, sci_name, com_name, confidence, file_name
FROM detections
WHERE sci_name = ? OR com_name = ?
ORDER BY confidence DESC
LIMIT 1;

-- name: GetSpeciesWithBestDetection :one
SELECT
    d.sci_name,
    d.com_name,
    COUNT(*) as detection_count,
    MAX(d.confidence) as max_confidence,
    (SELECT date FROM detections WHERE (sci_name = d.sci_name OR com_name = d.com_name) ORDER BY confidence DESC LIMIT 1) as best_date,
    (SELECT time FROM detections WHERE (sci_name = d.sci_name OR com_name = d.com_name) ORDER BY confidence DESC LIMIT 1) as best_time,
    (SELECT file_name FROM detections WHERE (sci_name = d.sci_name OR com_name = d.com_name) ORDER BY confidence DESC LIMIT 1) as best_file_name
FROM detections d
WHERE d.sci_name = ?
GROUP BY d.sci_name, d.com_name;

-- name: ListSpeciesSortedByAlphabetical :many
SELECT DISTINCT sci_name, com_name, COUNT(*) as detection_count, MAX(confidence) as max_confidence
FROM detections
GROUP BY sci_name, com_name
ORDER BY com_name ASC;

-- name: ListSpeciesSortedByConfidence :many
SELECT DISTINCT sci_name, com_name, COUNT(*) as detection_count, MAX(confidence) as max_confidence
FROM detections
GROUP BY sci_name, com_name
ORDER BY max_confidence DESC;

-- name: ListSpeciesSortedByDate :many
SELECT sci_name, com_name, COUNT(*) as detection_count, MAX(confidence) as max_confidence, MAX(date) as last_seen
FROM detections
GROUP BY sci_name, com_name
ORDER BY last_seen DESC;

-- =============================================================================
-- Phase 3.2: Today's Detections - Search, Filter, and Count Queries
-- =============================================================================

-- name: SearchDetectionsByDate :many
-- Text search within a specific date, with optional confidence filter
SELECT date, time, sci_name, com_name, confidence, lat, lon, cutoff, week, sens, overlap, file_name
FROM detections
WHERE date = ?
  AND (com_name LIKE ? OR sci_name LIKE ? OR file_name LIKE ? OR time LIKE ?)
  AND confidence >= ?
ORDER BY time DESC
LIMIT ? OFFSET ?;

-- name: CountSearchDetectionsByDate :one
-- Count for search within a specific date
SELECT COUNT(*) as count
FROM detections
WHERE date = ?
  AND (com_name LIKE ? OR sci_name LIKE ? OR file_name LIKE ? OR time LIKE ?)
  AND confidence >= ?;

-- name: SearchDetectionsExcludeByDate :many
-- Text search with NOT operator (exclude matches) within a specific date
SELECT date, time, sci_name, com_name, confidence, lat, lon, cutoff, week, sens, overlap, file_name
FROM detections
WHERE date = ?
  AND NOT (com_name LIKE ? OR sci_name LIKE ? OR file_name LIKE ? OR time LIKE ?)
  AND confidence >= ?
ORDER BY time DESC
LIMIT ? OFFSET ?;

-- name: CountSearchDetectionsExcludeByDate :one
-- Count for NOT search within a specific date
SELECT COUNT(*) as count
FROM detections
WHERE date = ?
  AND NOT (com_name LIKE ? OR sci_name LIKE ? OR file_name LIKE ? OR time LIKE ?)
  AND confidence >= ?;

-- name: ListDetectionsByDateWithConfidence :many
-- List detections for a date with minimum confidence filter
SELECT date, time, sci_name, com_name, confidence, lat, lon, cutoff, week, sens, overlap, file_name
FROM detections
WHERE date = ?
  AND confidence >= ?
ORDER BY time DESC
LIMIT ? OFFSET ?;

-- name: CountDetectionsByDateWithConfidence :one
-- Count detections for a date with minimum confidence filter
SELECT COUNT(*) as count
FROM detections
WHERE date = ?
  AND confidence >= ?;

-- name: CountDetectionsLastHour :one
-- Count detections from the last hour (for stats header)
SELECT COUNT(*) as count
FROM detections
WHERE date = DATE('now', 'localtime')
  AND time >= TIME('now', 'localtime', '-1 hour');

-- name: GetSpeciesDetectionHistory :many
-- Get daily detection counts for a species over a date range (for mini-chart)
SELECT date, COUNT(*) as detection_count
FROM detections
WHERE (com_name = ? OR sci_name = ?)
  AND date >= ?
GROUP BY date
ORDER BY date ASC;

-- name: DeleteDetectionByCompositeKey :exec
-- Delete a detection by its composite key
DELETE FROM detections
WHERE date = ? AND time = ? AND sci_name = ?;

-- =============================================================================
-- Phase 3.4: Species Management Queries
-- =============================================================================

-- name: DeleteAllDetectionsForSpecies :execresult
-- Delete ALL detections for a species (destructive!)
DELETE FROM detections
WHERE sci_name = ?;

-- name: GetSpeciesFilePaths :many
-- Get all file paths for a species (for file deletion)
SELECT date, com_name, file_name
FROM detections
WHERE sci_name = ?;

-- name: CountDetectionsBySpecies :one
-- Count detections for a specific species
SELECT COUNT(*) as count
FROM detections
WHERE sci_name = ?;

-- name: ListAllSpeciesWithLastSeen :many
-- Get all species with detection count, max confidence, and last seen date
SELECT sci_name, com_name, COUNT(*) as detection_count, MAX(confidence) as max_confidence, MAX(date) as last_seen
FROM detections
GROUP BY sci_name, com_name
ORDER BY detection_count DESC;

-- =============================================================================
-- New Species Today - Species detected today that have never been seen before
-- =============================================================================

-- name: GetNewSpeciesToday :many
-- Get species detected today that have never been detected before today
SELECT
    d.sci_name,
    d.com_name,
    MIN(d.time) as first_time,
    MAX(d.confidence) as max_confidence,
    COUNT(*) as detection_count
FROM detections d
WHERE d.date = DATE('now', 'localtime')
  AND d.sci_name NOT IN (
    SELECT DISTINCT sci_name
    FROM detections
    WHERE date < DATE('now', 'localtime')
  )
GROUP BY d.sci_name, d.com_name
ORDER BY first_time ASC;

-- =============================================================================
-- Species Hourly Heatmap - For bird activity visualization
-- =============================================================================

-- name: GetSpeciesHourlyDistributionToday :many
-- Gets detection counts per species per hour for today (for heatmap chart)
SELECT
    sci_name,
    com_name,
    CAST(strftime('%H', time) AS INTEGER) as hour,
    COUNT(*) as detection_count
FROM detections
WHERE date = DATE('now', 'localtime')
GROUP BY sci_name, com_name, hour
ORDER BY com_name, hour;
