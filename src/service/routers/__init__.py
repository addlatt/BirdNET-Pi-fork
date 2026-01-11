"""FastAPI routers for the BirdNET-Pi ML Service."""

from .analysis import router as analysis_router
from .status import router as status_router
from .vad import router as vad_router
from .llm import router as llm_router

__all__ = ["analysis_router", "status_router", "vad_router", "llm_router"]
