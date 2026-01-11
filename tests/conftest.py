"""Pytest configuration and shared fixtures for BirdNET-Pi tests.

This module provides lightweight fixtures that avoid loading heavy models
or creating large files on disk - important for Pi's limited storage.
"""

import os
import sys
from unittest.mock import MagicMock, patch

import pytest

# Ensure src is in path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "src"))


# ============================================================================
# Mock Settings
# ============================================================================

@pytest.fixture
def mock_settings():
    """Provide mock settings without loading from disk."""
    return {
        "OVERLAP": 0.0,
        "LATITUDE": 42.3601,
        "LONGITUDE": -71.0589,
        "CONFIDENCE": 0.7,
        "DATABASE_LANG": "en",
        "PRIVACY_THRESHOLD": 0,
        "EXTRACTION_LENGTH": 6,
        "MODEL": "BirdNET_GLOBAL_6K_V2.4_Model_FP16",
        "DATA_MODEL_VERSION": 1,
        "SENSITIVITY": 1.25,
        "SF_THRESH": 0.003,
        "RECS_DIR": "/tmp/test_recs",
        "EXTRACTED": "/tmp/test_extracted",
        "AUDIOFMT": "mp3",
        "RAW_SPECTROGRAM": 0,
        "RECORDING_LENGTH": 15,
        "BIRDWEATHER_ID": "",
        "HEARTBEAT_URL": "",
    }


# ============================================================================
# Mock BirdNET Model
# ============================================================================

@pytest.fixture
def mock_birdnet_model():
    """Create a mock BirdNET model that doesn't load TFLite."""
    model = MagicMock()
    model.sample_rate = 48000
    model.chunk_duration = 3
    model.predict.return_value = [
        ("Pica pica", 0.95),
        ("Corvus corax", 0.12),
        ("Turdus merula", 0.08),
    ]
    model.get_species_list.return_value = ["Pica pica", "Corvus corax"]
    return model


@pytest.fixture
def patch_birdnet_model(mock_birdnet_model):
    """Patch get_model to return mock instead of loading real model."""
    with patch("birdnet.models.get_model", return_value=mock_birdnet_model):
        yield mock_birdnet_model


# ============================================================================
# FastAPI Test Client
# ============================================================================

@pytest.fixture
def mock_birdnet_manager():
    """Create a mock BirdNETManager that doesn't load the real model."""
    manager = MagicMock()
    manager.name = "BirdNET"
    manager.is_loaded.return_value = True
    manager.memory_usage.return_value = 500 * 1024 * 1024  # 500MB
    manager.get_model_name.return_value = "BirdNET_GLOBAL_6K_V2.4_Model_FP16"
    manager.get_inference_count.return_value = 42
    manager.get_load_time.return_value = 1704067200.0  # 2024-01-01
    manager.get_uptime.return_value = 3600.0  # 1 hour
    manager.get_status.return_value = {
        "name": "BirdNET",
        "loaded": True,
        "memory_bytes": 500 * 1024 * 1024,
        "load_time": 1704067200.0,
        "uptime_seconds": 3600.0,
        "inference_count": 42,
        "model_name": "BirdNET_GLOBAL_6K_V2.4_Model_FP16",
    }
    return manager


@pytest.fixture
def test_client(mock_birdnet_manager):
    """Create FastAPI TestClient with mocked dependencies."""
    # Patch the birdnet_manager before importing the app
    with patch("service.models.birdnet.birdnet_manager", mock_birdnet_manager):
        with patch("service.routers.status.birdnet_manager", mock_birdnet_manager):
            with patch("service.routers.analysis.birdnet_manager", mock_birdnet_manager):
                # Import here so patches are applied
                from fastapi.testclient import TestClient
                from service.main import app

                # Don't run lifespan (which would load real model)
                client = TestClient(app, raise_server_exceptions=False)
                yield client


# ============================================================================
# Mock Detection Data
# ============================================================================

@pytest.fixture
def sample_detection():
    """Sample detection data matching the DB schema."""
    return {
        "date": "2024-01-15",
        "time": "14:30:45",
        "sci_name": "Pica pica",
        "com_name": "Eurasian Magpie",
        "confidence": 0.9234,
        "file_name": "Eurasian_Magpie-92-2024-01-15-birdnet-14:30:45.mp3",
        "lat": 42.3601,
        "lon": -71.0589,
    }


@pytest.fixture
def sample_detections():
    """Multiple sample detections for batch testing."""
    return [
        {
            "date": "2024-01-15",
            "time": "14:30:45",
            "sci_name": "Pica pica",
            "com_name": "Eurasian Magpie",
            "confidence": 0.9234,
        },
        {
            "date": "2024-01-15",
            "time": "14:31:12",
            "sci_name": "Corvus corax",
            "com_name": "Common Raven",
            "confidence": 0.8567,
        },
        {
            "date": "2024-01-15",
            "time": "14:32:00",
            "sci_name": "Turdus merula",
            "com_name": "Common Blackbird",
            "confidence": 0.7891,
        },
    ]


# ============================================================================
# Temporary Files (auto-cleaned by pytest)
# ============================================================================

@pytest.fixture
def temp_audio_path(tmp_path):
    """Create a fake audio file path (doesn't create actual audio)."""
    audio_file = tmp_path / "2024-01-15-birdnet-14:30:45.wav"
    # Create empty file to simulate existence
    audio_file.touch()
    return str(audio_file)
