# BirdNET-Pi Infrastructure Upgrade Plan

## Goal

Migrate from the current shell/PHP/Python mix to a clean, maintainable Go + Python + Preact architecture using an **incremental approach** that minimizes risk and delivers value early.

**Target Hardware:** Raspberry Pi 5 (4GB RAM, NVMe storage, Debian Trixie)

**Future-Proofing:** This plan is designed to support Part 2 features (VAD, LLM, interactive charts) without architectural changes.

---

## Current State

- Shell scripts for orchestration and scheduling
- PHP for web interface (recently reorganized with front-controller pattern)
- Python for BirdNET inference (recently restructured as proper package)
- Centralized configuration with JSON schema
- SQLite database with single `detections` table

## Target State

```
┌─────────────────────────────────────────────────────────────────┐
│                     Preact Frontend                              │
│         Modern UI, WebSocket for real-time updates               │
└────────────────────────┬────────────────────────────────────────┘
                         │ HTTP / WebSocket (typed messages)
┌────────────────────────▼────────────────────────────────────────┐
│                      Go Backend                                  │
│     API server, static files, scheduling, config management      │
│     SQLite read-only, proxies to Python for ML operations        │
│     Memory monitoring, WebSocket hub with channels               │
└──────────┬──────────────────────────────────────────────────────┘
           │ HTTP (bidirectional request/response)
           │
┌──────────▼──────────────────────────────────────────────────────┐
│                   Python ML Service                              │
│     FastAPI application (extensible router architecture)         │
│     BirdNET inference, detection notifications to Go             │
│     Model manager abstraction for future models (VAD, LLM)       │
└─────────────────────────────────────────────────────────────────┘
           │
           │ Direct write
           ▼
┌──────────────────┐
│     SQLite       │
│  (shared read)   │
└──────────────────┘
```

### Key Architecture Decisions

**1. Python as Full FastAPI Service (not minimal daemon)**
- Extensible router architecture supports Part 2 features (VAD, LLM)
- Model manager abstraction enables lazy loading/unloading
- Request/response pattern for future ML operations

**2. Go ↔ Python Bidirectional Communication**
- Python → Go: Detection notifications (fire-and-forget)
- Go → Python: Status checks, future ML requests (request/response with timeouts)

**3. WebSocket with Typed Messages**
- Channel-based subscriptions for different data streams
- Supports Part 2 features (spectrogram streaming, LLM responses)

**4. Memory-Aware Design**
- Monitoring infrastructure for Part 2 memory-heavy features
- Model manager supports load/unload lifecycle

---

## Technology Choices

### Frontend

| Technology | Size | Purpose |
|------------|------|---------|
| Preact | ~3KB | UI framework, React-compatible |
| Vite | dev only | Build tooling |
| uPlot | ~30KB | Charts (detection timeline, species counts) |
| Tailwind CSS | purged | Styling |

### Backend

| Technology | Purpose |
|------------|---------|
| Go | API server, static files, scheduling, config management |
| Chi | HTTP router |
| sqlc | Type-safe database access (read-only) |
| gorilla/websocket | Real-time updates with typed messages |
| golang-migrate | Database schema versioning |
| SQLite | Database (read by Go, written by Python) |

### Python ML Service

| Technology | Purpose |
|------------|---------|
| Python 3.11+ | Runtime |
| FastAPI | Full API framework (extensible for Part 2) |
| Existing `src/birdnet/` | Analysis, inference, reporting |
| inotify | File watching |
| SQLite | Direct writes |

---

## Project Structure

```
birdnet-pi/
├── cmd/
│   └── server/
│       └── main.go              # Go server entrypoint
├── internal/
│   ├── api/
│   │   ├── routes.go            # Route definitions
│   │   ├── detections.go        # Detection endpoints (read-only)
│   │   ├── species.go           # Species endpoints
│   │   ├── charts.go            # Chart data endpoints
│   │   ├── settings.go          # Config endpoints
│   │   ├── system.go            # System status, memory monitoring
│   │   └── internal.go          # Internal endpoints (Python notifications)
│   ├── db/
│   │   ├── schema.sql           # Database schema
│   │   ├── queries.sql          # sqlc queries (read-only)
│   │   └── generated/           # sqlc output
│   ├── ws/
│   │   ├── hub.go               # WebSocket hub with channels
│   │   ├── client.go            # WebSocket client management
│   │   └── messages.go          # Typed message definitions
│   ├── mlclient/
│   │   ├── client.go            # Python service client
│   │   └── types.go             # Request/response types
│   ├── config/
│   │   ├── config.go            # YAML config with schema validation
│   │   └── schema.go            # Config schema definitions
│   ├── monitor/
│   │   └── memory.go            # Memory monitoring (Part 2 ready)
│   └── scheduler/
│       └── jobs.go              # Cron-style jobs (cleanup, disk check)
├── migrations/
│   ├── 000001_initial_schema.up.sql
│   ├── 000001_initial_schema.down.sql
│   └── ...
├── src/
│   ├── birdnet/                 # Existing Python package
│   │   ├── analysis.py
│   │   ├── models.py
│   │   ├── reporting.py
│   │   ├── config.py
│   │   └── ...
│   └── service/                 # New: FastAPI service
│       ├── main.py              # FastAPI application
│       ├── routers/
│       │   ├── analysis.py      # /analysis/* endpoints
│       │   ├── status.py        # /health, /status, /memory
│       │   ├── vad.py           # /vad/* (Part 2, stub)
│       │   └── llm.py           # /llm/* (Part 2, stub)
│       ├── models/
│       │   ├── base.py          # ModelManager base class
│       │   ├── birdnet.py       # BirdNET model manager
│       │   ├── vad.py           # VAD model manager (Part 2)
│       │   └── llm.py           # LLM model manager (Part 2)
│       ├── pipeline.py          # Analysis pipeline
│       └── notifier.py          # Go notification client
├── web/
│   ├── src/
│   │   ├── app.jsx
│   │   ├── components/
│   │   │   ├── DetectionList.jsx
│   │   │   ├── SpeciesChart.jsx
│   │   │   ├── Spectrogram.jsx
│   │   │   └── Settings.jsx
│   │   ├── pages/
│   │   │   ├── Overview.jsx
│   │   │   ├── TodaysDetections.jsx
│   │   │   ├── Stats.jsx
│   │   │   ├── History.jsx
│   │   │   └── Settings.jsx
│   │   └── hooks/
│   │       ├── useWebSocket.js
│   │       └── useChartData.js
│   ├── index.html
│   └── vite.config.js
├── scripts/
│   ├── install/                 # KEEP: Installation scripts (shell)
│   ├── runtime/                 # MIGRATE: Recording daemon to Go
│   ├── tools/                   # KEEP: CLI utilities (shell)
│   └── config/                  # KEEP: Config management (shell)
├── config.yaml                  # Unified configuration
├── config.schema.yaml           # Validation schema
└── README.md
```

---

## Migration Phases (Incremental Approach)

### Phase 1: Go API Foundation + Preact Shell (Week 1) ✅ COMPLETE

**Goal:** Go server running alongside existing PHP, serving one Preact page, with Part 2-ready architecture.

**Implementation Summary:**
The actual Pi database schema differed from the plan—it uses capitalized column names (Date, Time, Sci_Name) with no auto-increment ID or created_at column, requiring a composite key pattern (date/time/species) for lookups. SQLite's WAL mode is incompatible with read-only connections, so the connection string was simplified to `mode=ro&cache=shared`. The Preact + Vite build produces minimal bundles (~35KB JS gzipped), and the Go server successfully serves real detection data from the Pi database with all tests passing.

