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
- `internal/images/` - Bird image fetching and caching
- `internal/mlclient/` - Python ML service client
- `internal/monitor/` - Memory monitoring
- `internal/scheduler/` - Task scheduling and execution
- `internal/tasks/` - Background task definitions
- `internal/testutil/` - Shared test helpers
- `internal/ws/` - WebSocket hub and client management

### API Endpoints

```
# Health
GET  /api/health

# Detections
GET    /api/detections                              # List (search/filter/paginate)
GET    /api/detections/{date}/{time}/{species}       # Single detection
DELETE /api/detections/{date}/{time}/{species}       # Delete detection
POST   /api/detections/reclassify                   # Re-run ML on recording

# Species
GET    /api/species                                 # List with counts
GET    /api/species/all                             # All known species
GET    /api/species/ranking                         # Top species ranking
GET    /api/species/{name}                          # Detail + recent detections
GET    /api/species/{name}/history                  # Detection history
GET    /api/species/{name}/count                    # Total count
GET    /api/species/{name}/image                    # Bird photo
POST   /api/species/{name}/image/blacklist          # Blacklist a photo URL
DELETE /api/species/{name}/all                      # Delete all detections

# Species Lists
GET  /api/species-lists                             # List include/exclude lists
PUT  /api/species-lists/{listType}                  # Replace list
POST /api/species-lists/{listType}/add              # Add entry
POST /api/species-lists/{listType}/remove           # Remove entry

# Stats & Heatmap
GET  /api/stats                                     # Summary statistics
GET  /api/heatmap/today                             # Today's hourly activity
GET  /api/dates                                     # Available detection dates

# Spectrogram & Streaming
GET  /api/spectrogram/info                          # Current spectrogram info
GET  /api/spectrogram/image                         # Spectrogram PNG
GET  /api/spectrogram/detections                    # Detections overlay
GET  /api/stream                                    # Live audio stream proxy

# System
GET  /api/system/status                             # Full system status
GET  /api/system/memory                             # Memory usage
GET  /api/system/update-check                       # Check for updates
POST /api/system/reboot                             # Reboot Pi
POST /api/system/shutdown                           # Shutdown Pi

# Settings
GET  /api/settings                                  # Get configuration
PUT  /api/settings                                  # Update configuration
GET  /api/settings/schema                           # Settings JSON schema
POST /api/settings/caddy/regenerate                 # Regenerate Caddyfile

# Services
GET  /api/services                                  # List systemd service statuses
POST /api/services/restart-all                      # Restart all services
POST /api/services/{name}/{action}                  # Control a service

# Recordings
GET  /api/recordings/dates                          # Available recording dates
GET  /api/recordings/species                        # Species with recordings
GET  /api/recordings/by-date/{date}                 # Recordings on a date
GET  /api/recordings/by-species/{name}              # Recordings of a species
POST /api/recordings/{date}/{species}/{file}/delete # Delete recording
POST /api/recordings/{date}/{species}/{file}/change # Reclassify recording
POST /api/recordings/{date}/{species}/{file}/lock   # Lock/unlock recording
POST /api/recordings/{date}/{species}/{file}/shift  # Shift detection window
GET  /api/recordings/exclusions                     # Exclusion list

# Reports
GET  /api/reports/weekly                            # Weekly summary
GET  /api/reports/weekly/export                     # Export weekly CSV

# Diagnostics
GET  /api/diagnostics/disk                          # Disk usage
GET  /api/diagnostics/most-recent                   # Most recent detection
GET  /api/diagnostics/pi                            # Pi hardware info
GET  /api/diagnostics/system                        # System diagnostics
GET  /api/diagnostics/species-count                 # Species count
GET  /api/diagnostics/logs                          # Recent log entries
GET  /api/logs/recent                               # Tail log output

# Image Cache
GET  /api/images/cache/stats                        # Cache statistics
POST /api/images/cache/refresh                      # Refresh cache

# Task Scheduler
GET  /api/tasks                                     # List scheduled tasks
GET  /api/tasks/history                             # All task run history
GET  /api/tasks/{name}                              # Task detail
POST /api/tasks/{name}/run                          # Trigger task
POST /api/tasks/{name}/cancel                       # Cancel running task
GET  /api/tasks/{name}/history                      # Task-specific history

# Backup & Restore
POST /api/backup/create                             # Create backup
POST /api/backup/restore                            # Restore from backup
GET  /api/backup/status                             # Backup status

# Labels
GET  /api/labels                                    # Detection labels
GET  /api/labels/model                              # Model label list

# Internal (Python ML → Go)
POST /internal/detection                            # Submit new detection

# WebSocket
WS   /ws                                            # Real-time detection events
WS   /ws/logs                                       # Live log stream
WS   /ws/logs/detections                            # Detection-specific log stream
```

## Preact Frontend Structure

```
web/src/
├── app.tsx              # Main app with routing
├── main.tsx             # Entry point
├── index.css            # Global styles (Tailwind)
├── components/          # Reusable components
│   ├── AudioPlayer.tsx
│   ├── BirdActivityHeatmap.tsx
│   ├── BirdImage.tsx
│   ├── DatePicker.tsx
│   ├── DetectionList.tsx
│   ├── EnhancedAudioPlayer.tsx
│   ├── Header.tsx
│   ├── OverviewStatsCards.tsx
│   ├── SearchFilters.tsx
│   ├── ServiceControls.tsx
│   ├── Spectrogram.tsx
│   ├── SpeciesDetail.tsx
│   ├── SpeciesListEditor.tsx
│   ├── SpeciesMiniChart.tsx
│   ├── SpeciesRankingList.tsx
│   ├── SpeciesTable.tsx
│   ├── StatsCards.tsx
│   ├── StatsHeader.tsx
│   └── settings/        # Settings form components
│       ├── FormInputs.tsx
│       └── NotificationSpeciesSelector.tsx
├── pages/               # Page components
│   ├── AdvancedSettings.tsx
│   ├── Backup.tsx
│   ├── Detections.tsx
│   ├── Overview.tsx
│   ├── Recordings.tsx
│   ├── Settings.tsx
│   ├── SpeciesManagement.tsx
│   ├── Spectrogram.tsx
│   └── Stats.tsx
├── hooks/               # Custom hooks
│   ├── useApi.ts        # API fetch functions
│   ├── useSettings.ts   # Settings state management
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
- `ralph/` - Autonomous development loop (PRD, progress log, runner script)
