# Python ML Service

The ML service is a FastAPI application that provides BirdNET bird detection inference and integrates with the Go API server. It runs on port 8001 and is accessed only by the Go server — never directly by browsers.

## Architecture

```
Go API Server (port 8080) ──HTTP──▶ Python ML Service (port 8001)
                                       ├── /status/*     Health & status
                                       ├── /analysis/*   BirdNET inference
                                       ├── /vad/*        Part 2 stub (501)
                                       └── /llm/*        Part 2 stub (501)
```

The Go server communicates with the ML service via `internal/mlclient/`. The ML service also calls back to the Go server at `POST /internal/detection` to notify it of new detections (which are then broadcast via WebSocket).

## Quick Start (Development)

```bash
# 1. Create and activate virtual environment
python3 -m venv venv
source venv/bin/activate

# 2. Install with service dependencies
pip install -e ".[service]"

# 3. Set required environment variables
export BIRDNET_CONFIG=/path/to/birdnet.conf
export GO_SERVER_URL=http://127.0.0.1:8080

# 4. Start the service
cd src
uvicorn service.main:app --host 127.0.0.1 --port 8001

# 5. Verify
curl http://localhost:8001/status/health
# → {"status": "ok"}
```

For development with auto-reload:

```bash
uvicorn service.main:app --host 127.0.0.1 --port 8001 --reload
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `BIRDNET_CONFIG` | *(required)* | Path to `birdnet.conf` configuration file |
| `GO_SERVER_URL` | `http://127.0.0.1:8080` | Go API server URL for detection callbacks |
| `PYTHONUNBUFFERED` | `1` | Set by systemd for unbuffered log output |

## Installation

### Install Options

```bash
# Service only (FastAPI + uvicorn + pydantic)
pip install -e ".[service]"

# Development (adds pytest, httpx)
pip install -e ".[dev]"

# Full (adds tensorflow, pandas, matplotlib, etc.)
pip install -e ".[full]"

# Everything
pip install -e ".[all]"
```

Dependencies are defined in `pyproject.toml` under `[project.optional-dependencies]`.

### Verify Installation

```bash
python -c "from service.main import app; print('OK')"
```

## Systemd Service

The service runs as a systemd unit defined in `deployment/birdnet-ml.service`.

### Install on Pi

```bash
sudo cp deployment/birdnet-ml.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable birdnet-ml
sudo systemctl start birdnet-ml
```

### Key Configuration

- **User/Group:** `birdnet:birdnet`
- **Working directory:** `/opt/birdnet/src`
- **Command:** `/opt/birdnet/venv/bin/uvicorn service.main:app --host 127.0.0.1 --port 8001`
- **Memory limit:** 2GB
- **Restart:** Always, with 5-second delay
- **Starts before:** `birdnet-server.service` (Go server)

### Management Commands

```bash
sudo systemctl status birdnet-ml       # Check status
sudo systemctl restart birdnet-ml      # Restart
sudo systemctl stop birdnet-ml         # Stop
sudo journalctl -u birdnet-ml -f       # Tail logs
```

## API Endpoints

### Status (`/status/*`)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/status/health` | Health check → `{"status": "ok"}` |
| GET | `/status/status` | Full service status (models, uptime, connectivity) |
| GET | `/status/memory` | Memory usage breakdown by model |
| GET | `/status/models` | Detailed model information |

### Analysis (`/analysis/*`)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/analysis/file` | Run BirdNET inference on an audio file |
| GET | `/analysis/queue` | Queue status |
| GET | `/analysis/model` | Current BirdNET model info |
| POST | `/analysis/model/reload` | Reload the BirdNET model |

### Root Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Service info (name, version, docs URL) |
| GET | `/health` | Root-level health check for load balancers |
| GET | `/docs` | Interactive Swagger UI |
| GET | `/redoc` | ReDoc API documentation |

### Part 2 Stubs (501 Not Implemented)

VAD and LLM endpoints are defined but return 501. See CLAUDE.md "Part 2 Features" section.

## Go Integration via `internal/mlclient/`

The Go server uses `internal/mlclient/client.go` to communicate with the ML service.

### Usage

```go
import "github.com/your-org/birdnet-pi/internal/mlclient"

// Create client (30-second default timeout)
client := mlclient.New("http://127.0.0.1:8001")

// Health check
err := client.GetHealth(ctx)

// Quick boolean health check (2-second timeout)
healthy := client.IsHealthy(ctx)

// Full status
status, err := client.GetStatus(ctx)

// Memory usage
mem, err := client.GetMemoryUsage(ctx)
```

### Detection Notification Flow

When a new bird detection occurs:

1. Analysis daemon runs BirdNET inference on incoming audio
2. Detection is written to SQLite database
3. Notifier POSTs to `GO_SERVER_URL/internal/detection`
4. Go server broadcasts detection via WebSocket to connected browsers

## Lifecycle

On startup, the service:
1. Sets the service start time (for uptime tracking)
2. Pre-loads the BirdNET model (logs error but doesn't fail if loading fails)

On shutdown:
1. Unloads BirdNET model if loaded
2. Closes connections gracefully

## Key Source Files

| File | Purpose |
|------|---------|
| `src/service/main.py` | FastAPI app, lifespan, router mounting |
| `src/service/routers/status.py` | Health and status endpoints |
| `src/service/routers/analysis.py` | BirdNET inference endpoints |
| `src/service/routers/vad.py` | VAD stubs (Part 2) |
| `src/service/routers/llm.py` | LLM stubs (Part 2) |
| `src/service/models/birdnet.py` | BirdNET model manager |
| `src/service/models/base.py` | Abstract model manager base class |
| `src/service/notifier.py` | Detection notifier (Python → Go) |
| `internal/mlclient/client.go` | Go HTTP client for ML service |
| `internal/mlclient/types.go` | Go response types |
| `deployment/birdnet-ml.service` | Systemd unit file |
| `pyproject.toml` | Python dependencies and extras |
