#!/usr/bin/env bash
# Weekly Report Script for BirdNET-Pi
# Called by cron to generate and send weekly bird detection report
#
# Cron entry: 0 9 * * 6 addlatt /usr/local/bin/weekly_report.sh
# (Every Saturday at 9 AM)

set -e

# Find the BirdNET-Pi installation directory
if [ -d "$HOME/BirdNET-Pi" ]; then
    INSTALL_DIR="$HOME/BirdNET-Pi"
elif [ -d "/home/addlatt/BirdNET-Pi" ]; then
    INSTALL_DIR="/home/addlatt/BirdNET-Pi"
else
    echo "Error: BirdNET-Pi installation not found"
    exit 1
fi

# Activate the virtual environment if it exists
if [ -f "$INSTALL_DIR/birdnet/bin/activate" ]; then
    source "$INSTALL_DIR/birdnet/bin/activate"
fi

# Run the weekly report script
cd "$INSTALL_DIR"
python "$INSTALL_DIR/scripts/tools/weekly_report.py" "$@"
