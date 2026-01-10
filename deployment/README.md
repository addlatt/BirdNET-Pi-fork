# BirdNET-Pi Deployment Guide (Phase 1)

This guide covers deploying the Phase 1 infrastructure upgrade with the Go API server running alongside the existing PHP application.

## Prerequisites

- Raspberry Pi with existing BirdNET-Pi installation
- Go 1.22+ installed
- Node.js 18+ and npm installed
- Caddy web server
- PHP-FPM

## Architecture

```
                    ┌──────────────────────────────────────────┐
                    │                 Caddy                     │
                    │                (:80)                      │
                    │                                          │
                    │  /api/*    → Go API (:8080)              │
                    │  /ws       → Go WebSocket                │
                    │  /app/*    → Preact SPA (static)         │
                    │  /*        → PHP (via php-fpm)           │
                    └──────────────────────────────────────────┘
                                      │
                    ┌─────────────────┼─────────────────┐
                    │                 │                 │
                    ▼                 ▼                 ▼
            ┌───────────┐    ┌───────────┐     ┌───────────┐
            │  Go API   │    │ PHP-FPM   │     │    ML     │
            │  Server   │    │  Server   │     │  Service  │
            │  (:8080)  │    │           │     │  (:8001)  │
            └───────────┘    └───────────┘     └───────────┘
                    │                 │
                    └────────┬────────┘
                             ▼
                    ┌───────────────┐
                    │   SQLite DB   │
                    │  (birds.db)   │
                    └───────────────┘
```

## Installation Steps

### 1. Build the Go Server

```bash
cd ~/BirdNET-Pi-fork
make build
```

This creates `bin/birdnet-server`.

### 2. Build the Preact Frontend

```bash
cd web
npm install
npm run build
```

This creates `web/dist/` with production assets.

### 3. Deploy Files

```bash
# Create deployment directory
sudo mkdir -p /var/www/birdnet/bin
sudo mkdir -p /var/www/birdnet/web

# Copy binary and web assets
sudo cp bin/birdnet-server /var/www/birdnet/bin/
sudo cp -r web/dist /var/www/birdnet/web/

# Set permissions
sudo chown -R birdnet:birdnet /var/www/birdnet
```

### 4. Install Caddy (if not already installed)

```bash
sudo apt update
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https curl
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update
sudo apt install caddy
```

### 5. Configure Caddy

```bash
# Backup existing config
sudo cp /etc/caddy/Caddyfile /etc/caddy/Caddyfile.bak

# Install new config
sudo cp deployment/Caddyfile /etc/caddy/Caddyfile

# Create log directory
sudo mkdir -p /var/log/caddy
sudo chown caddy:caddy /var/log/caddy

# Reload Caddy
sudo systemctl reload caddy
```

### 6. Install and Start Go API Service

```bash
# Install service file
sudo cp deployment/birdnet-api.service /etc/systemd/system/

# Reload systemd
sudo systemctl daemon-reload

# Enable and start service
sudo systemctl enable birdnet-api
sudo systemctl start birdnet-api

# Check status
sudo systemctl status birdnet-api
```

## Verification

### Test the Go API

```bash
# Health check
curl http://localhost:8080/api/health

# List detections
curl http://localhost:8080/api/detections

# List species
curl http://localhost:8080/api/species
```

### Test via Caddy

```bash
# Health check through Caddy
curl http://localhost/api/health
```

### View Logs

```bash
# Go API logs
sudo journalctl -u birdnet-api -f

# Caddy logs
sudo tail -f /var/log/caddy/birdnet.log
```

## Rollback

If issues occur, revert to Apache:

```bash
# Stop new services
sudo systemctl stop birdnet-api caddy

# Restart Apache
sudo systemctl start apache2
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| PORT | 8080 | Go API server port |
| DB_PATH | data/db/birds.db | Path to SQLite database |
| ML_SERVICE_URL | http://127.0.0.1:8001 | Python ML service URL |
| STATIC_DIR | web/dist | Path to Preact build output |
