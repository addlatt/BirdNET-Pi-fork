-- Initial schema for BirdNET-Pi detections database
-- This migration creates the core detections table

CREATE TABLE IF NOT EXISTS detections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date DATE NOT NULL,
    time TIME NOT NULL,
    sci_name VARCHAR(100) NOT NULL,
    com_name VARCHAR(100) NOT NULL,
    confidence REAL NOT NULL,
    lat REAL,
    lon REAL,
    cutoff REAL,
    week INTEGER,
    sens REAL,
    overlap REAL,
    file_name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_detections_date_time ON detections(date DESC, time DESC);
CREATE INDEX IF NOT EXISTS idx_detections_sci_name ON detections(sci_name);
CREATE INDEX IF NOT EXISTS idx_detections_com_name ON detections(com_name);
CREATE INDEX IF NOT EXISTS idx_detections_confidence ON detections(confidence);
