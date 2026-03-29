#!/usr/bin/env python3
"""
Livestream Service - Streams live audio to Icecast.

This service captures audio from a microphone or RTSP stream and
streams it to an Icecast server using ffmpeg. It replaces the
livestream.sh shell script.
"""

import asyncio
import logging
import os
import signal
import subprocess
import sys
from typing import Optional, List

# Configuration
CONFIG_PATH = "/etc/birdnet/birdnet.conf"
DEFAULT_CHANNELS = 1


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
    log_level_str = config.get('LogLevel_LiveAudioStreamService', 'error')
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


class LivestreamService:
    """Manages the ffmpeg livestream process."""

    def __init__(self, config: dict):
        self.config = config
        self.logger = logging.getLogger('LivestreamService')
        self.process: Optional[subprocess.Popen] = None

        # Audio settings
        self.channels = int(config.get('CHANNELS', DEFAULT_CHANNELS))
        self.rec_card = config.get('REC_CARD', '')
        self.ice_pwd = config.get('ICE_PWD', '')

        # RTSP settings
        self.rtsp_stream = config.get('RTSP_STREAM', '')
        self.rtsp_stream_to_livestream = config.get('RTSP_STREAM_TO_LIVESTREAM', '0')

        # Frequency shift settings
        self.activate_freqshift = config.get('ACTIVATE_FREQSHIFT_IN_LIVESTREAM', 'false').lower() == 'true'
        self.freqshift_lo = config.get('FREQSHIFT_LO', '2000')
        self.freqshift_hi = config.get('FREQSHIFT_HI', '8000')

        # Logging level for ffmpeg
        log_level_str = config.get('LogLevel_LiveAudioStreamService', 'error')
        self.ffmpeg_loglevel = log_level_str if log_level_str in ['debug', 'info', 'warning', 'error', 'quiet'] else 'error'

    def get_input_source(self) -> tuple[List[str], str]:
        """
        Determine the input source and return ffmpeg input arguments.

        Returns:
            Tuple of (input_args, description)
        """
        if self.rtsp_stream:
            # Parse RTSP streams (comma-separated)
            streams = [s.strip() for s in self.rtsp_stream.split(',') if s.strip()]

            if not streams:
                raise ValueError("No valid RTSP streams configured")

            # Select the appropriate stream
            try:
                stream_index = int(self.rtsp_stream_to_livestream)
            except ValueError:
                stream_index = 0

            # Bounds check
            if stream_index < 0 or stream_index >= len(streams):
                stream_index = 0

            selected_stream = streams[stream_index]
            self.logger.info(f"Using RTSP stream {stream_index}: {selected_stream}")

            return ['-i', selected_stream], f"RTSP: {selected_stream}"

        elif self.rec_card:
            self.logger.info(f"Using PulseAudio default source (card: {self.rec_card})")
            return ['-f', 'pulse', '-i', 'default'], f"PulseAudio: default (card: {self.rec_card})"

        else:
            raise ValueError("No recording card or RTSP stream configured")

    def build_ffmpeg_command(self) -> List[str]:
        """Build the ffmpeg command for streaming."""
        input_args, description = self.get_input_source()

        cmd = ['ffmpeg', '-nostdin', '-loglevel', self.ffmpeg_loglevel]

        # Audio channels
        cmd.extend(['-ac', str(self.channels)])

        # Input source
        cmd.extend(input_args)

        # Output codec and bitrate
        cmd.extend([
            '-acodec', 'libmp3lame',
            '-b:a', '320k',
            '-ac', str(self.channels),
            '-content_type', 'audio/mpeg',
        ])

        # Frequency shift filter
        if self.activate_freqshift:
            filter_str = f'rubberband=pitch={self.freqshift_lo}/{self.freqshift_hi}'
            cmd.extend(['-af', filter_str])
            self.logger.info(f"Frequency shift enabled: {filter_str}")

        # Output to Icecast
        icecast_url = f'icecast://source:{self.ice_pwd}@localhost:8000/stream'
        cmd.extend(['-f', 'mp3', icecast_url])

        return cmd

    async def start(self) -> None:
        """Start the livestream."""
        if not self.rec_card and not self.rtsp_stream:
            self.logger.error("Stream not supported - no recording card or RTSP stream configured")
            return

        try:
            cmd = self.build_ffmpeg_command()
            self.logger.info(f"Starting ffmpeg livestream")
            self.logger.debug(f"Command: {' '.join(cmd)}")

            # Start ffmpeg process
            self.process = await asyncio.create_subprocess_exec(
                *cmd,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE
            )

            self.logger.info(f"Livestream started (PID: {self.process.pid})")

            # Wait for process and log output
            await self._monitor_process()

        except ValueError as e:
            self.logger.error(f"Configuration error: {e}")
            raise
        except Exception as e:
            self.logger.error(f"Failed to start livestream: {e}")
            raise

    async def _monitor_process(self) -> None:
        """Monitor the ffmpeg process and log output."""
        if not self.process:
            return

        async def read_stream(stream, name):
            while True:
                line = await stream.readline()
                if not line:
                    break
                self.logger.debug(f"ffmpeg {name}: {line.decode().strip()}")

        # Read both stdout and stderr
        await asyncio.gather(
            read_stream(self.process.stdout, 'stdout'),
            read_stream(self.process.stderr, 'stderr'),
            self.process.wait()
        )

        return_code = self.process.returncode
        if return_code != 0:
            self.logger.warning(f"ffmpeg exited with code {return_code}")
        else:
            self.logger.info("ffmpeg exited normally")

    async def stop(self) -> None:
        """Stop the livestream."""
        if self.process:
            self.logger.info("Stopping ffmpeg")
            self.process.terminate()

            try:
                await asyncio.wait_for(self.process.wait(), timeout=5.0)
            except asyncio.TimeoutError:
                self.logger.warning("ffmpeg did not terminate, killing")
                self.process.kill()
                await self.process.wait()

            self.process = None


async def main():
    """Main entry point."""
    config = load_config()
    setup_logging(config)

    logger = logging.getLogger('LivestreamService')
    logger.info("Livestream service starting")

    service = LivestreamService(config)

    # Handle shutdown signals
    loop = asyncio.get_event_loop()
    shutdown_event = asyncio.Event()

    def shutdown_handler(sig):
        logger.info(f"Received signal {sig}, shutting down")
        shutdown_event.set()

    for sig in (signal.SIGINT, signal.SIGTERM):
        loop.add_signal_handler(sig, lambda s=sig: shutdown_handler(s))

    # Run the service with automatic restart
    reconnect_delay = int(config.get('FREQSHIFT_RECONNECT_DELAY', '5'))

    while not shutdown_event.is_set():
        try:
            await service.start()
        except Exception as e:
            logger.error(f"Livestream error: {e}")

        if not shutdown_event.is_set():
            logger.info(f"Reconnecting in {reconnect_delay} seconds...")
            try:
                await asyncio.wait_for(shutdown_event.wait(), timeout=reconnect_delay)
            except asyncio.TimeoutError:
                pass

    # Cleanup
    await service.stop()
    logger.info("Service stopped")


if __name__ == '__main__':
    asyncio.run(main())
