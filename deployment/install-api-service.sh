#!/bin/bash
# Install and enable the BirdNET API systemd service
# Run on the Pi: bash ~/BirdNET-Pi/deployment/install-api-service.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVICE_TEMPLATE="$SCRIPT_DIR/birdnet-api.service"
SERVICE_NAME="birdnet-api"
TEMP_SERVICE="/tmp/${SERVICE_NAME}.service"

echo "Installing $SERVICE_NAME service..."

# Stop any manually running instance
pkill -f 'bin/birdnet-server' 2>/dev/null || true

# Substitute placeholders in service file
echo "Configuring service for user: $USER"
sed -e "s|%USER%|$USER|g" \
    -e "s|%HOME%|$HOME|g" \
    "$SERVICE_TEMPLATE" > "$TEMP_SERVICE"

# Copy service file to systemd
sudo cp "$TEMP_SERVICE" /etc/systemd/system/${SERVICE_NAME}.service
rm -f "$TEMP_SERVICE"

# Reload systemd to pick up new service
sudo systemctl daemon-reload

# Enable service to start on boot
sudo systemctl enable "$SERVICE_NAME"

# Start the service
sudo systemctl start "$SERVICE_NAME"

# Show status
echo ""
echo "Service installed and started!"
echo ""
sudo systemctl status "$SERVICE_NAME" --no-pager

echo ""
echo "Useful commands:"
echo "  View logs:        sudo journalctl -u $SERVICE_NAME -f"
echo "  Restart:          sudo systemctl restart $SERVICE_NAME"
echo "  Stop:             sudo systemctl stop $SERVICE_NAME"
echo "  Check status:     sudo systemctl status $SERVICE_NAME"
