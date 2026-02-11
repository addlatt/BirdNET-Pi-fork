# BirdNET-Pi Deployment Guide

This guide covers deploying BirdNET-Pi with the Go API server, Preact frontend, and Python services.

## Prerequisites

- Raspberry Pi with existing BirdNET-Pi installation
- Go 1.22+ installed
- Node.js 18+ and npm installed
- Caddy web server

## Architecture

```
                    ┌──────────────────────────────────────────┐
                    │                 Caddy                     │
                    │                (:80)                      │
                    │                                          │
                    │  /api/*    → Go API (:8080)              │
                    │  /ws       → Go WebSocket (:8080)        │
                    │  /*        → Preact SPA (static)         │
                    └──────────────────────────────────────────┘
                                      │
                    ┌─────────────────┼─────────────────┐
                    │                 │                 │
                    ▼                 ▼                 ▼
            ┌───────────┐    ┌───────────┐     ┌───────────┐
            │  Go API   │    │  Preact   │     │    ML     │
            │  Server   │    │  SPA      │     │  Service  │
            │  (:8080)  │    │ (static)  │     │  (:8001)  │
            └───────────┘    └───────────┘     └───────────┘
                    │
                    ▼
            ┌───────────────┐
            │   SQLite DB   │
            │  (birds.db)   │
            └───────────────┘
```

## Installation Steps

### 1. Build the Go Server

```bash
cd ~/BirdNET-Pi
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

### 3. Install Caddy (if not already installed)

```bash
sudo apt update
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https curl
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update
sudo apt install caddy
```

### 4. Configure Caddy

```bash
# Backup existing config
sudo cp /etc/caddy/Caddyfile /etc/caddy/Caddyfile.bak

# Install new config
sudo cp deployment/Caddyfile /etc/caddy/Caddyfile

# Reload Caddy
sudo systemctl reload caddy
```

### 5. Install and Start Go API Service

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
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| PORT | 8080 | Go API server port |
| DB_PATH | data/db/birds.db | Path to SQLite database |
| ML_SERVICE_URL | http://127.0.0.1:8001 | Python ML service URL |
| STATIC_DIR | web/dist | Path to Preact build output |
