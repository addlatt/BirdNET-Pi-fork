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
WHERE sci_name = ? OR com_name = ?
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

-- =============================================================================
-- Phase 4.4: Recordings Browser Queries
-- =============================================================================

-- name: ListSpeciesWithStats :many
-- List all species with stats (sorted alphabetically)
SELECT sci_name, com_name, COUNT(*) as detection_count, MAX(confidence) as max_confidence, MAX(date) as last_seen
FROM detections
GROUP BY sci_name, com_name
ORDER BY com_name ASC;

-- name: ListSpeciesWithStatsByOccurrences :many
-- List all species with stats (sorted by occurrences)
SELECT sci_name, com_name, COUNT(*) as detection_count, MAX(confidence) as max_confidence, MAX(date) as last_seen
FROM detections
GROUP BY sci_name, com_name
ORDER BY detection_count DESC;

-- name: ListSpeciesWithStatsByConfidence :many
-- List all species with stats (sorted by confidence)
SELECT sci_name, com_name, COUNT(*) as detection_count, MAX(confidence) as max_confidence, MAX(date) as last_seen
FROM detections
GROUP BY sci_name, com_name
ORDER BY max_confidence DESC;

-- name: ListSpeciesWithStatsByDate2 :many
-- List all species with stats (sorted by last seen date)
SELECT sci_name, com_name, COUNT(*) as detection_count, MAX(confidence) as max_confidence, MAX(date) as last_seen
FROM detections
GROUP BY sci_name, com_name
ORDER BY last_seen DESC;

-- name: ListSpeciesWithStatsByDate :many
-- List species with stats for a specific date (sorted by occurrences)
SELECT sci_name, com_name, COUNT(*) as detection_count, MAX(confidence) as max_confidence, date as last_seen
FROM detections
WHERE date = ?
GROUP BY sci_name, com_name
ORDER BY detection_count DESC;

-- name: ListDetectionsBySpeciesAndDate :many
-- List detections for a species on a specific date
SELECT date, time, sci_name, com_name, confidence, lat, lon, cutoff, week, sens, overlap, file_name
FROM detections
WHERE (sci_name = ? OR com_name = ?) AND date = ?
ORDER BY time DESC
LIMIT ? OFFSET ?;

-- name: DeleteDetectionByFileName :exec
-- Delete a detection by filename
DELETE FROM detections
WHERE file_name = ?;

-- =============================================================================
-- Species Ranking Query (for Home Page)
-- =============================================================================

-- name: GetSpeciesRanking :many
-- Get unique species ranked by detection frequency, with latest and best detection info
WITH latest AS (
    SELECT sci_name, com_name, date, time, file_name, confidence,
           ROW_NUMBER() OVER (PARTITION BY sci_name ORDER BY date DESC, time DESC) as rn
    FROM detections
),
best AS (
    SELECT sci_name, com_name, date, time, file_name, confidence,
           ROW_NUMBER() OVER (PARTITION BY sci_name ORDER BY confidence DESC, date DESC, time DESC) as rn
    FROM detections
)
SELECT
    l.sci_name,
    l.com_name,
    (SELECT COUNT(*) FROM detections WHERE sci_name = l.sci_name) as detection_count,
    l.date as latest_date,
    l.time as latest_time,
    l.file_name as latest_file,
    l.confidence as latest_confidence,
    b.date as best_date,
    b.time as best_time,
    b.file_name as best_file,
    b.confidence as best_confidence
FROM latest l
JOIN best b ON l.sci_name = b.sci_name AND b.rn = 1
WHERE l.rn = 1
ORDER BY detection_count DESC;

-- name: GetSpeciesRankingByDateRange :many
-- Get unique species ranked by detection frequency within a date range
WITH filtered AS (
    SELECT * FROM detections WHERE date >= ? AND date <= ?
),
latest AS (
    SELECT sci_name, com_name, date, time, file_name, confidence,
           ROW_NUMBER() OVER (PARTITION BY sci_name ORDER BY date DESC, time DESC) as rn
    FROM filtered
),
best AS (
    SELECT sci_name, com_name, date, time, file_name, confidence,
           ROW_NUMBER() OVER (PARTITION BY sci_name ORDER BY confidence DESC, date DESC, time DESC) as rn
    FROM filtered
)
SELECT
    l.sci_name,
    l.com_name,
    (SELECT COUNT(*) FROM filtered WHERE sci_name = l.sci_name) as detection_count,
    l.date as latest_date,
    l.time as latest_time,
    l.file_name as latest_file,
    l.confidence as latest_confidence,
    b.date as best_date,
    b.time as best_time,
    b.file_name as best_file,
    b.confidence as best_confidence
FROM latest l
JOIN best b ON l.sci_name = b.sci_name AND b.rn = 1
WHERE l.rn = 1
ORDER BY detection_count DESC;
