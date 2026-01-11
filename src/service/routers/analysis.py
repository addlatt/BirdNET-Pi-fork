"""Analysis router for BirdNET inference endpoints.

Provides endpoints for analyzing audio files and managing the analysis queue.
"""

import logging
import os
from typing import Optional

from fastapi import APIRouter, HTTPException, BackgroundTasks
from pydantic import BaseModel, Field

from ..models.birdnet import birdnet_manager

router = APIRouter()
log = logging.getLogger(__name__)


class AnalyzeFileRequest(BaseModel):
    """Request body for file analysis."""

    file_path: str = Field(..., description="Path to audio file to analyze")
    lat: Optional[float] = Field(None, description="Latitude for species filtering")
    lon: Optional[float] = Field(None, description="Longitude for species filtering")
    week: Optional[int] = Field(None, ge=1, le=52, description="ISO week number")


class AnalyzeFileResponse(BaseModel):
    """Response for file analysis."""

    file_path: str
    detections: list[dict]
    duration_seconds: float
    model_name: Optional[str]


class DetectionResult(BaseModel):
    """Single detection result."""

    start: float
    end: float
    scientific_name: str
    common_name: str
    confidence: float


@router.post("/file", response_model=AnalyzeFileResponse)
async def analyze_file(request: AnalyzeFileRequest):
    """Analyze a single audio file for bird sounds.

    This endpoint runs BirdNET inference on the provided audio file
    and returns all detections above the confidence threshold.

    Args:
        request: Analysis request with file path and optional location.

    Returns:
        List of detections with species and confidence scores.

    Raises:
        HTTPException: If file not found or analysis fails.
    """
    import time

    # Validate file exists
    if not os.path.isfile(request.file_path):
        raise HTTPException(
            status_code=404,
            detail=f"File not found: {request.file_path}",
        )

    start_time = time.time()

    try:
        # Import analysis functions
        from birdnet.analysis import readAudioData, analyzeAudioData
        from birdnet.config import get_settings

        conf = get_settings()

        # Get model parameters
        model = birdnet_manager.get_model()
        overlap = float(conf.get("OVERLAP", 0))

        # Use provided location or fall back to config
        lat = request.lat if request.lat is not None else float(conf.get("LATITUDE", 0))
        lon = request.lon if request.lon is not None else float(conf.get("LONGITUDE", 0))
        week = request.week if request.week is not None else 1

        # Read and analyze audio
        audio_chunks = readAudioData(
            request.file_path,
            overlap,
            model.sample_rate,
            model.chunk_duration,
        )

        raw_detections, _ = analyzeAudioData(
            audio_chunks,
            overlap,
            lat,
            lon,
            week,
        )

        # Format detections
        detections = []
        confidence_threshold = float(conf.get("CONFIDENCE", 0.7))

        for time_slot, entries in raw_detections.items():
            start, end = time_slot.split(";")
            for sci_name, confidence in entries:
                if confidence >= confidence_threshold:
                    detections.append({
                        "start": float(start),
                        "end": float(end),
                        "scientific_name": sci_name,
                        "common_name": sci_name,  # Would need language lookup
                        "confidence": round(confidence, 4),
                    })

        duration = time.time() - start_time
        birdnet_manager.increment_inference_count()

        return AnalyzeFileResponse(
            file_path=request.file_path,
            detections=detections,
            duration_seconds=round(duration, 3),
            model_name=birdnet_manager.get_model_name(),
        )

    except Exception as e:
        log.exception("Analysis failed for %s", request.file_path)
        raise HTTPException(
            status_code=500,
            detail=f"Analysis failed: {str(e)}",
        )


@router.get("/queue")
async def get_queue_status():
    """Get the analysis queue status.

    Returns:
        Queue statistics and current processing state.
    """
    # The queue is managed by the pipeline module
    # For now, return basic status
    return {
        "queue_length": 0,
        "processing": False,
        "current_file": None,
        "files_processed_today": 0,
    }


@router.get("/model")
async def get_model_info():
    """Get information about the loaded BirdNET model.

    Returns:
        Model configuration and capabilities.
    """
    model = birdnet_manager.get_model()

    return {
        "name": birdnet_manager.get_model_name(),
        "loaded": birdnet_manager.is_loaded(),
        "sample_rate": model.sample_rate if model else None,
        "chunk_duration": model.chunk_duration if model else None,
        "inference_count": birdnet_manager.get_inference_count(),
        "memory_bytes": birdnet_manager.memory_usage(),
    }


@router.post("/model/reload")
async def reload_model():
    """Reload the BirdNET model.

    Useful after configuration changes or to free memory.

    Returns:
        New model status after reload.
    """
    log.info("Reloading BirdNET model...")
    birdnet_manager.unload()
    birdnet_manager.load()

    return {
        "status": "reloaded",
        "model": birdnet_manager.get_status(),
    }
