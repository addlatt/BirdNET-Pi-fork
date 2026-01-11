-- Initial schema for BirdNET-Pi detections database
-- This migration matches the existing Python-created schema

CREATE TABLE IF NOT EXISTS detections (
    Date DATE NOT NULL,
    Time TIME NOT NULL,
    Sci_Name VARCHAR(100) NOT NULL,
    Com_Name VARCHAR(100) NOT NULL,
    Confidence REAL,
    Lat REAL,
    Lon REAL,
    Cutoff REAL,
    Week INTEGER,
    Sens REAL,
    Overlap REAL,
    File_Name VARCHAR(100) NOT NULL
);

-- Indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_detections_date_time ON detections(Date DESC, Time DESC);
CREATE INDEX IF NOT EXISTS idx_detections_sci_name ON detections(Sci_Name);
CREATE INDEX IF NOT EXISTS idx_detections_com_name ON detections(Com_Name);
CREATE INDEX IF NOT EXISTS idx_detections_confidence ON detections(Confidence);
