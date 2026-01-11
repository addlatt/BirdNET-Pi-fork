-- Rollback initial schema
DROP INDEX IF EXISTS idx_detections_confidence;
DROP INDEX IF EXISTS idx_detections_com_name;
DROP INDEX IF EXISTS idx_detections_sci_name;
DROP INDEX IF EXISTS idx_detections_date_time;
DROP TABLE IF EXISTS detections;