**Tasks:**
- [x] Initialize Go module with Chi router
- [x] Set up SQLite with sqlc (read-only queries)
- [x] Implement golang-migrate for schema versioning
- [x] Create initial migration from existing schema
- [x] Implement WebSocket hub with typed messages and channels
- [x] Implement memory monitoring infrastructure
- [x] Create ML client with request/response support
- [x] Basic endpoints: health, detections, species, system status
- [x] Scaffold Preact + Vite project
- [x] Build Overview page in Preact (hitting Go API)
- [x] Configure Caddy to serve both PHP and Go

**Go Endpoints:**
```
# Public API
GET  /api/health
GET  /api/detections
GET  /api/detections/{date}/{time}/{species}  # Composite key (no auto-increment ID in schema)
GET  /api/species
GET  /api/stats

# System (Part 2 ready)
GET  /api/system/status
GET  /api/system/memory

# Internal (Python → Go)
POST /internal/detection

# WebSocket
WS   /ws
```

**WebSocket Message Types:**
```go
// internal/ws/messages.go

type WSMessage struct {
    Type    string          `json:"type"`
    Channel string          `json:"channel,omitempty"`
    Payload json.RawMessage `json:"payload"`
}

// Part 1 message types
const (
    TypeDetection = "detection"
    TypeStatus    = "status"
)

// Part 2 message types (defined now, used later)
const (
    TypeSpectrogramFrame = "spectrogram_frame"
    TypeLLMStream        = "llm_stream"
    TypeVADResult        = "vad_result"
)
```

**WebSocket Hub with Channels:**
```go
// internal/ws/hub.go

type Hub struct {
    clients     map[*Client]bool
    channels    map[string]map[*Client]bool
    broadcast   chan *Message
    subscribe   chan *Subscription
    unsubscribe chan *Subscription
}

func (h *Hub) Subscribe(client *Client, channel string)
func (h *Hub) Unsubscribe(client *Client, channel string)
func (h *Hub) Broadcast(channel string, msgType string, payload interface{})
func (h *Hub) BroadcastAll(msgType string, payload interface{})
```

**ML Client Interface:**
```go
// internal/mlclient/client.go

type Client struct {
    baseURL    string
    httpClient *http.Client
}

// Part 1 methods
func (c *Client) GetStatus(ctx context.Context) (*Status, error)
func (c *Client) GetHealth(ctx context.Context) error
func (c *Client) GetMemoryUsage(ctx context.Context) (*MemoryStats, error)

// Part 2 methods (interface ready, implementation later)
// func (c *Client) CheckVAD(ctx context.Context, audioPath string) (*VADResult, error)
// func (c *Client) AskLLM(ctx context.Context, req *LLMRequest) (*LLMResponse, error)
```

**Memory Monitor:**
```go
// internal/monitor/memory.go

type MemoryMonitor struct {
    components map[string]func() uint64
}

func (m *MemoryMonitor) Register(name string, getter func() uint64)
func (m *MemoryMonitor) GetUsage() map[string]uint64
func (m *MemoryMonitor) GetTotal() uint64

// Part 2: decision helpers
func (m *MemoryMonitor) ShouldUnloadLLM(threshold uint64) bool
```

**Caddy Configuration:**
```caddyfile
http:// {
  # New Preact frontend (served by Go)
  handle /app/* {
    reverse_proxy localhost:8080
  }

  # New Go API
  handle /api/* {
    reverse_proxy localhost:8080
  }

  # WebSocket
  handle /ws {
    reverse_proxy localhost:8080
  }

  # Existing PHP (keep running)
  handle /* {
    root * /home/{user}/BirdNET-Pi-fork/src/web/public
    php_fastcgi unix//run/php/php-fpm.sock
    file_server
  }

  # Recordings (unchanged)
  handle /By_Date/* {
    root * /home/{user}/BirdSongs/Extracted
    file_server
  }
}
```

**Milestone:** Go server running with Part 2-ready WebSocket and monitoring infrastructure.

---

### Phase 2: Python FastAPI Service + Detection Flow (Week 2) ✅ COMPLETE

**Goal:** Python as full FastAPI service with extensible architecture, real-time detection updates flowing to browser.

**Implementation Summary:**
The FastAPI service was built with a clean router architecture under `src/service/`, with the ModelManager abstract base class providing lifecycle management (load/unload/memory tracking) that Part 2 models (VAD, LLM) will extend. Testing required careful mocking strategy—`sys.modules` patching was needed to avoid importing TensorFlow/TFLite in the test environment, and `requests.Session` had to be mocked at the class level since it's exposed as a property. The detection notification flow integrates cleanly: `reporting.py` calls the notifier after DB writes, which POSTs to Go's `/internal/detection`, triggering WebSocket broadcast to all connected frontends. The Overview page's real-time updates were already functional from Phase 1's WebSocket infrastructure.

**Tasks:**
- [x] Create FastAPI application with router architecture
- [x] Implement ModelManager base class for model lifecycle
- [x] Create BirdNET model manager using ModelManager pattern
- [x] Migrate analysis daemon to use FastAPI service
- [x] Implement notifier module for Go communication
- [x] Add memory reporting to status endpoint
- [x] Create stub routers for Part 2 (VAD, LLM)
- [x] Implement internal endpoint in Go to receive notifications
- [x] Add `useWebSocket` hook to Preact with channel support
- [x] Update Overview page with real-time detection list
- [x] Set up systemd service

**Python Service Structure:**
```python
# src/service/main.py

from fastapi import FastAPI
from contextlib import asynccontextmanager

from .routers import analysis, status, vad, llm
from .models.birdnet import birdnet_manager

@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup: load BirdNET model
    birdnet_manager.load()
    yield
    # Shutdown: cleanup
    birdnet_manager.unload()

app = FastAPI(
    title="BirdNET-Pi ML Service",
    lifespan=lifespan
)

# Part 1 routers
app.include_router(analysis.router, prefix="/analysis", tags=["analysis"])
app.include_router(status.router, prefix="/status", tags=["status"])

# Part 2 routers (stubs, return 501 Not Implemented)
app.include_router(vad.router, prefix="/vad", tags=["vad"])
app.include_router(llm.router, prefix="/llm", tags=["llm"])
```

**Model Manager Base Class:**
```python
# src/service/models/base.py

from abc import ABC, abstractmethod
from typing import Optional
import threading

class ModelManager(ABC):
    """Base class for ML model lifecycle management."""

    def __init__(self):
        self._model = None
        self._lock = threading.Lock()
        self._load_time: Optional[float] = None

    @abstractmethod
    def _load_model(self):
        """Load the model. Implement in subclass."""
        pass

    @abstractmethod
    def _unload_model(self):
        """Unload the model. Implement in subclass."""
        pass

    @abstractmethod
    def memory_usage(self) -> int:
        """Return approximate memory usage in bytes."""
        pass

    def load(self):
        with self._lock:
            if self._model is None:
                self._model = self._load_model()
                self._load_time = time.time()

    def unload(self):
        with self._lock:
            if self._model is not None:
                self._unload_model()
                self._model = None
                self._load_time = None

    def is_loaded(self) -> bool:
        return self._model is not None

    def get_model(self):
        if not self.is_loaded():
            self.load()
        return self._model
```

**BirdNET Model Manager:**
```python
# src/service/models/birdnet.py

from .base import ModelManager
from birdnet.models import get_model

class BirdNETManager(ModelManager):
    def __init__(self):
        super().__init__()
        self._memory_estimate = 500 * 1024 * 1024  # ~500MB

    def _load_model(self):
        return get_model()

    def _unload_model(self):
        self._model = None
        # Force garbage collection
        import gc
        gc.collect()

    def memory_usage(self) -> int:
        if self.is_loaded():
            return self._memory_estimate
        return 0

# Singleton instance
birdnet_manager = BirdNETManager()
```

