# BirdNET-Pi Fork Development Guide

## Project Overview

This is a modernized fork of BirdNET-Pi with a Go + Preact + Python architecture:

- **Go API server** (port 8080): Serves REST API, WebSocket, and static files
- **Preact frontend**: Modern TypeScript UI at `/`
- **Python ML service**: BirdNET inference (existing analysis daemon)
- **Caddy**: Reverse proxy handling all routing

## Architecture

```
Browser → Caddy (port 80) → Go API (port 8080) → SQLite (read-only)
                                              → Python ML service
                                              → WebSocket hub
```

## Key File Locations

| Component | Local Path | Pi Path |
|-----------|------------|---------|
| Go server entry | `cmd/server/main.go` | `~/BirdNET-Pi/bin/birdnet-server` |
| Go API handlers | `internal/api/` | (compiled into binary) |
| Go config | `internal/config/` | (compiled into binary) |
| WebSocket hub | `internal/ws/` | (compiled into binary) |
| Database queries | `internal/db/queries.sql` | (compiled into binary) |
| Preact source | `web/src/` | `~/BirdNET-Pi/web/src/` |
| Preact build | `web/dist/` | `~/BirdNET-Pi/web/dist/` |
| Database | `data/db/birds.db` | `~/BirdNET-Pi/data/db/birds.db` |
| Config file | - | `~/BirdNET-Pi/birdnet.conf` |
| Caddyfile | `deployment/Caddyfile` | `/etc/caddy/Caddyfile` |
| Systemd service | `deployment/birdnet-api.service` | `/etc/systemd/system/birdnet-api.service` |
| Species lists | `data/species_lists/` | `~/BirdNET-Pi/data/species_lists/` |
| Bird recordings | - | `~/BirdSongs/Extracted/By_Date/` |

## Development Workflow

### Local Development

```bash
# 1. Edit Go code in cmd/server/ or internal/
# 2. Edit Preact code in web/src/

# Build Go binary (for local testing)
go build -o server ./cmd/server

# Build Preact app
cd web && npm run build
```

### Deploy to Pi

```bash
# Quick deploy: commit, push, pull on Pi, rebuild
git add -A && git commit -m "message" && git push
ssh addlatt@birdnet "cd ~/BirdNET-Pi && git pull && go build -o bin/birdnet-server ./cmd/server"
ssh addlatt@birdnet "cd ~/BirdNET-Pi/web && npm run build"
ssh addlatt@birdnet "sudo systemctl restart birdnet-api"
```

### One-liner for full deploy

```bash
git add -A && git commit -m "message" && git push && \
ssh addlatt@birdnet "cd ~/BirdNET-Pi && git pull && go build -o bin/birdnet-server ./cmd/server && cd web && npm run build && sudo systemctl restart birdnet-api"
```

## Pi Access

```bash
# SSH to Pi
ssh addlatt@birdnet

# Service management
sudo systemctl status birdnet-api      # Check Go server status
sudo systemctl restart birdnet-api     # Restart Go server
sudo systemctl stop birdnet-api        # Stop Go server
sudo journalctl -u birdnet-api -f      # View logs

# Test API
curl http://localhost:8080/api/health
curl http://localhost:8080/api/system/status
```

## Go API Structure

### Main Entry Point
- `cmd/server/main.go` - Server initialization, routing, middleware

### Internal Packages
- `internal/api/` - HTTP handlers for all endpoints
- `internal/config/` - INI config file management
- `internal/db/` - SQLite queries (sqlc generated)
- `internal/ws/` - WebSocket hub and client management
- `internal/mlclient/` - Python ML service client
- `internal/monitor/` - Memory monitoring

### Key API Endpoints

```
GET  /api/health                    # Health check
GET  /api/system/status             # Full system status
GET  /api/detections                # List detections (with search/filter)
GET  /api/species                   # List species with counts
GET  /api/settings                  # Get configuration
PUT  /api/settings                  # Update configuration
GET  /api/services                  # List systemd service statuses
POST /api/services/{name}/{action}  # Control services
WS   /ws                            # WebSocket for real-time updates
```

