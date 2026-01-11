"""Analysis pipeline for BirdNET-Pi.

This module provides the file watching and analysis pipeline that:
1. Watches StreamData/ for new WAV files (via inotify)
2. Analyzes files with BirdNET
3. Saves detections to database
4. Notifies Go server for WebSocket broadcast
5. Handles audio extraction and cleanup

This can be run as a standalone daemon or integrated with the FastAPI service.
"""

import logging
import os
import os.path
import re
import signal
import sys
import threading
from queue import Queue
from subprocess import CalledProcessError
from typing import Optional

import inotify.adapters
from inotify.constants import IN_CLOSE_WRITE

from birdnet.analysis import load_global_model, run_analysis
from birdnet.classes import Detection, ParseFileName
from birdnet.config import get_settings
from birdnet.helpers import get_wav_files, ANALYZING_NOW
from birdnet.reporting import (
    extract_detection,
    summary,
    write_to_file,
    write_to_db,
    apprise,
    bird_weather,
    heartbeat,
    update_json_file,
)

from .notifier import notify_detection

log = logging.getLogger(__name__)

# Global shutdown flag
_shutdown = False


def sig_handler(sig_num, curr_stack_frame):
    """Signal handler for graceful shutdown."""
    global _shutdown
    log.info("Caught shutdown signal %d", sig_num)
    _shutdown = True


class AnalysisPipeline:
    """Manages the file watching and analysis pipeline.

    This class encapsulates the analysis loop that watches for new
    recordings and processes them through BirdNET.
    """

    def __init__(self):
        """Initialize the analysis pipeline."""
        self._shutdown = False
        self._report_queue: Optional[Queue] = None
        self._report_thread: Optional[threading.Thread] = None
        self._current_file: Optional[str] = None
        self._files_processed = 0

    def start(self) -> None:
        """Start the analysis pipeline.

        This is a blocking call that runs until shutdown is requested.
        """
        global _shutdown

        # Register signal handlers
        signal.signal(signal.SIGINT, sig_handler)
        signal.signal(signal.SIGTERM, sig_handler)

        # Load model
        load_global_model()

        conf = get_settings()
        stream_dir = os.path.join(conf["RECS_DIR"], "StreamData")

        # Set up inotify watcher
        i = inotify.adapters.Inotify()
        i.add_watch(stream_dir, mask=IN_CLOSE_WRITE)

        # Start reporting queue
        self._report_queue = Queue()
        self._report_thread = threading.Thread(
            target=self._handle_reporting_queue,
            args=(self._report_queue,),
        )
        self._report_thread.start()

        # Process backlog
        backlog = get_wav_files()
        log.info("Processing backlog of %d files", len(backlog))

        for file_name in backlog:
            if _shutdown:
                break
            self._process_file(file_name, self._report_queue)

        log.info("Backlog processing complete")

        # Watch for new files
        empty_count = 0
        recording_length = int(conf.get("RECORDING_LENGTH", 15))

        for event in i.event_gen():
            if _shutdown:
                break

            if event is None:
                # Timeout - check if we should restart
                if empty_count > (recording_length * 2 + 30):
                    log.error("No notifications for too long, restarting...")
                    break
                empty_count += 1
                continue

            (_, type_names, path, file_name) = event

            if not re.search(r"\.wav$", file_name):
                continue

            log.debug("PATH=[%s] FILENAME=[%s] EVENT_TYPES=%s", path, file_name, type_names)

            file_path = os.path.join(path, file_name)

            if file_path in backlog:
                # File was in backlog, skip duplicate processing
                backlog = []
                continue

            self._process_file(file_path, self._report_queue)
            empty_count = 0

        # Cleanup
        self._report_queue.put(None)
        self._report_thread.join()
        self._report_queue.join()
        log.info("Pipeline shutdown complete")

    def _process_file(self, file_name: str, report_queue: Queue) -> None:
        """Process a single audio file.

        Args:
            file_name: Path to the WAV file.
            report_queue: Queue for reporting results.
        """
        try:
            if os.path.getsize(file_name) == 0:
                os.remove(file_name)
                return

            log.info("Analyzing %s", file_name)
            self._current_file = file_name

            # Write lock file
            with open(ANALYZING_NOW, "w") as analyzing:
                analyzing.write(file_name)

            # Parse filename and run analysis
            file = ParseFileName(file_name)
            detections = run_analysis(file)

            # Wait for queue to be empty before adding
            if not report_queue.empty():
                log.warning("Reporting queue not yet empty")
            report_queue.join()

            # Queue for reporting
            report_queue.put((file, detections))
            self._files_processed += 1

        except BaseException as e:
            stderr = e.stderr.decode("utf-8") if isinstance(e, CalledProcessError) else ""
            log.exception("Unexpected error processing %s: %s", file_name, stderr, exc_info=e)
        finally:
            self._current_file = None

    def _handle_reporting_queue(self, queue: Queue) -> None:
        """Handle the reporting queue in a background thread.

        Args:
            queue: Queue containing (file, detections) tuples.
        """
        conf = get_settings()

        while True:
            msg = queue.get()

            # Check for shutdown signal
            if msg is None:
                break

            file, detections = msg

            try:
                # Update JSON sidecar
                update_json_file(file, detections)

                for detection in detections:
                    # Extract audio clip
                    detection.file_name_extr = extract_detection(file, detection)

                    # Log detection
                    log.info(
                        "%s;%s",
                        summary(file, detection),
                        os.path.basename(detection.file_name_extr),
                    )

                    # Write to BirdDB.txt
                    write_to_file(file, detection)

                    # Write to SQLite database
                    write_to_db(file, detection)

                    # Notify Go server for WebSocket broadcast
                    notify_detection(
                        date=detection.date,
                        time=detection.time,
                        sci_name=detection.scientific_name,
                        com_name=detection.common_name,
                        confidence=detection.confidence,
                        file_name=os.path.basename(detection.file_name_extr),
                        lat=float(conf.get("LATITUDE", 0)),
                        lon=float(conf.get("LONGITUDE", 0)),
                    )

                # Send notifications
                apprise(file, detections)
                bird_weather(file, detections)
                heartbeat()

                # Delete original WAV file
                os.remove(file.file_name)

            except BaseException as e:
                stderr = e.stderr.decode("utf-8") if isinstance(e, CalledProcessError) else ""
                log.exception("Unexpected error in reporting: %s", stderr, exc_info=e)

            queue.task_done()

        # Mark shutdown signal as processed
        queue.task_done()
        log.info("Reporting queue handler shutdown")

    def get_status(self) -> dict:
        """Get pipeline status.

        Returns:
            Dictionary with pipeline status information.
        """
        return {
            "running": not _shutdown,
            "current_file": self._current_file,
            "files_processed": self._files_processed,
            "queue_size": self._report_queue.qsize() if self._report_queue else 0,
        }


def run_pipeline() -> None:
    """Run the analysis pipeline as a standalone daemon."""
    setup_logging()
    pipeline = AnalysisPipeline()
    pipeline.start()


def setup_logging() -> None:
    """Configure logging for standalone mode."""
    logger = logging.getLogger()
    formatter = logging.Formatter("[%(name)s][%(levelname)s] %(message)s")
    handler = logging.StreamHandler(stream=sys.stdout)
    handler.setFormatter(formatter)
    logger.addHandler(handler)
    logger.setLevel(logging.INFO)


if __name__ == "__main__":
    run_pipeline()