**Part 2 Stub Routers:**
```python
# src/service/routers/vad.py

from fastapi import APIRouter, HTTPException

router = APIRouter()

@router.post("/check")
async def check_vad(audio_path: str):
    """Check audio file for voice activity. (Part 2)"""
    raise HTTPException(status_code=501, detail="VAD not implemented yet")

@router.get("/status")
async def vad_status():
    """Get VAD model status. (Part 2)"""
    return {"enabled": False, "status": "not_implemented"}
```

```python
# src/service/routers/llm.py

from fastapi import APIRouter, HTTPException

router = APIRouter()

@router.post("/ask")
async def ask_llm(question: str):
    """Ask a question to the LLM. (Part 2)"""
    raise HTTPException(status_code=501, detail="LLM not implemented yet")

@router.get("/status")
async def llm_status():
    """Get LLM model status. (Part 2)"""
    return {"enabled": False, "loaded": False, "status": "not_implemented"}
```

**Status Router with Memory:**
```python
# src/service/routers/status.py

from fastapi import APIRouter
from ..models.birdnet import birdnet_manager

router = APIRouter()

@router.get("/health")
async def health():
    return {"status": "ok"}

@router.get("/status")
async def status():
    return {
        "birdnet": {
            "loaded": birdnet_manager.is_loaded(),
            "memory_bytes": birdnet_manager.memory_usage(),
        },
        "vad": {"enabled": False},
        "llm": {"enabled": False, "loaded": False},
    }

@router.get("/memory")
async def memory():
    return {
        "birdnet": birdnet_manager.memory_usage(),
        "vad": 0,  # Part 2
        "llm": 0,  # Part 2
        "total": birdnet_manager.memory_usage(),
    }
```

**Detection Notifier:**
```python
# src/service/notifier.py

import os
import requests
from typing import Optional
import logging

log = logging.getLogger(__name__)

GO_SERVER_URL = os.environ.get("GO_SERVER_URL", "http://127.0.0.1:8080")

def notify_detection(detection: dict) -> bool:
    """Notify Go server of new detection for WebSocket broadcast."""
    try:
        response = requests.post(
            f"{GO_SERVER_URL}/internal/detection",
            json=detection,
            timeout=1.0
        )
        return response.status_code == 200
    except requests.RequestException as e:
        # Non-fatal: Go might be down, detection is already in DB
        log.warning(f"Failed to notify Go server: {e}")
        return False
```

**Python Service Endpoints:**
```
# Analysis
POST /analysis/file           # Analyze single file
GET  /analysis/queue          # Queue status

# Status
GET  /status/health           # Health check
GET  /status/status           # Full status
GET  /status/memory           # Memory usage by component

# Part 2 stubs (return 501)
POST /vad/check
GET  /vad/status
POST /llm/ask
GET  /llm/status
```

**Detection Flow:**
```
1. Recording daemon writes WAV to StreamData/
2. Python service (inotify) detects new file
3. Python analyzes with BirdNET model
4. Python writes to SQLite
5. Python POSTs to Go: POST /internal/detection
6. Go broadcasts to WebSocket channel "detections"
7. Preact receives via WebSocket, updates UI
```

**Milestone:** Full FastAPI service with extensible architecture, real-time updates working.

#### Post-Phase 2: Legacy Code Removal

After validating Phase 2, the legacy pipeline was deleted to ensure Pi testing uses only the new infrastructure:

**Deleted:**
- `scripts/runtime/birdnet_analysis.py` - legacy analysis daemon (replaced by `src/service/pipeline.py`)
- `scripts/tools/weekly_report.sh` - depended entirely on PHP endpoint

**Simplified:**
- `deployment/birdnet-analysis.service` - now directly runs `python -m service.pipeline`
- `scripts/tools/disk_check.sh` - removed PHP call, kept disk management logic

**Graceful fallback retained:**
- `src/birdnet/notifications.py` - image API has 2s timeout and silent failure (images optional)

---

### Phase 3: Incremental Preact Pages (Weeks 3-4)

**Goal:** Migrate remaining PHP pages to Preact, one at a time.

**Migration Order (by complexity):**

| Week | Page | Complexity | Status | Notes |
|------|------|------------|--------|-------|
| 3.1 | Stats | Low | ✅ Complete | Read-only, species list with sorting, detail modal |
| 3.2 | Today's Detections | Low | ✅ Complete | Search, filters, delete, info links, bird images, history chart |1
| 3.3 | History | Medium | ✅ Complete | Date picker, pagination, reuses DetectionList |
| 3.4 | Species Management | Medium | ✅ Complete | Species table, list editor, toggles, delete |
| 4.1 | Spectrogram | Medium | ✅ Complete | Image updates (Part 2: real-time WebSocket) |
| 4.2 | Settings | High | ✅ Complete | Form validation, schema-driven, service control |
| 4.3 | Advanced Settings | High | ✅ Complete | Schema validation, INI parser, multiple sections |
| 4.4 | Play/Audio | Medium | ✅ Complete | Audio player with Web Audio API, recordings browser |

**Tasks per page:**
- [ ] Create Go API endpoints for page data
- [ ] Build Preact page component
- [ ] Add to Preact router
- [ ] Test alongside PHP equivalent
- [ ] Update Caddy routing when ready

---

#### Phase 3.1 Learnings (Stats Page Migration)

**Completed:** Stats page with species list, sorting, detail modal, audio player

**Backend Changes:**

1. **sqlc Type Overrides Required** - SQLite DATE/TIME columns and aggregate functions (MAX, AVG) return `interface{}` or `time.Time` by default. Added explicit overrides in `sqlc.yaml`:
   ```yaml
   overrides:
     - column: "detections.date"
       go_type: "string"
     - column: "detections.time"
       go_type: "string"
     - db_type: "DATE"
       go_type: "string"
   ```

2. **Aggregate Function Handling** - `MAX(confidence)` returns `interface{}`, requiring helper functions for type assertions:
   ```go
   func toFloat64(v interface{}) float64 {
     switch val := v.(type) {
     case float64: return val
     case int64: return float64(val)
     default: return 0
     }
   }
   ```

3. **Composite Key Queries** - Detection lookups require `(date, time, sci_name)` composite key, not a single ID. Added `GetDetectionByCompositeKey` query.