## Preact Frontend Structure

```
web/src/
├── app.tsx              # Main app with routing
├── main.tsx             # Entry point
├── index.css            # Global styles (Tailwind)
├── components/          # Reusable components
│   ├── Header.tsx
│   ├── DetectionList.tsx
│   ├── AudioPlayer.tsx
│   └── settings/        # Settings form components
├── pages/               # Page components
│   ├── Overview.tsx
│   ├── TodaysDetections.tsx
│   ├── History.tsx
│   ├── Spectrogram.tsx
│   ├── Recordings.tsx
│   ├── Settings.tsx
│   └── SpeciesManagement.tsx
├── hooks/               # Custom hooks
│   ├── useApi.ts        # API fetch functions
│   └── useWebSocket.ts  # WebSocket connection
└── types/               # TypeScript types
    ├── api.ts           # API response types
    └── settings.ts      # Settings types
```

## Database

SQLite database at `data/db/birds.db` with main table:

```sql
CREATE TABLE detections (
    Date DATE NOT NULL,
    Time TIME NOT NULL,
    Sci_Name VARCHAR(100) NOT NULL,
    Com_Name VARCHAR(100) NOT NULL,
    Confidence REAL,
    Lat REAL,
    Lon REAL,
    Cutoff REAL,
    Week INTEGER,
    Sens REAL,
    Overlap REAL,
    File_Name VARCHAR(100) NOT NULL
);
```

**Note:** Go server has read-only access. Python analysis daemon writes detections.

## Adding New Features

### New API Endpoint

1. Add query to `internal/db/queries.sql`
2. Run `sqlc generate` (if DB query needed)
3. Add handler in `internal/api/`
4. Register route in `cmd/server/main.go`
5. Add TypeScript types in `web/src/types/api.ts`
6. Add fetch function in `web/src/hooks/useApi.ts`

### New Page

1. Create page component in `web/src/pages/`
2. Add route in `web/src/app.tsx`
3. Add nav link in `web/src/components/Header.tsx`

## Testing

```bash
# Go tests
go test ./...

# Python tests
cd src && pytest

# Manual API testing
curl http://localhost:8080/api/health
curl http://localhost:8080/api/detections?limit=5
```

## Useful Commands

```bash
# Check what's running on Pi
ssh addlatt@birdnet "ps aux | grep birdnet"

# View recent detections
ssh addlatt@birdnet "curl -s http://localhost:8080/api/detections?limit=3 | jq"

# Check disk space
ssh addlatt@birdnet "df -h"

# Tail logs
ssh addlatt@birdnet "sudo journalctl -u birdnet-api -f"

# Rebuild and restart everything
ssh addlatt@birdnet "cd ~/BirdNET-Pi && git pull && go build -o bin/birdnet-server ./cmd/server && cd web && npm run build && sudo systemctl restart birdnet-api"
```

## Branch

Active development branch: `ralph-1`

## Ralph Loop (Autonomous Development)

This repo supports autonomous development via a Ralph loop. When running inside the loop:

1. **Read `ralph/prd.json`** — find the first story where `"passes": false`
2. **Read `ralph/progress.txt`** — learn from previous iterations
3. **Implement one story** — only one per iteration
4. **Verify** — run `make test` for Go changes, `make build-web` for frontend changes
5. **Update `ralph/prd.json`** — set `"passes": true` on the completed story
6. **Append to `ralph/progress.txt`** — log what you did and any learnings
7. **Commit** — atomic commit with a descriptive message

Key files:
- `ralph/prd.json` — structured task list with pass/fail tracking
- `ralph/progress.txt` — append-only log of iteration learnings
- `ralph/ralph.sh` — the loop runner (`./ralph/ralph.sh [max_iterations]`)

## Related Documentation

- `plans/infrastructure-upgrade.md` - Full migration plan and architecture details
- `deployment/birdnet-api.service` - Systemd service configuration
- `deployment/Caddyfile` - Web server routing configuration
