#!/usr/bin/env python3
"""
Spectrogram Service - Generates live spectrograms from audio files.

This service monitors for new audio files being analyzed and generates
spectrogram images using sox. It replaces the spectrogram.sh shell script.
"""

import asyncio
import logging
import os
import signal
import subprocess
import sys
import time
from pathlib import Path
from typing import Optional

# Try to use inotify for efficient file watching
try:
    from watchdog.observers import Observer
    from watchdog.events import FileSystemEventHandler, FileModifiedEvent
    HAS_WATCHDOG = True
except ImportError:
    HAS_WATCHDOG = False
    logging.warning("watchdog not installed, falling back to polling")

# Configuration
CONFIG_PATH = "/etc/birdnet/birdnet.conf"
DEFAULT_RECORDING_LENGTH = 15


def load_config() -> dict:
    """Load configuration from birdnet.conf."""
    config = {}

    if not os.path.exists(CONFIG_PATH):
        logging.warning(f"Config file not found: {CONFIG_PATH}")
        return config

    with open(CONFIG_PATH, 'r') as f:
        for line in f:
            line = line.strip()
            if not line or line.startswith('#'):
                continue

            if '=' in line:
                key, value = line.split('=', 1)
                key = key.strip()
                value = value.strip().strip('"').strip("'")
                config[key] = value

    return config


def setup_logging(config: dict) -> None:
    """Configure logging based on config."""
    log_level_str = config.get('LogLevel_SpectrogramViewerService', 'error')
    log_level_map = {
        'debug': logging.DEBUG,
        'info': logging.INFO,
        'warning': logging.WARNING,
        'error': logging.ERROR,
    }
    log_level = log_level_map.get(log_level_str.lower(), logging.ERROR)

    logging.basicConfig(
        level=log_level,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
        stream=sys.stdout
    )


class SpectrogramGenerator:
    """Generates spectrograms from audio files using sox."""

    def __init__(self, config: dict):
        self.config = config
        self.home = os.path.expanduser('~')
        self.extracted_dir = config.get('EXTRACTED', f'{self.home}/BirdSongs/Extracted')
        self.stream_data_dir = f'{self.home}/BirdSongs/StreamData'
        self.analyzing_file = os.path.join(self.stream_data_dir, 'analyzing_now.txt')
        self.output_file = os.path.join(self.extracted_dir, 'spectrogram.png')

        # Calculate loop time (2/3 of recording length)
        recording_length = int(config.get('RECORDING_LENGTH', DEFAULT_RECORDING_LENGTH))
        self.loop_time = recording_length * 2 // 3
        self.last_run = 0

        # Raw spectrogram option
        self.raw_spectrogram = config.get('RAW_SPECTROGRAM', '0') == '1'

        self.logger = logging.getLogger('SpectrogramGenerator')

        # Ensure directories exist
        os.makedirs(self.stream_data_dir, exist_ok=True)
        os.makedirs(self.extracted_dir, exist_ok=True)

        # Create analyzing_now.txt if it doesn't exist
        if not os.path.exists(self.analyzing_file):
            Path(self.analyzing_file).touch()

    def generate(self) -> bool:
        """Generate a spectrogram from the current analyzing file."""
        now = time.time()

        # Rate limit
        if now < self.last_run + self.loop_time:
            return False

        # Read the current file being analyzed
        try:
            with open(self.analyzing_file, 'r') as f:
                audio_file = f.read().strip()
        except Exception as e:
            self.logger.debug(f"Could not read analyzing file: {e}")
            return False

        if not audio_file or not os.path.exists(audio_file):
            self.logger.debug(f"Audio file not found: {audio_file}")
            return False

        # Generate spectrogram
        try:
            # Build sox command
            # Remove home prefix from comment
            comment = audio_file.replace(self.home + '/', '')

            cmd = [
                'sox', '-V1',
                audio_file,
                '-n',  # Output to null (we only want spectrogram)
                'remix', '1',  # Mix to mono
                'rate', '24k',  # Resample to 24kHz
                'spectrogram',
                '-c', comment,  # Comment (filename)
                '-o', self.output_file
            ]

            if self.raw_spectrogram:
                cmd.append('-r')  # Raw spectrogram (no axes)

            self.logger.info(f"Generating spectrogram for: {audio_file}")
            result = subprocess.run(cmd, capture_output=True, text=True)

            if result.returncode != 0:
                self.logger.error(f"Sox failed: {result.stderr}")
                return False

            self.last_run = now
            self.logger.debug("Spectrogram generated successfully")
            return True

        except Exception as e:
            self.logger.error(f"Failed to generate spectrogram: {e}")
            return False


if HAS_WATCHDOG:
    class AnalyzingFileHandler(FileSystemEventHandler):
        """Handles file modification events for the analyzing_now.txt file."""

        def __init__(self, generator: SpectrogramGenerator):
            self.generator = generator
            self.logger = logging.getLogger('FileHandler')

        def on_modified(self, event):
            if event.is_directory:
                return

            if event.src_path.endswith('analyzing_now.txt'):
                self.logger.debug(f"File modified: {event.src_path}")
                self.generator.generate()


async def run_with_watchdog(generator: SpectrogramGenerator):
    """Run the service using watchdog for file monitoring."""
    logger = logging.getLogger('SpectrogramService')
    logger.info("Starting spectrogram service with watchdog")

    event_handler = AnalyzingFileHandler(generator)
    observer = Observer()
    observer.schedule(event_handler, generator.stream_data_dir, recursive=False)
    observer.start()

    try:
        while True:
            await asyncio.sleep(1)
    except asyncio.CancelledError:
        observer.stop()
        observer.join()
        raise


async def run_with_polling(generator: SpectrogramGenerator):
    """Run the service using polling for file monitoring."""
    logger = logging.getLogger('SpectrogramService')
    logger.info("Starting spectrogram service with polling")

    last_mtime = 0

    try:
        while True:
            try:
                stat = os.stat(generator.analyzing_file)
                if stat.st_mtime > last_mtime:
                    last_mtime = stat.st_mtime
                    generator.generate()
            except FileNotFoundError:
                pass

            await asyncio.sleep(1)
    except asyncio.CancelledError:
        raise


async def main():
    """Main entry point."""
    config = load_config()
    setup_logging(config)

    logger = logging.getLogger('SpectrogramService')
    logger.info("Spectrogram service starting")

    generator = SpectrogramGenerator(config)

    # Handle shutdown signals
    loop = asyncio.get_event_loop()

    def shutdown_handler(sig):
        logger.info(f"Received signal {sig}, shutting down")
        for task in asyncio.all_tasks(loop):
            task.cancel()

    for sig in (signal.SIGINT, signal.SIGTERM):
        loop.add_signal_handler(sig, lambda s=sig: shutdown_handler(s))

    # Run the service
    try:
        if HAS_WATCHDOG:
            await run_with_watchdog(generator)
        else:
            await run_with_polling(generator)
    except asyncio.CancelledError:
        logger.info("Service stopped")


if __name__ == '__main__':
    asyncio.run(main())
