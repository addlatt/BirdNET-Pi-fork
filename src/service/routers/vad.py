"""VAD (Voice Activity Detection) router - Part 2 stub.

This router provides placeholder endpoints for the Silero VAD feature
that will be implemented in Part 2. Currently returns 501 Not Implemented.
"""

from fastapi import APIRouter, HTTPException
from pydantic import BaseModel, Field
from typing import Optional

router = APIRouter()


class VADCheckRequest(BaseModel):
    """Request body for VAD check."""

    audio_path: str = Field(..., description="Path to audio file to check")
    threshold: Optional[float] = Field(
        0.5,
        ge=0.0,
        le=1.0,
        description="Speech detection threshold",
    )


class VADCheckResponse(BaseModel):
    """Response for VAD check."""

    audio_path: str
    has_speech: bool
    speech_probability: float
    segments: list[dict]


@router.post("/check", response_model=VADCheckResponse)
async def check_vad(request: VADCheckRequest):
    """Check audio file for voice/speech activity.

    Part 2 feature - not yet implemented.

    Args:
        request: VAD check request with audio path.

    Raises:
        HTTPException: 501 Not Implemented.
    """
    raise HTTPException(
        status_code=501,
        detail="VAD feature not implemented yet. Coming in Part 2.",
    )


@router.get("/status")
async def vad_status():
    """Get VAD model status.

    Returns:
        VAD feature status (currently disabled).
    """
    return {
        "enabled": False,
        "loaded": False,
        "model_name": "Silero VAD",
        "status": "not_implemented",
        "memory_bytes": 0,
        "description": "Voice Activity Detection - Coming in Part 2",
    }


@router.post("/load")
async def load_vad():
    """Load the VAD model.

    Part 2 feature - not yet implemented.

    Raises:
        HTTPException: 501 Not Implemented.
    """
    raise HTTPException(
        status_code=501,
        detail="VAD feature not implemented yet. Coming in Part 2.",
    )


@router.post("/unload")
async def unload_vad():
    """Unload the VAD model.

    Part 2 feature - not yet implemented.

    Raises:
        HTTPException: 501 Not Implemented.
    """
    raise HTTPException(
        status_code=501,
        detail="VAD feature not implemented yet. Coming in Part 2.",
    )
