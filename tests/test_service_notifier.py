"""Tests for service.notifier module.

Tests the Go server notification client with mocked HTTP requests.
No actual network calls are made.
"""

from unittest.mock import patch, MagicMock

import pytest


class TestDetectionNotifier:
    """Test DetectionNotifier class."""

    def test_initialization_default_url(self):
        """Test notifier uses default URL."""
        from service.notifier import DetectionNotifier

        notifier = DetectionNotifier()
        assert "127.0.0.1:8080" in notifier.base_url

    def test_initialization_custom_url(self):
        """Test notifier accepts custom URL."""
        from service.notifier import DetectionNotifier

        notifier = DetectionNotifier("http://custom:9000")
        assert notifier.base_url == "http://custom:9000"

    def test_initialization_from_env(self):
        """Test notifier reads from environment."""
        from service.notifier import DetectionNotifier

        with patch.dict("os.environ", {"GO_SERVER_URL": "http://env-server:8080"}):
            # Need to reimport to pick up env var
            import importlib
            import service.notifier

            importlib.reload(service.notifier)
            notifier = service.notifier.DetectionNotifier()
            assert "env-server" in notifier.base_url or "127.0.0.1" in notifier.base_url

    def test_notify_detection_success(self, sample_detection):
        """Test successful detection notification."""
        from service.notifier import DetectionNotifier

        mock_response = MagicMock()
        mock_response.status_code = 200

        mock_session = MagicMock()
        mock_session.post.return_value = mock_response

        with patch("requests.Session", return_value=mock_session):
            notifier = DetectionNotifier()

            result = notifier.notify_detection(
                date=sample_detection["date"],
                time=sample_detection["time"],
                sci_name=sample_detection["sci_name"],
                com_name=sample_detection["com_name"],
                confidence=sample_detection["confidence"],
                file_name=sample_detection["file_name"],
            )

            assert result is True
            mock_session.post.assert_called_once()

            # Check the URL
            call_args = mock_session.post.call_args
            assert "/internal/detection" in call_args[0][0]

            # Check the payload
            payload = call_args[1]["json"]
            assert payload["sci_name"] == "Pica pica"
            assert payload["com_name"] == "Eurasian Magpie"

    def test_notify_detection_with_location(self, sample_detection):
        """Test notification includes lat/lon when provided."""
        from service.notifier import DetectionNotifier

        mock_response = MagicMock()
        mock_response.status_code = 200

        mock_session = MagicMock()
        mock_session.post.return_value = mock_response

        with patch("requests.Session", return_value=mock_session):
            notifier = DetectionNotifier()

            result = notifier.notify_detection(
                date=sample_detection["date"],
                time=sample_detection["time"],
                sci_name=sample_detection["sci_name"],
                com_name=sample_detection["com_name"],
                confidence=sample_detection["confidence"],
                file_name=sample_detection["file_name"],
                lat=42.3601,
                lon=-71.0589,
            )

            assert result is True
            payload = mock_session.post.call_args[1]["json"]
            assert payload["lat"] == 42.3601
            assert payload["lon"] == -71.0589

    def test_notify_detection_failure_status(self, sample_detection):
        """Test notification handles non-200 response."""
        from service.notifier import DetectionNotifier

        mock_response = MagicMock()
        mock_response.status_code = 500
        mock_response.text = "Internal Server Error"

        mock_session = MagicMock()
        mock_session.post.return_value = mock_response

        with patch("requests.Session", return_value=mock_session):
            notifier = DetectionNotifier()

            result = notifier.notify_detection(
                date=sample_detection["date"],
                time=sample_detection["time"],
                sci_name=sample_detection["sci_name"],
                com_name=sample_detection["com_name"],
                confidence=sample_detection["confidence"],
                file_name=sample_detection["file_name"],
            )

            assert result is False

    def test_notify_detection_timeout(self, sample_detection):
        """Test notification handles timeout gracefully."""
        from service.notifier import DetectionNotifier
        import requests

        mock_session = MagicMock()
        mock_session.post.side_effect = requests.Timeout("Connection timed out")

        with patch("requests.Session", return_value=mock_session):
            notifier = DetectionNotifier()

            result = notifier.notify_detection(
                date=sample_detection["date"],
                time=sample_detection["time"],
                sci_name=sample_detection["sci_name"],
                com_name=sample_detection["com_name"],
                confidence=sample_detection["confidence"],
                file_name=sample_detection["file_name"],
            )

            assert result is False

    def test_notify_detection_connection_error(self, sample_detection):
        """Test notification handles connection error gracefully."""
        from service.notifier import DetectionNotifier
        import requests

        mock_session = MagicMock()
        mock_session.post.side_effect = requests.ConnectionError("Connection refused")

        with patch("requests.Session", return_value=mock_session):
            notifier = DetectionNotifier()

            result = notifier.notify_detection(
                date=sample_detection["date"],
                time=sample_detection["time"],
                sci_name=sample_detection["sci_name"],
                com_name=sample_detection["com_name"],
                confidence=sample_detection["confidence"],
                file_name=sample_detection["file_name"],
            )

            assert result is False

    def test_health_check_success(self):
        """Test health check when Go server is reachable."""
        from service.notifier import DetectionNotifier

        mock_response = MagicMock()
        mock_response.status_code = 200

        mock_session = MagicMock()
        mock_session.get.return_value = mock_response

        with patch("requests.Session", return_value=mock_session):
            notifier = DetectionNotifier()

            result = notifier.health_check()

            assert result is True
            mock_session.get.assert_called_once()
            assert "/api/health" in mock_session.get.call_args[0][0]

    def test_health_check_failure(self):
        """Test health check when Go server is unreachable."""
        from service.notifier import DetectionNotifier
        import requests

        mock_session = MagicMock()
        mock_session.get.side_effect = requests.ConnectionError()

        with patch("requests.Session", return_value=mock_session):
            notifier = DetectionNotifier()

            result = notifier.health_check()

            assert result is False

    def test_close_session(self):
        """Test session cleanup."""
        from service.notifier import DetectionNotifier

        mock_session = MagicMock()

        with patch("requests.Session", return_value=mock_session):
            notifier = DetectionNotifier()
            # Access session to create it
            _ = notifier.session
            assert notifier._session is not None

            notifier.close()
            assert notifier._session is None
            mock_session.close.assert_called_once()


