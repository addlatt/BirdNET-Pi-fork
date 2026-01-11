"""BirdNET-Pi ML Service - FastAPI Application.

This service provides:
- BirdNET bird detection inference
- Real-time detection notifications to Go backend
- Extensible architecture for Part 2 features (VAD, LLM)

Run with:
    uvicorn service.main:app --host 127.0.0.1 --port 8001

Or for development:
    uvicorn service.main:app --host 127.0.0.1 --port 8001 --reload
"""

import logging
import sys
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from .models.birdnet import birdnet_manager
from .routers import analysis_router, status_router, vad_router, llm_router
from .routers.status import set_start_time

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format="[%(name)s][%(levelname)s] %(message)s",
    stream=sys.stdout,
)
log = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Manage application lifecycle - startup and shutdown.

    Startup:
        - Load BirdNET model
        - Set service start time

    Shutdown:
        - Unload models
        - Close connections
    """
    # Startup
    log.info("Starting BirdNET-Pi ML Service...")
    set_start_time()

    # Pre-load BirdNET model
    log.info("Loading BirdNET model...")
    try:
        birdnet_manager.load()
        log.info("BirdNET model loaded successfully")
    except Exception as e:
        log.error("Failed to load BirdNET model: %s", e)
        # Don't fail startup - model can be loaded on first request

    yield

    # Shutdown
    log.info("Shutting down BirdNET-Pi ML Service...")

    # Unload models
    if birdnet_manager.is_loaded():
        log.info("Unloading BirdNET model...")
        birdnet_manager.unload()

    log.info("Shutdown complete")


# Create FastAPI application
app = FastAPI(
    title="BirdNET-Pi ML Service",
    description="Machine learning service for BirdNET-Pi bird detection",
    version="1.0.0",
    lifespan=lifespan,
    docs_url="/docs",
    redoc_url="/redoc",
)

# Configure CORS for development
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # Restrict in production
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Mount routers
app.include_router(
    status_router,
    prefix="/status",
    tags=["Status"],
)

app.include_router(
    analysis_router,
    prefix="/analysis",
    tags=["Analysis"],
)

app.include_router(
    vad_router,
    prefix="/vad",
    tags=["VAD (Part 2)"],
)

app.include_router(
    llm_router,
    prefix="/llm",
    tags=["LLM (Part 2)"],
)


@app.get("/")
async def root():
    """Root endpoint with service information."""
    return {
        "service": "BirdNET-Pi ML Service",
        "version": "1.0.0",
        "docs": "/docs",
        "status": "/status/health",
    }


@app.get("/health")
async def health():
    """Root-level health check for load balancers."""
    return {"status": "ok"}


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(
        "service.main:app",
        host="127.0.0.1",
        port=8001,
        reload=True,
    )
