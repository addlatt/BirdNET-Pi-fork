-- name: GetDetection :one
SELECT id, date, time, sci_name, com_name, confidence, lat, lon, cutoff, week, sens, overlap, file_name, created_at
FROM detections
WHERE id = ?
LIMIT 1;

-- name: ListDetections :many
SELECT id, date, time, sci_name, com_name, confidence, lat, lon, cutoff, week, sens, overlap, file_name, created_at
FROM detections
ORDER BY date DESC, time DESC
LIMIT ? OFFSET ?;

-- name: ListDetectionsByDate :many
SELECT id, date, time, sci_name, com_name, confidence, lat, lon, cutoff, week, sens, overlap, file_name, created_at
FROM detections
WHERE date = ?
ORDER BY time DESC;

-- name: ListDetectionsByDateRange :many
SELECT id, date, time, sci_name, com_name, confidence, lat, lon, cutoff, week, sens, overlap, file_name, created_at
FROM detections
WHERE date >= ? AND date <= ?
ORDER BY date DESC, time DESC
LIMIT ? OFFSET ?;

-- name: ListDetectionsBySpecies :many
SELECT id, date, time, sci_name, com_name, confidence, lat, lon, cutoff, week, sens, overlap, file_name, created_at
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
WHERE date = DATE('now');

-- name: ListSpecies :many
SELECT DISTINCT sci_name, com_name, COUNT(*) as detection_count, MAX(confidence) as max_confidence
FROM detections
GROUP BY sci_name, com_name
ORDER BY detection_count DESC;

-- name: ListSpeciesToday :many
SELECT DISTINCT sci_name, com_name, COUNT(*) as detection_count, MAX(confidence) as max_confidence
FROM detections
WHERE date = DATE('now')
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
SELECT id, date, time, sci_name, com_name, confidence, lat, lon, cutoff, week, sens, overlap, file_name, created_at
FROM detections
ORDER BY date DESC, time DESC
LIMIT ?;

-- name: GetTotalSpeciesCount :one
SELECT COUNT(DISTINCT sci_name) as count
FROM detections;

-- name: GetTotalSpeciesCountToday :one
SELECT COUNT(DISTINCT sci_name) as count
FROM detections
WHERE date = DATE('now');

-- name: GetDetectionDates :many
SELECT DISTINCT date
FROM detections
ORDER BY date DESC
LIMIT ? OFFSET ?;
