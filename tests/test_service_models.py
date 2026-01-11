"""Tests for service.models module.

Tests the ModelManager base class and BirdNETManager without loading
the actual TFLite model (~500MB saved).
"""

import sys
import time
from unittest.mock import MagicMock, patch

import pytest


# Create mock modules for birdnet.models to avoid TensorFlow import
@pytest.fixture(autouse=True)
def mock_birdnet_modules():
    """Mock birdnet.models and birdnet.config to avoid TensorFlow import."""
    mock_models = MagicMock()
    mock_config = MagicMock()

    # Store original modules if they exist
    orig_models = sys.modules.get("birdnet.models")
    orig_config = sys.modules.get("birdnet.config")

    # Insert mocks
    sys.modules["birdnet.models"] = mock_models
    sys.modules["birdnet.config"] = mock_config

    yield {"models": mock_models, "config": mock_config}

    # Restore original modules
    if orig_models is not None:
        sys.modules["birdnet.models"] = orig_models
    else:
        sys.modules.pop("birdnet.models", None)

    if orig_config is not None:
        sys.modules["birdnet.config"] = orig_config
    else:
        sys.modules.pop("birdnet.config", None)


class TestModelManagerBase:
    """Test the ModelManager abstract base class."""

    def test_model_manager_interface(self):
        """Test that ModelManager enforces the abstract interface."""
        from service.models.base import ModelManager

        # Can't instantiate abstract class
        with pytest.raises(TypeError):
            ModelManager("test")

    def test_concrete_implementation(self):
        """Test a concrete ModelManager implementation."""
        from service.models.base import ModelManager

        class TestManager(ModelManager):
            def _load_model(self):
                return {"loaded": True}

            def _unload_model(self):
                pass

            def memory_usage(self):
                return 100 * 1024 * 1024  # 100MB

        manager = TestManager("test-model")

        # Initially not loaded
        assert not manager.is_loaded()
        assert manager.memory_usage() == 100 * 1024 * 1024
        assert manager.get_load_time() is None
        assert manager.get_uptime() is None

        # Load model
        manager.load()
        assert manager.is_loaded()
        assert manager.get_load_time() is not None
        assert manager.get_uptime() >= 0

        # Get model (should return cached)
        model = manager.get_model()
        assert model == {"loaded": True}

        # Unload
        manager.unload()
        assert not manager.is_loaded()
        assert manager.get_load_time() is None

    def test_inference_counting(self):
        """Test inference counter."""
        from service.models.base import ModelManager

        class TestManager(ModelManager):
            def _load_model(self):
                return {}

            def _unload_model(self):
                pass

            def memory_usage(self):
                return 0

        manager = TestManager("test")
        assert manager.get_inference_count() == 0

        manager.increment_inference_count()
        manager.increment_inference_count()
        manager.increment_inference_count()

        assert manager.get_inference_count() == 3

    def test_get_status(self):
        """Test status dictionary generation."""
        from service.models.base import ModelManager

        class TestManager(ModelManager):
            def _load_model(self):
                return {}

            def _unload_model(self):
                pass

            def memory_usage(self):
                return 50 * 1024 * 1024

        manager = TestManager("test-model")
        manager.load()
        manager.increment_inference_count()

        status = manager.get_status()

        assert status["name"] == "test-model"
        assert status["loaded"] is True
        assert status["memory_bytes"] == 50 * 1024 * 1024
        assert status["inference_count"] == 1
        assert "load_time" in status
        assert "uptime_seconds" in status


class TestBirdNETManager:
    """Test BirdNETManager with mocked model loading."""

    def test_initialization(self):
        """Test BirdNETManager can be initialized without loading model."""
        from service.models.birdnet import BirdNETManager

        manager = BirdNETManager()
        assert manager.name == "BirdNET"
        assert not manager.is_loaded()
        assert manager.memory_usage() == 0
        assert manager.get_model_name() is None

    def test_load_with_mock(self, mock_birdnet_model, mock_settings, mock_birdnet_modules):
        """Test loading with mocked model."""
        # Configure the mocked modules
        mock_birdnet_modules["models"].get_model = MagicMock(return_value=mock_birdnet_model)
        mock_birdnet_modules["config"].get_settings = MagicMock(return_value=mock_settings)

        from service.models.birdnet import BirdNETManager

        manager = BirdNETManager()
        manager.load()

        assert manager.is_loaded()
        assert manager.get_model_name() == "BirdNET_GLOBAL_6K_V2.4_Model_FP16"
        assert manager.memory_usage() > 0

    def test_predict_with_mock(self, mock_birdnet_model, mock_settings, mock_birdnet_modules):
        """Test prediction with mocked model."""
        import numpy as np

        # Configure the mocked modules
        mock_birdnet_modules["models"].get_model = MagicMock(return_value=mock_birdnet_model)
        mock_birdnet_modules["config"].get_settings = MagicMock(return_value=mock_settings)

        from service.models.birdnet import BirdNETManager

        manager = BirdNETManager()

        # Fake audio chunk
        chunk = np.zeros(48000 * 3, dtype=np.float32)
        predictions = manager.predict(chunk)

        assert len(predictions) == 3
        assert predictions[0][0] == "Pica pica"
        assert predictions[0][1] == 0.95
        assert manager.get_inference_count() == 1

    def test_set_metadata_with_mock(self, mock_birdnet_model, mock_settings, mock_birdnet_modules):
        """Test setting location metadata."""
        # Configure the mocked modules
        mock_birdnet_modules["models"].get_model = MagicMock(return_value=mock_birdnet_model)
        mock_birdnet_modules["config"].get_settings = MagicMock(return_value=mock_settings)

        from service.models.birdnet import BirdNETManager

        manager = BirdNETManager()
        manager.set_metadata(lat=42.0, lon=-71.0, week=25)

        mock_birdnet_model.set_meta_data.assert_called_once_with(42.0, -71.0, 25)

    def test_status_includes_model_name(self, mock_birdnet_model, mock_settings, mock_birdnet_modules):
        """Test that status includes BirdNET-specific info."""
        # Configure the mocked modules
        mock_birdnet_modules["models"].get_model = MagicMock(return_value=mock_birdnet_model)
        mock_birdnet_modules["config"].get_settings = MagicMock(return_value=mock_settings)

        from service.models.birdnet import BirdNETManager

        manager = BirdNETManager()
        manager.load()

        status = manager.get_status()
        assert "model_name" in status
        assert status["model_name"] == "BirdNET_GLOBAL_6K_V2.4_Model_FP16"


class TestBirdNETManagerSingleton:
    """Test the singleton birdnet_manager instance."""

    def test_singleton_exists(self):
        """Test that the singleton instance is created."""
        from service.models.birdnet import birdnet_manager

        assert birdnet_manager is not None
        assert birdnet_manager.name == "BirdNET"
