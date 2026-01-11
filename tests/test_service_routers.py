"""Tests for service.routers module.

Tests FastAPI endpoints using TestClient with mocked dependencies.
No actual model loading or network calls.
"""

from unittest.mock import patch, MagicMock

import pytest


class TestStatusRouter:
    """Test /status/* endpoints."""

    def test_health_endpoint(self, test_client):
        """Test /status/health returns ok."""
        response = test_client.get("/status/health")
        assert response.status_code == 200
        assert response.json()["status"] == "ok"

    def test_root_health_endpoint(self, test_client):
        """Test /health root endpoint."""
        response = test_client.get("/health")
        assert response.status_code == 200
        assert response.json()["status"] == "ok"

    def test_status_endpoint(self, test_client):
        """Test /status/status returns comprehensive status."""
        # Mock the notifier health check
        with patch("service.routers.status.get_notifier") as mock_get_notifier:
            mock_notifier = MagicMock()
            mock_notifier.health_check.return_value = True
            mock_get_notifier.return_value = mock_notifier

            response = test_client.get("/status/status")

        assert response.status_code == 200
        data = response.json()

        # Check structure
        assert "service" in data
        assert "birdnet" in data
        assert "vad" in data
        assert "llm" in data
        assert "go_server" in data

        # Service info
        assert data["service"]["status"] == "ok"

        # BirdNET status (from mock)
        assert data["birdnet"]["loaded"] is True
        assert data["birdnet"]["name"] == "BirdNET"

        # Part 2 stubs
        assert data["vad"]["enabled"] is False
        assert data["llm"]["enabled"] is False

    def test_memory_endpoint(self, test_client):
        """Test /status/memory returns memory breakdown."""
        response = test_client.get("/status/memory")
        assert response.status_code == 200

        data = response.json()
        assert "birdnet" in data
        assert "vad" in data
        assert "llm" in data
        assert "total" in data
        assert "breakdown" in data

        # BirdNET should show ~500MB from mock
        assert data["birdnet"] == 500 * 1024 * 1024
        assert data["total"] == 500 * 1024 * 1024

    def test_models_endpoint(self, test_client):
        """Test /status/models returns all model info."""
        response = test_client.get("/status/models")
        assert response.status_code == 200

        data = response.json()
        assert "birdnet" in data
        assert "vad" in data
        assert "llm" in data


class TestAnalysisRouter:
    """Test /analysis/* endpoints."""

    def test_get_model_info(self, test_client, mock_birdnet_manager):
        """Test /analysis/model returns model info."""
        # Set up the mock to return a model with expected attributes
        mock_model = MagicMock()
        mock_model.sample_rate = 48000
        mock_model.chunk_duration = 3
        mock_birdnet_manager.get_model.return_value = mock_model

        response = test_client.get("/analysis/model")
        assert response.status_code == 200

        data = response.json()
        assert "name" in data
        assert "loaded" in data
        assert "sample_rate" in data
        assert "chunk_duration" in data

    def test_get_queue_status(self, test_client):
        """Test /analysis/queue returns queue info."""
        response = test_client.get("/analysis/queue")
        assert response.status_code == 200

        data = response.json()
        assert "queue_length" in data
        assert "processing" in data

    def test_analyze_file_not_found(self, test_client):
        """Test /analysis/file returns 404 for missing file."""
        response = test_client.post(
            "/analysis/file",
            json={"file_path": "/nonexistent/file.wav"},
        )
        assert response.status_code == 404
        assert "not found" in response.json()["detail"].lower()

    def test_analyze_file_success(self, test_client, temp_audio_path, mock_settings):
        """Test /analysis/file with mocked analysis."""
        import sys

        # Mock birdnet modules to avoid TensorFlow import
        mock_analysis = MagicMock()
        mock_config = MagicMock()

        mock_chunks = [[0.0] * 48000 * 3]  # Fake audio chunks
        mock_detections = {
            "0.0;3.0": [("Pica pica", 0.95), ("Corvus corax", 0.12)],
        }

        mock_analysis.readAudioData = MagicMock(return_value=mock_chunks)
        mock_analysis.analyzeAudioData = MagicMock(return_value=(mock_detections, []))
        mock_config.get_settings = MagicMock(return_value=mock_settings)

        # Insert mock modules
        sys.modules["birdnet.analysis"] = mock_analysis
        sys.modules["birdnet.config"] = mock_config

        try:
            response = test_client.post(
                "/analysis/file",
                json={"file_path": temp_audio_path},
            )

            assert response.status_code == 200
            data = response.json()

            assert data["file_path"] == temp_audio_path
            assert "detections" in data
            assert "duration_seconds" in data
            assert len(data["detections"]) == 1  # Only one above threshold (0.7)
        finally:
            # Cleanup
            sys.modules.pop("birdnet.analysis", None)
            sys.modules.pop("birdnet.config", None)


class TestVADRouter:
    """Test /vad/* endpoints (Part 2 stubs)."""

    def test_vad_status(self, test_client):
        """Test /vad/status returns not implemented status."""
        response = test_client.get("/vad/status")
        assert response.status_code == 200

        data = response.json()
        assert data["enabled"] is False
        assert data["status"] == "not_implemented"

    def test_vad_check_not_implemented(self, test_client):
        """Test /vad/check returns 501."""
        response = test_client.post(
            "/vad/check",
            json={"audio_path": "/some/file.wav"},
        )
        assert response.status_code == 501
        assert "Part 2" in response.json()["detail"]

    def test_vad_load_not_implemented(self, test_client):
        """Test /vad/load returns 501."""
        response = test_client.post("/vad/load")
        assert response.status_code == 501


class TestLLMRouter:
    """Test /llm/* endpoints (Part 2 stubs)."""

    def test_llm_status(self, test_client):
        """Test /llm/status returns not implemented status."""
        response = test_client.get("/llm/status")
        assert response.status_code == 200

        data = response.json()
        assert data["enabled"] is False
        assert data["status"] == "not_implemented"

    def test_llm_ask_not_implemented(self, test_client):
        """Test /llm/ask returns 501."""
        response = test_client.post(
            "/llm/ask",
            json={"question": "What bird is this?"},
        )
        assert response.status_code == 501
        assert "Part 2" in response.json()["detail"]

    def test_llm_models_list(self, test_client):
        """Test /llm/models returns available models."""
        response = test_client.get("/llm/models")
        assert response.status_code == 200

        data = response.json()
        assert "available_models" in data
        assert len(data["available_models"]) > 0


class TestRootEndpoints:
    """Test root-level endpoints."""

    def test_root_info(self, test_client):
        """Test / returns service info."""
        response = test_client.get("/")
        assert response.status_code == 200

        data = response.json()
        assert data["service"] == "BirdNET-Pi ML Service"
        assert "version" in data
        assert "docs" in data