4. **Sort Parameter API Design** - Species endpoint accepts `?sort=alphabetical|occurrences|confidence|date` with separate sqlc queries per sort order (sqlc doesn't support dynamic ORDER BY).

**Frontend Changes:**

1. **TypeScript Migration** - Converted entire frontend from JSX to TSX for type safety matching Go API:
   - `src/types/api.ts` - All API response types mirroring Go structs
   - Typed hooks: `useApi.ts`, `useWebSocket.ts`
   - Strict mode enabled in `tsconfig.json`

2. **Component Architecture:**
   - `AudioPlayer.tsx` - Reusable audio player with spectrogram overlay
   - `SpeciesDetail.tsx` - Modal component for species detail view
   - Pattern: Fetch on mount, show loading state, handle errors

3. **API Hook Pattern:**
   ```typescript
   export async function fetchSpeciesDetail(name: string): Promise<SpeciesDetail> {
     return apiFetch<SpeciesDetail>(`${API_BASE}/species/${encodeURIComponent(name)}`);
   }
   ```

**Key Files Added:**
- `internal/api/species.go` - Species list and detail endpoints
- `web/src/types/api.ts` - TypeScript API types
- `web/src/components/AudioPlayer.tsx` - Audio player component
- `web/src/components/SpeciesDetail.tsx` - Species detail modal

---

#### Phase 3.2 Learnings (Today's Detections Page Migration)

**Completed:** Full PHP feature parity including search, confidence filter, stats header, delete detection, info links, bird images, species history chart

**Backend Changes:**

1. **sqlc NOT Operator Limitation** - Attempted to add a query with `NOT LIKE` patterns for exclusion search, but sqlc only generated 2 parameters (Date, Confidence) instead of the full set. This appears to be a sqlc bug with complex WHERE clauses. **Workaround:** Implement NOT search in Go code if needed (rare use case).

2. **Search Query Pattern** - Multi-field text search requires repeating the search pattern for each field:
   ```sql
   WHERE (com_name LIKE ? OR sci_name LIKE ? OR file_name LIKE ? OR time LIKE ?)
     AND confidence >= ?
   ```
   The API wraps the search term with `%` wildcards before passing to sqlc.

3. **Filtered Count Query** - Need separate COUNT query with same filters to get accurate total for pagination:
   ```sql
   -- name: CountSearchDetectionsByDate :one
   SELECT COUNT(*) as count FROM detections
   WHERE date = ? AND (com_name LIKE ? OR ...) AND confidence >= ?;
   ```

4. **Species History Date Calculation** - Proper Go time handling for date range:
   ```go
   startDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
   ```

5. **Delete Endpoint** - Uses composite key (date/time/sci_name) in URL path, requires proper URL encoding on frontend.

**Frontend Changes:**

1. **Wikipedia API Integration** - For bird images, the API requires `origin=*` parameter for CORS. Scientific names need underscores:
   ```typescript
   url.searchParams.set('titles', sciName);  // "Turdus migratorius"
   url.searchParams.set('origin', '*');
   ```
   Implemented with in-memory cache to avoid repeated API calls.

2. **Real-time Filter Sync** - WebSocket detection updates must be filtered client-side to match current search/confidence filters. New detections that don't pass filters are silently ignored.

3. **Delete Confirmation UX** - Inline confirmation panel with clear warning message, not a browser `confirm()` dialog.

4. **Info Link URL Patterns:**
   - Wikipedia: `/wiki/{Sci_Name}` with spaces → underscores
   - AllAboutBirds: `/guide/{Com_Name}` with spaces → underscores
   - eBird: `/species/{sci_name}` lowercase, no spaces

5. **Component Architecture:**
   - `StatsHeader.tsx` - Standalone component fetching its own stats
   - `SearchFilters.tsx` - Controlled component with callbacks for filter changes
   - `SpeciesMiniChart.tsx` - Modal with bar chart, configurable date range
   - `BirdImage.tsx` - Async image loader with Wikipedia API and caching

**Key Files Added/Modified:**
- `internal/db/queries.sql` - Search, count, delete, history queries
- `internal/api/detections.go` - Enhanced with search, filters, delete, history
- `internal/api/stats.go` - Added `detections_last_hour`
- `web/src/components/StatsHeader.tsx` - Summary statistics display
- `web/src/components/SearchFilters.tsx` - Search bar and confidence dropdown
- `web/src/components/SpeciesMiniChart.tsx` - Detection history bar chart modal
- `web/src/components/BirdImage.tsx` - Wikipedia image fetcher with cache
- `web/src/components/DetectionList.tsx` - Enhanced with images, links, delete, chart trigger
- `web/src/pages/TodaysDetections.tsx` - Integrated all new features

**API Endpoints Added:**
```
DELETE /api/detections/{date}/{time}/{species}  # Delete by composite key
GET    /api/species/{name}/history?days=30      # Detection history for charts
```

**API Parameters Added:**
```
GET /api/detections?search=robin&min_confidence=0.7  # Text search + confidence filter
GET /api/stats                                        # Now includes detections_last_hour
```

---

#### Phase 3.3 Learnings (History Page Migration)

**Completed:** History page with date picker, date navigation, and full detection browsing for any historical date

**Backend Changes:**

1. **Existing Query Reuse** - The `GetDetectionDates` query already existed in `queries.sql` from earlier planning. Verified it returns dates sorted DESC (most recent first) with LIMIT/OFFSET for pagination.

2. **Simple Dates Endpoint** - New `/api/dates?limit=365` endpoint returns just the list of dates with detections. Response format:
   ```json
   {"dates": ["2026-01-11", "2026-01-10", ...], "total": 45}
   ```

3. **Route Registration Pattern** - Added route alongside existing detection routes in `cmd/server/main.go`:
   ```go
   r.Get("/dates", handlers.ListDates)
   ```

**Frontend Changes:**

1. **DatePicker Component Design** - Created `DatePicker.tsx` with:
   - Native `<input type="date">` for cross-browser compatibility
   - Previous/Next buttons that jump to dates with detections (not just ±1 day)
   - "Today" quick button
   - Quick access chips for 7 most recent dates
   - Status indicator showing if selected date has detections
   - Available dates passed as prop, converted to Set for O(1) lookup

2. **Navigation Logic** - Previous/Next find closest dates with detections:
   ```typescript
   // availableDates sorted DESC, so index+1 is earlier, index-1 is later
   const prevDate = availableDates[currentIndex + 1];  // Earlier date
   const nextDate = availableDates[currentIndex - 1];  // Later date
   ```

3. **Component Reuse Strategy** - History page maximizes reuse:
   - `DetectionList` - Full detection display (audio, spectrogram, images, delete, chart)
   - `SearchFilters` - Search bar and confidence filter
   - Only new component is `DatePicker`

4. **Date Formatting** - Short format for chips, full format for header:
   ```typescript
   // Chip: "Jan 11"
   date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
   // Header: "Saturday, January 11, 2026"
   date.toLocaleDateString('en-US', { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' });
   ```

5. **State Reset on Date Change** - When date changes, reset page to 1:
   ```typescript
   const handleDateChange = useCallback((date: string) => {
     setSelectedDate(date);
     setPage(1);
   }, []);
   ```

**Key Files Added/Modified:**
- `internal/api/detections.go:382-426` - `ListDates` handler and `ListDatesResponse` type
- `cmd/server/main.go:70-71` - Route registration for `/api/dates`
- `web/src/types/api.ts` - `ListDatesResponse` and `ListDatesParams` types
- `web/src/hooks/useApi.ts` - `fetchDates` function
- `web/src/components/DatePicker.tsx` - New date picker component
- `web/src/pages/History.tsx` - New history page
- `web/src/components/Header.tsx` - Added "History" nav link, renamed "Detections" to "Today"
- `web/src/app.tsx` - Added `/app/history` route

**API Endpoint Added:**
```
GET /api/dates?limit=365  # List dates with detections (sorted DESC)
```

---

#### Phase 3.4 Learnings (Species Management Page Migration)

**Completed:** Full species management page with sortable/filterable table, toggle icons for confirmed/excluded/whitelisted status, species list editor modal, and delete all detections functionality

**Backend Changes:**

1. **Species List File Management** - Created `species_lists.go` to manage text-based species list files (confirmed, excluded, whitelisted, include). Each file contains one scientific name per line:
   ```go
   // Read list file, return as string slice
   func (h *Handlers) readSpeciesList(listType string) ([]string, error)
   // Write list file from string slice
   func (h *Handlers) writeSpeciesList(listType string, species []string) error
   ```

2. **File Path Configuration** - Added `scriptsDir` and `dataDir` fields to Handlers struct, configured via `SCRIPTS_DIR` and `DATA_DIR` environment variables. Species list files are stored at `{scriptsDir}/confirmed.txt`, etc.

3. **Delete All Species Detections** - Composite operation that deletes database records AND associated audio/spectrogram files:
   ```go
   // 1. Get file paths for species
   // 2. Delete files from disk (By_Date/{date}/{filename}.{mp3,png})
   // 3. Delete database records
   ```

4. **Labels Endpoint** - New `/api/labels` endpoint reads the BirdNET labels file to provide autocomplete suggestions in the list editor.

**Frontend Changes:**

1. **SpeciesTable Component** - Sortable/filterable table with:
   - Column headers that toggle sort direction (name, scientific name, detections, confidence, last seen)
   - Search filter with debouncing
   - Toggle icons for confirmed (✓), excluded (✗), whitelisted (♡) status
   - Delete button with confirmation
   - localStorage persistence for sort/filter preferences

2. **SpeciesListEditor Modal** - Dual-list UI pattern:
   - Left panel: searchable list of all available species (from labels)
   - Right panel: current list members with remove buttons
   - Add/remove operations with real-time updates
   - Cancel/Save buttons

3. **List Management Cards** - Four cards showing counts for include, exclude, whitelist, and confirmed lists with Edit buttons.

4. **Delete Confirmation Modal** - Shows detection and file counts before confirming destructive operation.

**Key Files Added/Modified:**
- `internal/db/queries.sql:146-162` - `DeleteAllDetectionsForSpecies`, `GetSpeciesFilePaths`, `CountDetectionsBySpecies`, `ListAllSpeciesWithLastSeen` queries
- `internal/api/species_lists.go` - NEW file for species list file CRUD operations
- `internal/api/species.go:157-250` - `GetSpeciesCount`, `DeleteAllSpeciesDetections`, `ListAllSpecies` endpoints
- `cmd/server/main.go` - Added `SCRIPTS_DIR`, `DATA_DIR` env vars and 8 new routes
- `web/src/types/api.ts` - `SpeciesListType`, `SpeciesListsResponse`, `LabelsResponse`, `SpeciesCountResponse`, `DeleteSpeciesResponse` types
- `web/src/hooks/useApi.ts` - `fetchAllSpecies`, `fetchSpeciesCount`, `deleteAllSpeciesDetections`, `fetchSpeciesLists`, `addToSpeciesList`, `removeFromSpeciesList`, `updateSpeciesList`, `fetchLabels` functions
- `web/src/components/SpeciesTable.tsx` - NEW sortable/filterable species table component
- `web/src/components/SpeciesListEditor.tsx` - NEW modal for editing species lists
- `web/src/pages/SpeciesManagement.tsx` - NEW page integrating all species management features
- `web/src/app.tsx` - Added `/app/species` route
- `web/src/components/Header.tsx` - Added "Species" nav link

**API Endpoints Added:**
```
GET    /api/species/all                      # List all detected species with counts
GET    /api/species/{name}/count             # Get detection/file counts for species
DELETE /api/species/{name}/all               # Delete all detections for species
GET    /api/species-lists                    # Get all species lists
PUT    /api/species-lists/{listType}         # Replace entire list
POST   /api/species-lists/{listType}/add     # Add species to list
POST   /api/species-lists/{listType}/remove  # Remove species from list
GET    /api/labels                           # Get BirdNET model labels for autocomplete
```

---

### Phase 4: Live/Spectrogram Page

#### Phase 4.1 Learnings (Spectrogram Page Migration - Part 1)

**Completed:** Static spectrogram page with auto-refreshing image, live audio stream player, recent detections feed with WebSocket updates, and connection status indicator.

**PHP Analysis Findings:**

The original PHP spectrogram page (`src/web/app/pages/spectrogram.php`) had two modes:
1. **Legacy mode**: Static PNG image refresh for older/mobile browsers
2. **Canvas mode**: Real-time Web Audio API visualization with frequency analysis

For Part 1, we implemented the simpler static image approach with:
- Sox-generated spectrogram.png served via Go endpoint
- Icecast2 livestream audio playback
- WebSocket-based detection notifications

**Backend Changes:**

1. **Spectrogram Info Endpoint** - Returns metadata about spectrogram availability:
   ```go
   GET /api/spectrogram/info
   Response: {
     image_url: string,
     last_modified: string,
     available: boolean,
     livestream_url: string,
     refresh_seconds: number
   }
   ```

2. **Spectrogram Image Endpoint** - Serves the spectrogram PNG with cache-control headers:
   ```go
   GET /api/spectrogram/image
   // Serves {dataDir}/extracted/spectrogram.png
   // Headers: Cache-Control: no-cache, no-store, must-revalidate
   ```

3. **Recent Detections Endpoint** - Returns latest detections for sidebar feed:
   ```go
   GET /api/spectrogram/detections?limit=10
   Response: {
     detections: RecentDetection[],
     total: number
   }
   ```

**Frontend Changes:**

1. **Spectrogram Page Component** (`web/src/pages/Spectrogram.tsx`):
   - Auto-refreshing spectrogram image (configurable interval)
   - HTML5 audio player for Icecast livestream
   - Recent detections sidebar with WebSocket real-time updates
   - Connection status indicator (Live/Offline)
   - Graceful handling of missing spectrogram image

2. **Auto-refresh Pattern** - Uses `setInterval` with cache-busting URL:
   ```typescript
   useEffect(() => {
     const refreshMs = (info.refresh_seconds || 3) * 1000;
     const interval = setInterval(() => {
       setImageUrl(getSpectrogramImageUrl()); // Adds ?t=Date.now()
     }, refreshMs);
     return () => clearInterval(interval);
   }, [info]);
   ```

3. **WebSocket Integration** - Reuses existing hook pattern:
   ```typescript
   subscribe<DetectionNotification>('detection', (payload) => {
     setRecentDetections(prev => [detection, ...prev].slice(0, 10));
   });
   ```

**Key Files Added/Modified:**
- `internal/api/spectrogram.go` - NEW file with 3 endpoints
- `cmd/server/main.go:95-98` - Added spectrogram routes
- `web/src/types/api.ts:308-334` - `SpectrogramInfoResponse`, `RecentDetection`, `RecentDetectionsResponse` types
- `web/src/hooks/useApi.ts:368-399` - `fetchSpectrogramInfo`, `getSpectrogramImageUrl`, `fetchRecentDetections` functions
- `web/src/pages/Spectrogram.tsx` - NEW page component
- `web/src/app.tsx` - Added `/app/live` route
- `web/src/components/Header.tsx` - Added "Live" nav link

**API Endpoints Added:**
```
GET  /api/spectrogram/info        # Spectrogram metadata and livestream URL
GET  /api/spectrogram/image       # Serve spectrogram PNG with no-cache headers
GET  /api/spectrogram/detections  # Recent detections for sidebar feed
```

**Part 2 Preparation:**
The WebSocket infrastructure already supports typed messages for future real-time spectrogram streaming (Web Audio API visualization). The `spectrogram_frame` message type is defined in `types/api.ts`.

---

#### Phase 4.2 & 4.3 Learnings (Settings & Advanced Settings Migration)

**Completed:** Full settings system with Go config package, API endpoints, and Preact pages mirroring PHP config.php, advanced.php, and service_controls.php

**Backend Changes:**

1. **Config Package Architecture** - Created `internal/config/` package with 4 files:
   - `types.go` - Config struct with 60+ fields matching all PHP parameters
   - `ini.go` - INI file parser/writer using regex replacement (mirrors PHP's `preg_replace`)
   - `schema.go` - JSON Schema validation rules from existing `birdnet.schema.json`
   - `config.go` - Config loader with defaults, Manager struct for operations

2. **INI File Handling** - PHP used regex replacement to preserve comments in config files. Go implementation mirrors this:
   ```go
   pattern := regexp.MustCompile(`(?m)^` + key + `=.*$`)
   content = pattern.ReplaceAllString(content, key+"="+value)
   ```

3. **Service Restart Logic** - Tracks which config changes require service restarts:
   ```go
   var configFieldToService = map[string]string{
       "RECORDING_LENGTH": "birdnet-recording",
       "RTSP_STREAM":      "birdnet-recording",
       "ICE_PWD":          "icecast2",
       // ... 15+ mappings
   }
   ```

4. **Systemd Service Control** - 8 managed services with status checking:
   ```go
   var managedServices = []string{
       "birdnet-recording", "birdnet-analysis", "birdnet-stats",
       "icecast2", "birdnet-server", "caddy", "php-fpm", "avahi-daemon",
   }
   ```

5. **Stall Detection** - Recording backlog check to detect stalled services:
   ```go
   func getRecordingBacklog() int {
       // Count WAV files in StreamData older than 2 minutes
   }
   ```

**Frontend Changes:**

1. **Form Input Components** (`web/src/components/settings/FormInputs.tsx`):
   - TextInput, NumberInput, SliderInput, SelectInput
   - TextAreaInput, ToggleInput, CheckboxInput
   - FormSection, SaveButton, AlertMessage
   - All with consistent Tailwind styling and dark mode support

2. **useSettings Hook** - Comprehensive state management:
   ```typescript
   const { config, loading, saving, error, validationErrors,
           restartedServices, refresh, save, clearErrors } = useSettings();
   ```

3. **useServices Hook** - Service management with polling:
   ```typescript
   const { services, actionLoading, performAction, restartAll } = useServices(true, 10000);
   ```

4. **Settings Page Structure**:
   - Basic Settings (`/app/settings`): Location, Model, Analysis, Recording, Integrations, Notifications, Display
   - Advanced Settings (`/app/advanced-settings`): Privacy, Disk, Hardware, Passwords, URLs, Frequency Shifting, Time, Logging
   - Service Controls (`/app/services`): Status badges, action buttons, restart all

5. **Header Navigation** - Settings dropdown with gear icon:
   ```typescript
   const settingsLinks = [
     { href: '/app/settings', label: 'Settings' },
     { href: '/app/advanced-settings', label: 'Advanced' },
     { href: '/app/services', label: 'Services' },
   ];
   ```

**Key Files Added:**
- `internal/config/types.go` - Config struct with 60+ fields
- `internal/config/ini.go` - INI parser/writer with regex replacement
- `internal/config/schema.go` - JSON Schema validation
- `internal/config/config.go` - Manager and loader
- `internal/api/settings.go` - GET/PUT settings, schema endpoint
- `internal/api/services.go` - Service control endpoints
- `web/src/types/settings.ts` - TypeScript types matching Go structs
- `web/src/hooks/useSettings.ts` - Settings and services hooks
- `web/src/components/settings/FormInputs.tsx` - Reusable form components
- `web/src/pages/Settings.tsx` - Basic settings page
- `web/src/pages/AdvancedSettings.tsx` - Advanced settings page
- `web/src/components/ServiceControls.tsx` - Service management component

**API Endpoints Added:**
```
GET  /api/settings              # Get all configuration
PUT  /api/settings              # Update configuration (partial updates supported)
GET  /api/settings/schema       # Get validation schema for form generation
GET  /api/services              # List service statuses
POST /api/services/restart-all  # Restart all managed services
POST /api/services/{name}/{action}  # start/stop/restart/enable/disable
```

---

#### Phase 4.4 Learnings (Play/Audio Page Migration)

**Completed:** Full recordings browser with navigation hierarchy, enhanced audio player with Web Audio API, and file management actions (delete, change ID, lock, shift)

**Backend Changes:**

1. **Recordings API** - Created `internal/api/recordings.go` with comprehensive file management:
   - `ListRecordingDates` - Dates with recordings on disk
   - `ListRecordingSpecies` - Species list with sorting (alphabetical, occurrences, confidence, date)
   - `ListRecordingsByDate` - Species detected on a specific date
   - `ListRecordingsBySpecies` - Audio files for a species with pagination
   - `DeleteRecording` - Delete audio file and database record
   - `ChangeRecordingIdentification` - Move file to different species folder, update database
   - `ToggleRecordingLock` - Add/remove from `disk_check_exclude.txt` exclusion list
   - `ToggleRecordingShift` - Create/remove frequency-shifted audio version

2. **SQLite DATE Format Issue** - go-sqlite3 returns DATE columns as ISO timestamps (`2026-01-13T00:00:00Z`). Fixed by extracting date portion:
   ```go
   dateStr := d.Date
   if idx := strings.Index(dateStr, "T"); idx > 0 {
       dateStr = dateStr[:idx]
   }
   ```

3. **File Path Construction** - Audio files stored at `~/BirdSongs/Extracted/By_Date/{date}/{Species_Name}/{filename}.mp3` with corresponding `.png` spectrogram.

**Frontend Changes:**

1. **EnhancedAudioPlayer Component** (`web/src/components/EnhancedAudioPlayer.tsx`):
   - Web Audio API integration with AudioContext, GainNode, BiquadFilter
   - Gain control: Off, 6dB, 12dB, 18dB, 24dB, 30dB
   - Highpass filter: Off, 250Hz, 500Hz, 1000Hz, 1500Hz
   - Lowpass filter: Off, 2000Hz, 4000Hz, 8000Hz
   - LocalStorage persistence for audio preferences
   - Spectrogram image with progress overlay and playhead
   - Menu dropdown with Info, Download, and action buttons

2. **Recordings Page** (`web/src/pages/Recordings.tsx`):
   - Navigation hierarchy: Choose view → By Species/By Date → Species/Date list → Audio files
   - Sort controls for species list and file list
   - "Locked only" filter for protected files
   - Modal for changing species identification with searchable labels
   - Pagination for large file lists

3. **Header Navigation** - Added "Play" link to main navigation bar

**Key Files Added:**
- `internal/api/recordings.go` - 9 API handlers for recordings management
- `internal/db/queries.sql` - Added ListDetectionsBySpecies, ListDetectionsBySpeciesAndDate queries
- `web/src/components/EnhancedAudioPlayer.tsx` - Advanced audio player with Web Audio API
- `web/src/pages/Recordings.tsx` - Recordings browser page
- `web/src/types/api.ts` - RecordingFile, ListRecordingFilesResponse, etc.
- `web/src/hooks/useApi.ts` - fetchRecordingDates, fetchRecordingsBySpecies, etc.

**API Endpoints Added:**
```
GET  /api/recordings/dates                              # Dates with recordings
GET  /api/recordings/species?sort=occurrences           # Species with recording counts
GET  /api/recordings/by-date/{date}                     # Species for a date
GET  /api/recordings/by-species/{name}?date=&sort=&only_locked=&page=&limit=
POST /api/recordings/{date}/{species}/{filename}/delete
POST /api/recordings/{date}/{species}/{filename}/change  # Body: {new_species: string}
POST /api/recordings/{date}/{species}/{filename}/lock    # Toggle exclusion list
POST /api/recordings/{date}/{species}/{filename}/shift   # Toggle frequency shift
GET  /api/recordings/exclusions                          # List locked files
```

---

**Go Endpoints Added:**
```
# Charts (Part 2 interactive graphs ready)
GET  /api/charts/timeline?start=&end=&species=
GET  /api/charts/frequency?start=&end=
GET  /api/charts/heatmap?start=&end=
GET  /api/charts/confidence?species=

# Stats
GET  /api/stats/daily
GET  /api/stats/weekly
GET  /api/stats/species-counts

# History
GET  /api/detections/by-date/:date
GET  /api/dates

# Settings
GET  /api/settings
PUT  /api/settings
GET  /api/settings/schema

# Services
GET  /api/services
POST /api/services/:name/restart
POST /api/services/:name/stop
POST /api/services/:name/start

# Audio
GET  /api/audio/dates
GET  /api/audio/files/:date
```

**Preact WebSocket Hook with Channels:**
```javascript
// web/src/hooks/useWebSocket.js

import { useState, useEffect, useCallback, useRef } from 'preact/hooks';

export function useWebSocket(url) {
  const [isConnected, setIsConnected] = useState(false);
  const [lastMessage, setLastMessage] = useState(null);
  const ws = useRef(null);
  const subscribers = useRef(new Map());

  useEffect(() => {
    ws.current = new WebSocket(url);

    ws.current.onopen = () => setIsConnected(true);
    ws.current.onclose = () => setIsConnected(false);

    ws.current.onmessage = (event) => {
      const message = JSON.parse(event.data);
      setLastMessage(message);

      // Notify channel subscribers
      const channelSubs = subscribers.current.get(message.type) || [];
      channelSubs.forEach(callback => callback(message.payload));
    };

    return () => ws.current?.close();
  }, [url]);

  const subscribe = useCallback((messageType, callback) => {
    if (!subscribers.current.has(messageType)) {
      subscribers.current.set(messageType, []);
    }
    subscribers.current.get(messageType).push(callback);

    // Return unsubscribe function
    return () => {
      const subs = subscribers.current.get(messageType);
      const index = subs.indexOf(callback);
      if (index > -1) subs.splice(index, 1);
    };
  }, []);

  return { isConnected, lastMessage, subscribe };
}
```

**Milestone:** All pages migrated to Preact, PHP still available as fallback.

---

### Phase 4: Go Scheduler + Runtime Scripts (Week 5)

**Goal:** Replace continuously-running shell cron jobs with Go scheduler.

**Scripts to Migrate to Go:**

| Script | New Location | Trigger |
|--------|--------------|---------|
| `cleanup.sh` | `internal/scheduler/cleanup.go` | Every 3 minutes |
| `disk_check.sh` | `internal/scheduler/disk.go` | Every 5 minutes |
| `disk_species_clean.sh` | `internal/scheduler/retention.go` | Daily 2 AM |

**Scripts to KEEP in Shell:**

| Script | Reason |
|--------|--------|
| `install_*.sh` | Run once during installation |
| `uninstall.sh` | Rarely used |
| `backup_data.sh` | Interactive/manual operation |
| `update_birdnet.sh` | Complex git operations |
| `createdb.sh` | Installation only |
| `dump_logs.sh` | Diagnostic tool |
| `print_diagnostic_info.sh` | Diagnostic tool |

**Tasks:**
- [ ] Implement scheduler package with cron-like timing
- [ ] Migrate cleanup job
- [ ] Migrate disk check job
- [ ] Migrate retention job
- [ ] Add scheduler endpoints for manual triggers
- [ ] Test scheduler reliability over 24h
- [ ] Remove migrated cron entries

**Scheduler API:**
```
GET  /api/scheduler/jobs
POST /api/scheduler/jobs/:name/run
GET  /api/scheduler/jobs/:name/log
```

**Milestone:** Go handles all periodic runtime tasks.

---

### Phase 5: Configuration Migration (Week 6)

**Goal:** Unified YAML configuration with validation, Part 2 sections included.

**Tasks:**
- [ ] Create `config.yaml` format with Part 2 sections
- [ ] Create `config.schema.yaml` with full validation
- [ ] Implement Go config loader with schema validation
- [ ] Update Python to read YAML
- [ ] Add config reload endpoint
- [ ] Build Settings page with schema-driven forms
- [ ] Write migration script for existing installations

**Configuration Structure (Part 2 Ready):**
```yaml
# config.yaml

# === Part 1 Configuration ===

server:
  host: 0.0.0.0
  port: 8080

location:
  latitude: 42.3601
  longitude: -71.0589

analysis:
  model: BirdNET_GLOBAL_6K_V2.4_Model_FP16
  confidence: 0.7          # min: 0.01, max: 0.99
  sensitivity: 1.25        # min: 0.5, max: 1.5
  overlap: 0.0             # min: 0.0, max: 2.9
  privacy_threshold: 0     # 0=off, 1-3=increasing

recording:
  device: default
  length: 15               # seconds
  channels: 2

extraction:
  length: 6                # seconds
  format: mp3              # mp3, wav, flac, ogg, opus

storage:
  recordings_path: ~/BirdSongs
  retention_days: 30
  max_files_per_species: 0 # 0 = unlimited
  purge_threshold: 95      # disk % to trigger purge

notifications:
  apprise:
    enabled: false
    title: "New BirdNET-Pi Detection"
    notify_each_detection: false
    notify_new_species: false
    notify_weekly_report: true
  birdweather:
    id: ""

interface:
  site_name: "BirdNET-Pi"
  color_scheme: light      # light, dark
  image_provider: WIKIPEDIA # WIKIPEDIA, FLICKR
  info_site: ALLABOUTBIRDS  # ALLABOUTBIRDS, EBIRD

# === Part 2 Configuration (disabled by default) ===

vad:
  enabled: false
  threshold: 0.5           # min: 0.0, max: 1.0
  action: skip             # skip, flag, log_only
  min_speech_duration: 0.5 # seconds

llm:
  enabled: false
  model: tinyllama-1.1b    # tinyllama-1.1b, qwen2.5-0.5b, phi-3-mini
  model_path: ""           # Custom path, or auto-download
  idle_timeout: 300        # seconds before unloading
  max_tokens: 512
  temperature: 0.7

spectrogram:
  enabled: false
  fps: 10
  websocket_channel: spectrogram

charts:
  default_range_days: 7
  max_range_days: 365
  heatmap_enabled: true
```

**Configuration Schema:**
```yaml
# config.schema.yaml

type: object
required:
  - location
  - analysis
properties:
  server:
    type: object
    properties:
      host:
        type: string
        default: "0.0.0.0"
      port:
        type: integer
        default: 8080
        minimum: 1
        maximum: 65535

  location:
    type: object
    required:
      - latitude
      - longitude
    properties:
      latitude:
        type: number
        minimum: -90
        maximum: 90
      longitude:
        type: number
        minimum: -180
        maximum: 180

  analysis:
    type: object
    properties:
      model:
        type: string
        enum:
          - BirdNET_GLOBAL_6K_V2.4_Model_FP16
          - BirdNET_6K_GLOBAL_MODEL
        default: BirdNET_GLOBAL_6K_V2.4_Model_FP16
      confidence:
        type: number
        minimum: 0.01
        maximum: 0.99
        default: 0.7
      sensitivity:
        type: number
        minimum: 0.5
        maximum: 1.5
        default: 1.25

  # Part 2 schemas
  vad:
    type: object
    properties:
      enabled:
        type: boolean
        default: false
      threshold:
        type: number
        minimum: 0.0
        maximum: 1.0
        default: 0.5
      action:
        type: string
        enum: [skip, flag, log_only]
        default: skip

  llm:
    type: object
    properties:
      enabled:
        type: boolean
        default: false
      model:
        type: string
        enum: [tinyllama-1.1b, qwen2.5-0.5b, phi-3-mini]
        default: tinyllama-1.1b
      idle_timeout:
        type: integer
        minimum: 60
        maximum: 3600
        default: 300
```

**Milestone:** Clean YAML configuration with Part 2 sections ready.

---

### Phase 6: PHP Decommission + Cleanup (Week 7)

**Goal:** Remove PHP, finalize architecture.

**Tasks:**
- [ ] Run both systems in parallel for 1 week
- [ ] Validate all Preact pages against PHP equivalents
- [ ] Update Caddy to remove PHP routes
- [ ] Stop php-fpm service
- [ ] Archive `src/web/` PHP code
- [ ] Update systemd units
- [ ] Update installation scripts
- [ ] Update documentation
- [ ] Set up backup strategy

**Final Caddy Configuration:**
```caddyfile
http:// {
  # Go serves everything
  reverse_proxy localhost:8080

  # Direct file serving for recordings (optional optimization)
  handle /recordings/* {
    root * /home/{user}/BirdSongs/Extracted
    file_server
  }
}
```

**Milestone:** PHP fully retired, Part 2-ready architecture complete.

---

## Systemd Services

### Go Server
```ini
# /etc/systemd/system/birdnet-server.service
[Unit]
Description=BirdNET Go Server
After=network.target birdnet-ml.service

[Service]
Type=simple
ExecStart=/opt/birdnet/server
WorkingDirectory=/opt/birdnet
Restart=always
RestartSec=5
User=birdnet
Environment=BIRDNET_CONFIG=/etc/birdnet/config.yaml

[Install]
WantedBy=multi-user.target
```

### Python ML Service
```ini
# /etc/systemd/system/birdnet-ml.service
[Unit]
Description=BirdNET ML Service
After=network.target

[Service]
Type=simple
ExecStart=/opt/birdnet/venv/bin/uvicorn service.main:app --host 127.0.0.1 --port 8001
WorkingDirectory=/opt/birdnet/src
Restart=always
RestartSec=5
User=birdnet
Environment=BIRDNET_CONFIG=/etc/birdnet/config.yaml
Environment=GO_SERVER_URL=http://127.0.0.1:8080

[Install]
WantedBy=multi-user.target
```

### Recording Daemon (unchanged)
```ini
# /etc/systemd/system/birdnet-recording.service
# Keep existing shell-based recording daemon
```

---

## Database Schema Versioning

Using golang-migrate for schema management:

```
migrations/
├── 000001_initial_schema.up.sql
├── 000001_initial_schema.down.sql
├── 000002_add_indexes.up.sql
├── 000002_add_indexes.down.sql
└── ...
```

### Initial Migration (000001_initial_schema.up.sql)
```sql
-- Matches actual Pi database schema (capitalized column names, no auto-increment ID)
CREATE TABLE IF NOT EXISTS detections (
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

CREATE INDEX IF NOT EXISTS idx_detections_date_time ON detections(Date DESC, Time DESC);
CREATE INDEX IF NOT EXISTS idx_detections_sci_name ON detections(Sci_Name);
CREATE INDEX IF NOT EXISTS idx_detections_com_name ON detections(Com_Name);
CREATE INDEX IF NOT EXISTS idx_detections_confidence ON detections(Confidence);

-- Schema version tracking (golang-migrate handles this)
```

### Part 2 Migration (000003_add_vad.up.sql)
```sql
-- Added when VAD feature is implemented
ALTER TABLE detections ADD COLUMN vad_score REAL;
ALTER TABLE detections ADD COLUMN vad_skipped BOOLEAN DEFAULT FALSE;

CREATE INDEX idx_detections_vad_skipped ON detections(vad_skipped) WHERE vad_skipped = TRUE;
```

---

## Resource Budget

### Part 1 (Infrastructure)

| Component | RAM | Notes |
|-----------|-----|-------|
| Go server | ~20MB | Includes WebSocket connections |
| Python ML service | ~500MB | BirdNET model loaded |
| SQLite | ~50MB | Shared cache |
| Recording daemon | ~20MB | ffmpeg/arecord |
| System | ~500MB | |
| **Total** | ~1.1GB | |

### Part 2 (All Features Enabled)

| Component | RAM | Notes |
|-----------|-----|-------|
| Part 1 infrastructure | ~1.1GB | |
| Silero VAD | ~100MB | Always loaded when enabled |
| LLM (TinyLlama) | ~1GB | Lazy loaded, unloads after idle |
| Spectrogram buffer | ~50MB | When streaming enabled |
| **Peak Total** | ~2.3GB | LLM loaded |
| **Typical** | ~1.3GB | LLM unloaded |

Comfortable within 4GB, with headroom for OS and buffers.

---

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Data loss during migration | Full backup before each phase, parallel running |
| Performance regression | Benchmark critical paths before/after |
| Detection notification failure | Non-fatal: detection in DB, WebSocket is optional |
| Go learning curve | Start with simple endpoints, iterate |
| Breaking existing installations | Incremental migration, backward-compatible config |
| PHP/Preact feature mismatch | Side-by-side testing before PHP removal |
| Part 2 memory pressure | ModelManager with lazy load/unload, monitoring |

---

## Success Criteria

### Part 1 Complete
- [ ] All existing functionality preserved
- [ ] Real-time detection updates via WebSocket
- [ ] Single YAML configuration file with validation
- [ ] No PHP in runtime
- [ ] Shell scripts retained for installation/maintenance only
- [ ] Sub-second page loads
- [ ] Database schema versioned with migrations
- [ ] Clean codebase, documented

### Part 2 Ready
- [ ] Python service has extensible router architecture
- [ ] ModelManager base class implemented
- [ ] Part 2 stub endpoints return 501
- [ ] WebSocket hub supports typed messages and channels
- [ ] Memory monitoring infrastructure in place
- [ ] Config schema includes Part 2 sections (disabled)
- [ ] ML client supports request/response pattern

---

## Timeline Summary

| Phase | Duration | Key Deliverable |
|-------|----------|-----------------|
| 1. Go API + Preact Shell | Week 1 | Go running with Part 2-ready infrastructure |
| 2. Python FastAPI Service | Week 2 | Extensible ML service, real-time updates |
| 3. Preact Pages | Weeks 3-4 | All pages migrated incrementally |
| 4. Go Scheduler | Week 5 | Runtime jobs in Go |
| 5. Configuration | Week 6 | YAML config with Part 2 sections |
| 6. PHP Decommission | Week 7 | Clean final architecture |

**Total: 7 weeks**

---

## Architecture Diagram (Part 2 Ready)

```
┌─────────────────────────────────────────────────────────────────┐
│                     Preact Frontend                              │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐               │
│  │  Detection  │ │ Interactive │ │   LLM UI    │  (Part 2)     │
│  │    List     │ │   Charts    │ │             │               │
│  └──────┬──────┘ └──────┬──────┘ └──────┬──────┘               │
└─────────┼───────────────┼───────────────┼───────────────────────┘
          │               │               │
          │ WebSocket     │ HTTP          │ HTTP
          │ (typed msgs)  │               │
┌─────────▼───────────────▼───────────────▼───────────────────────┐
│                      Go Backend                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  /api/detections   /api/charts/*   /api/llm/*  (proxy)   │   │
│  │  /api/species      /api/system/*   /api/vad/*  (proxy)   │   │
│  └──────────────────────────────────────────────────────────┘   │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │  WebSocket  │  │  MLClient   │  │  Memory     │              │
│  │  Hub        │  │ (req/resp)  │  │  Monitor    │              │
│  │  (channels) │  │             │  │             │              │
│  └──────┬──────┘  └──────┬──────┘  └─────────────┘              │
└─────────┼────────────────┼──────────────────────────────────────┘
          │                │
          │ notifications  │ HTTP (request/response)
          │                │
┌─────────▼────────────────▼──────────────────────────────────────┐
│                   Python ML Service (FastAPI)                    │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  Routers:  /analysis/*   /status/*   /vad/*    /llm/*      │ │
│  │             (Part 1)      (Part 1)   (stub)    (stub)      │ │
│  └────────────────────────────────────────────────────────────┘ │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐                 │
│  │  BirdNET   │  │  Silero    │  │  TinyLlama │                 │
│  │  Manager   │  │  VAD Mgr   │  │  LLM Mgr   │                 │
│  │  (loaded)  │  │  (Part 2)  │  │  (Part 2)  │                 │
│  └────────────┘  └────────────┘  └────────────┘                 │
│         │                                                        │
│         └─── All extend ModelManager base class                  │
└──────────────────────────────────────────────────────────────────┘
          │
          │ Direct write
          ▼
┌──────────────────┐
│     SQLite       │
│  (versioned)     │
└──────────────────┘
```

---

## Quick Reference: What's Part 2 Ready

| Component | Part 1 Implementation | Part 2 Extension Point |
|-----------|----------------------|------------------------|
| WebSocket | Detection notifications | Typed messages, channels for spectrogram/LLM |
| Python Service | BirdNET analysis | Router architecture for VAD/LLM routers |
| Model Loading | BirdNETManager | ModelManager base class for VAD/LLM |
| Memory | Basic monitoring | Per-component tracking, unload decisions |
| Config | Core settings | VAD/LLM/spectrogram sections (disabled) |
| Database | Base schema + migrations | Migration system for VAD columns |
| Go ML Client | Status checks | Request/response for VAD/LLM calls |
