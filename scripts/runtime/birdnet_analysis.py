#!/usr/bin/env python3
"""Wrapper script for the BirdNET analysis pipeline.

This script provides a backward-compatible entry point for the systemd service.
It imports and runs the new pipeline from src/service/pipeline.py.
"""

import sys
import os

# Add src directory to Python path
src_dir = os.path.join(os.path.dirname(os.path.dirname(os.path.dirname(__file__))), "src")
sys.path.insert(0, src_dir)

from service.pipeline import run_pipeline

if __name__ == "__main__":
    run_pipeline()
