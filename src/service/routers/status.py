"""Status router for health checks and system status.

Provides endpoints for monitoring the ML service health, status,
and memory usage.
"""

import os
import platform
import time
from typing import Optional

from fastapi import APIRouter

from ..models.birdnet import birdnet_manager
from ..notifier import get_notifier

router = APIRouter()

# Service start time for uptime calculation
_start_time: Optional[float] = None


def set_start_time() -> None:
    """Set the service start time. Called during startup."""
    global _start_time
    _start_time = time.time()


def get_uptime() -> Optional[float]:
    """Get service uptime in seconds."""
    if _start_time is None:
        return None
    return time.time() - _start_time


@router.get("/health")
async def health():
    """Health check endpoint.

    Returns:
        Simple status response for load balancer health checks.
    """
    return {"status": "ok"}


@router.get("/status")
async def status():
    """Get comprehensive service status.

    Returns:
        Status of all models and service components.
    """
    # Check Go server connectivity
    go_server_healthy = get_notifier().health_check()

    return {
        "service": {
            "status": "ok",
            "uptime_seconds": get_uptime(),
            "python_version": platform.python_version(),
            "platform": platform.platform(),
        },
        "birdnet": birdnet_manager.get_status(),
        "vad": {
            "enabled": False,
            "status": "not_implemented",
        },
        "llm": {
            "enabled": False,
            "loaded": False,
            "status": "not_implemented",
        },
        "go_server": {
            "url": os.environ.get("GO_SERVER_URL", "http://127.0.0.1:8080"),
            "healthy": go_server_healthy,
        },
    }


@router.get("/memory")
async def memory():
    """Get memory usage by component.

    Returns:
        Memory usage in bytes for each model/component.
    """
    birdnet_mem = birdnet_manager.memory_usage()

    # Part 2 placeholders
    vad_mem = 0
    llm_mem = 0

    return {
        "birdnet": birdnet_mem,
        "vad": vad_mem,
        "llm": llm_mem,
        "total": birdnet_mem + vad_mem + llm_mem,
        "breakdown": {
            "birdnet": {
                "bytes": birdnet_mem,
                "mb": round(birdnet_mem / (1024 * 1024), 2),
                "loaded": birdnet_manager.is_loaded(),
            },
            "vad": {
                "bytes": vad_mem,
                "mb": 0,
                "loaded": False,
            },
            "llm": {
                "bytes": llm_mem,
                "mb": 0,
                "loaded": False,
            },
        },
    }


@router.get("/models")
async def models():
    """Get status of all models.

    Returns:
        Detailed status for each model manager.
    """
    return {
        "birdnet": birdnet_manager.get_status(),
        "vad": {
            "name": "Silero VAD",
            "loaded": False,
            "memory_bytes": 0,
            "status": "not_implemented",
        },
        "llm": {
            "name": "TinyLlama",
            "loaded": False,
            "memory_bytes": 0,
            "status": "not_implemented",
        },
    }