class TestNotifierConvenienceFunctions:
    """Test module-level convenience functions."""

    def test_get_notifier_singleton(self):
        """Test get_notifier returns singleton."""
        from service.notifier import get_notifier

        notifier1 = get_notifier()
        notifier2 = get_notifier()

        assert notifier1 is notifier2

    def test_notify_detection_function(self, sample_detection):
        """Test module-level notify_detection function."""
        from service import notifier

        mock_response = MagicMock()
        mock_response.status_code = 200

        mock_session = MagicMock()
        mock_session.post.return_value = mock_response

        # Reset singleton for clean test
        notifier._notifier = None

        with patch("requests.Session", return_value=mock_session):
            result = notifier.notify_detection(
                date=sample_detection["date"],
                time=sample_detection["time"],
                sci_name=sample_detection["sci_name"],
                com_name=sample_detection["com_name"],
                confidence=sample_detection["confidence"],
                file_name=sample_detection["file_name"],
            )

            assert result is True


class TestNotifyDetectionDict:
    """Test notification with dictionary input."""

    def test_notify_detection_dict(self, sample_detection):
        """Test notify_detection_dict method."""
        from service.notifier import DetectionNotifier

        mock_response = MagicMock()
        mock_response.status_code = 200

        mock_session = MagicMock()
        mock_session.post.return_value = mock_response

        with patch("requests.Session", return_value=mock_session):
            notifier = DetectionNotifier()

            result = notifier.notify_detection_dict(sample_detection)

            assert result is True
            payload = mock_session.post.call_args[1]["json"]
            assert payload == sample_detection
