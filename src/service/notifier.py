"""Detection notifier for Go server communication.

Sends detection notifications to the Go backend for WebSocket broadcast
to connected frontend clients.
"""

import logging
import os
from typing import Optional

import requests

log = logging.getLogger(__name__)

# Go server URL - defaults to localhost:8080
GO_SERVER_URL = os.environ.get("GO_SERVER_URL", "http://127.0.0.1:8080")

# Timeout for notification requests (should be short, fire-and-forget)
NOTIFY_TIMEOUT = 1.0


class DetectionNotifier:
    """HTTP client for notifying Go server of new detections.

    The Go server will broadcast these to WebSocket clients for real-time
    updates. Failures are non-fatal since the detection is already in the DB.
    """

    def __init__(self, base_url: Optional[str] = None):
        """Initialize the notifier.

        Args:
            base_url: Go server URL. Defaults to GO_SERVER_URL env var.
        """
        self.base_url = base_url or GO_SERVER_URL
        self._session: Optional[requests.Session] = None

    @property
    def session(self) -> requests.Session:
        """Get or create the requests session."""
        if self._session is None:
            self._session = requests.Session()
        return self._session

    def notify_detection(
        self,
        date: str,
        time: str,
        sci_name: str,
        com_name: str,
        confidence: float,
        file_name: str,
        lat: Optional[float] = None,
        lon: Optional[float] = None,
    ) -> bool:
        """Notify Go server of a new detection.

        Args:
            date: Detection date (YYYY-MM-DD).
            time: Detection time (HH:MM:SS).
            sci_name: Scientific species name.
            com_name: Common species name.
            confidence: Detection confidence (0.0-1.0).
            file_name: Extracted audio file name.
            lat: Optional latitude.
            lon: Optional longitude.

        Returns:
            True if notification succeeded, False otherwise.
        """
        detection = {
            "date": date,
            "time": time,
            "sci_name": sci_name,
            "com_name": com_name,
            "confidence": confidence,
            "file_name": file_name,
        }

        if lat is not None:
            detection["lat"] = lat
        if lon is not None:
            detection["lon"] = lon

        return self._send_notification(detection)

    def notify_detection_dict(self, detection: dict) -> bool:
        """Notify Go server using a detection dictionary.

        Args:
            detection: Dictionary with detection fields.

        Returns:
            True if notification succeeded, False otherwise.
        """
        return self._send_notification(detection)

    def _send_notification(self, detection: dict) -> bool:
        """Send detection notification to Go server.

        Args:
            detection: Detection data to send.

        Returns:
            True if notification succeeded, False otherwise.
        """
        url = f"{self.base_url}/internal/detection"

        try:
            response = self.session.post(
                url,
                json=detection,
                timeout=NOTIFY_TIMEOUT,
            )

            if response.status_code == 200:
                log.debug(
                    "Detection notification sent: %s (%s)",
                    detection.get("com_name"),
                    detection.get("sci_name"),
                )
                return True
            else:
                log.warning(
                    "Detection notification failed with status %d: %s",
                    response.status_code,
                    response.text,
                )
                return False

        except requests.Timeout:
            log.warning("Detection notification timed out")
            return False
        except requests.ConnectionError:
            log.warning("Could not connect to Go server at %s", self.base_url)
            return False
        except requests.RequestException as e:
            log.warning("Detection notification failed: %s", e)
            return False

    def health_check(self) -> bool:
        """Check if the Go server is reachable.

        Returns:
            True if Go server is healthy, False otherwise.
        """
        url = f"{self.base_url}/api/health"

        try:
            response = self.session.get(url, timeout=NOTIFY_TIMEOUT)
            return response.status_code == 200
        except requests.RequestException:
            return False

    def close(self) -> None:
        """Close the HTTP session."""
        if self._session is not None:
            self._session.close()
            self._session = None


# Module-level singleton for convenience
_notifier: Optional[DetectionNotifier] = None


def get_notifier() -> DetectionNotifier:
    """Get the singleton notifier instance."""
    global _notifier
    if _notifier is None:
        _notifier = DetectionNotifier()
    return _notifier


def notify_detection(
    date: str,
    time: str,
    sci_name: str,
    com_name: str,
    confidence: float,
    file_name: str,
    lat: Optional[float] = None,
    lon: Optional[float] = None,
) -> bool:
    """Convenience function to notify Go server of a detection.

    Args:
        date: Detection date (YYYY-MM-DD).
        time: Detection time (HH:MM:SS).
        sci_name: Scientific species name.
        com_name: Common species name.
        confidence: Detection confidence (0.0-1.0).
        file_name: Extracted audio file name.
        lat: Optional latitude.
        lon: Optional longitude.

    Returns:
        True if notification succeeded, False otherwise.
    """
    return get_notifier().notify_detection(
        date=date,
        time=time,
        sci_name=sci_name,
        com_name=com_name,
        confidence=confidence,
        file_name=file_name,
        lat=lat,
        lon=lon,
    )
