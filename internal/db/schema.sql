-- BirdNET-Pi database schema
-- This schema matches the actual Pi database structure
-- Note: The Pi database uses capitalized column names but SQLite is case-insensitive

CREATE TABLE IF NOT EXISTS detections (
    date DATE NOT NULL,
    time TIME NOT NULL,
    sci_name VARCHAR(100) NOT NULL,
    com_name VARCHAR(100) NOT NULL,
    confidence REAL,
    lat REAL,
    lon REAL,
    cutoff REAL,
    week INTEGER,
    sens REAL,
    overlap REAL,
    file_name VARCHAR(255) NOT NULL
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_detections_date_time ON detections(date DESC, time DESC);
CREATE INDEX IF NOT EXISTS idx_detections_sci_name ON detections(sci_name);
CREATE INDEX IF NOT EXISTS idx_detections_com_name ON detections(com_name);
CREATE INDEX IF NOT EXISTS idx_detections_confidence ON detections(confidence);
