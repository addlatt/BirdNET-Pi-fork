# PHP Deprecation Plan

## Goal

Remove all PHP dependencies from BirdNET-Pi, leaving only:
- **Go** - API server, WebSocket, task scheduling
- **Python** - ML inference, services (spectrogram, livestream)
- **Preact** - Frontend UI

## Current PHP Footprint

23 PHP files remain (9,513 lines). All have Go or Preact replacements -- these are legacy files awaiting Phase 4 deletion.

```
src/web/
├── public/
│   └── index.php              # Front controller (90 lines)
├── app/
│   ├── bootstrap.php          # Config loader (197 lines)
│   ├── router.php             # View router, AJAX handlers (546 lines)
│   ├── lib/
│   │   └── common.php         # DB functions, image providers (555 lines) → replaced by internal/images/
│   └── pages/
│       ├── overview.php       # (618 lines) → Overview.tsx
│       ├── todays_detections.php  # (568 lines) → Detections.tsx
│       ├── spectrogram.php    # (546 lines) → Spectrogram.tsx
│       ├── stats.php          # (226 lines) → Stats.tsx
│       ├── species_tools.php  # (402 lines) → SpeciesManagement.tsx
│       ├── species_list.php   # (125 lines) → SpeciesManagement.tsx
│       ├── play.php           # (709 lines) → Recordings.tsx
│       ├── history.php        # (167 lines) → Stats.tsx (charts)
│       ├── weekly_report.php  # (203 lines) → /api/reports/weekly
│       ├── config.php         # (696 lines) → Settings.tsx
│       ├── advanced.php       # (745 lines) → AdvancedSettings.tsx
│       ├── system_controls.php # (123 lines) → /api/system/*
│       ├── service_controls.php # (104 lines) → ServiceControls.tsx
│       ├── backup.php         # (21 lines) → Backup.tsx
│       ├── restore.php        # (97 lines) → /api/backup/restore
│       └── api.php            # (51 lines) → /api/species/{name}/image
└── vendor/
    └── adminer/               # Database admin (2,724 lines, 3 files) -- keeping until Phase 4
```

**Already removed:**
- `data/ebird.php` -- converted to `data/ebird.json` (Phase 1)
- `src/web/vendor/filemanager/` -- replaced by Recordings.tsx (Phase 1)

## Go API Coverage

The Go backend now has **68 registered API routes** across 21 handler files (16,772 lines in `internal/`). Every PHP page has a corresponding Go API and/or Preact replacement:

| PHP File | Go/Preact Replacement | Status |
|----------|----------------------|--------|
| overview.php | Overview.tsx + /api/detections, /api/species/ranking | ✅ Replaced |
| todays_detections.php | Detections.tsx + /api/detections | ✅ Replaced |
| stats.php | Stats.tsx + /api/stats | ✅ Replaced |
| spectrogram.php | Spectrogram.tsx + /api/spectrogram/* | ✅ Replaced |
| species_list.php | SpeciesManagement.tsx + /api/species-lists | ✅ Replaced |
| species_tools.php | SpeciesManagement.tsx + /api/reclassify | ✅ Replaced |
| play.php | Recordings.tsx + /api/recordings/* | ✅ Replaced |
| config.php | Settings.tsx + /api/settings | ✅ Replaced |
| advanced.php | AdvancedSettings.tsx + /api/settings | ✅ Replaced |
| service_controls.php | ServiceControls.tsx + /api/services/* | ✅ Replaced |
| history.php | Stats.tsx (charts) + /api/stats | ✅ Replaced |
| weekly_report.php | /api/reports/weekly + /api/reports/weekly/export | ✅ API Ready |
| system_controls.php | /api/system/update-check, reboot, shutdown | ✅ API Ready |
| backup.php | Backup.tsx + /api/backup/create | ✅ Replaced |
| restore.php | Backup.tsx + /api/backup/restore, /api/backup/status | ✅ Replaced |
| api.php (image API) | /api/species/{name}/image + 3 more endpoints | ✅ Replaced |
| common.php (image classes) | internal/images/* (6 Go files) | ✅ Replaced |
| bootstrap.php | internal/config/ | ✅ Replaced |
| router.php | chi router in cmd/server/main.go | ✅ Replaced |
| index.php | Static file server in cmd/server/main.go | ✅ Replaced |
| adminer/ (3 files) | No replacement planned (remove in Phase 4) | ⏳ Phase 4 |

**Coverage: 100% of PHP functionality has Go/Preact replacements. Only Phase 4 deletion remains.**

---

## Gaps -- All Closed

### Gap 1: Weekly Report & eBird Export ✅ CLOSED

**PHP Files:** `weekly_report.php`, `data/ebird.php`

**Status:** Go endpoints implemented in Phase 1. PHP files still exist but fully replaced.

**Implemented Go Endpoints:**
```
GET /api/reports/weekly
    ?week=2024-W05           # ISO week format (optional, defaults to current)
    Response: { week, start_date, end_date, total_detections, unique_species,
                comparison: { prev_total, change_pct },
                top_species: [{ sci_name, com_name, count, change_pct }],
                new_species: [{ sci_name, com_name, count, first_detected }] }

GET /api/reports/weekly/export
    ?format=csv|ebird&week=2024-W05
    Response: text/csv download with species codes
```

**Files Created:**
- `internal/api/reports.go` - Report generation + export endpoints
- `data/ebird.json` - 6,523 eBird species codes (converted from PHP)

---

### Gap 2: System Controls ⚡ PARTIALLY CLOSED

**PHP File:** `system_controls.php`

**Status:** Core endpoints implemented in Phase 1. Update/clear data endpoints deferred.

**Implemented Go Endpoints:**
```
GET /api/system/update-check    - Check git for updates
POST /api/system/reboot         - Reboot (requires {"confirm": true})
POST /api/system/shutdown       - Shutdown (requires {"confirm": true})
```

**Remaining Endpoints (deferred to Phase 4):**
```
POST /api/system/update         - git pull && rebuild (streamed via WebSocket)
POST /api/system/clear-data     - clear_all_data.sh (requires confirmation)
```

---

### Gap 3: Backup System ✅ CLOSED

**PHP Files:** `backup.php`, `restore.php`

**Status:** Go endpoints implemented in Phase 2. Backup.tsx frontend added.

**Implemented Go Endpoints:**
```
POST /api/backup/create    - Stream backup as tar.gz download
POST /api/backup/restore   - Upload and restore backup file (atomic, two-pass)
GET  /api/backup/status    - Get restore progress by ID (+ WebSocket broadcasts)
```

---

### Gap 4: Image Provider Caching ✅ CLOSED

**PHP File:** `common.php` (Flickr + Wikipedia classes), `api.php` (image endpoint)

**Status:** Go implementation complete in Phase 3. Full Flickr + Wikipedia support with SQLite caching, blacklist management, and cache expiration.

**Implemented Go Endpoints:**
```
GET  /api/species/{name}/image           - Fetch species image (cached or fresh)
    Query: ?provider=flickr|wikipedia|auto&com_name=Common+Name
    Response: { sci_name, com_name, provider, image_url, title, source_id,
                author_url, license_url, photos_url, cached_at }

POST /api/species/{name}/image/blacklist - Blacklist image + get replacement
    Body: { provider, com_name }

GET  /api/images/cache/stats             - Cache statistics
    Response: { flickr_count, wikipedia_count, total_count, expired_count }

POST /api/images/cache/refresh           - Refresh expired images (background)
```

---

### Gap 5: Vendor Tool Replacements

#### Adminer (Database Admin) - REMOVING IN PHASE 4

**Current:** PHP-based SQLite browser at /adminer (3 files, 2,724 lines)

**Decision:** Remove entirely. Users can use CLI `sqlite3` or external tools.

#### File Manager ✅ REMOVED (Phase 1)

**Replacement:** Recordings.tsx + /api/recordings endpoints

#### phpsysinfo ✅ REMOVED (Phase 1)

**Replacement:** /api/diagnostics/system + /api/system/status endpoints

---

## Migration Phases

### Phase 1: Quick Wins ✅ COMPLETE

**Goal:** Close easy gaps, remove unused vendor tools

**Status:** Completed 2026-02-03 | Branch: `php-deprecation-phase1`

**Tasks:**
- [x] Add `GET /api/reports/weekly` endpoint
- [x] Add `GET /api/reports/weekly/export` endpoint (CSV + eBird format)
- [x] Add `GET /api/system/update-check` endpoint
- [x] Add `POST /api/system/reboot` endpoint (with confirmation)
- [x] Add `POST /api/system/shutdown` endpoint (with confirmation)
- [x] Convert `data/ebird.php` to `data/ebird.json`
- [x] Remove filemanager from Caddyfile
- [x] Remove phpsysinfo from Caddyfile and install script
- [x] Update `internal/config/caddy.go` template (remove filemanager/phpsysinfo)
- [x] Update `src/web/app/lib/common.php` to use ebird.json
- [x] Update `src/web/app/router.php` with deprecation messages

**Files Created:**
- `internal/api/reports.go` - Weekly report + export endpoints
- `data/ebird.json` - 6,523 eBird species codes

**Files Modified:**
- `internal/api/system.go` - Added update-check, reboot, shutdown handlers
- `cmd/server/main.go` - Registered 5 new routes
- `deployment/Caddyfile` - Removed filemanager/phpsysinfo routes
- `scripts/install/install_services.sh` - Removed phpsysinfo function and refs
- `internal/config/caddy.go` - Removed deprecated routes from template
- `src/web/app/lib/common.php` - Updated eBird lookup to use JSON
- `src/web/app/router.php` - Added deprecation messages for removed tools

**Files Deleted:**
- `data/ebird.php`
- `src/web/vendor/filemanager/` (4 files)

**New API Endpoints:**
```
GET  /api/system/update-check    - Check git for updates
POST /api/system/reboot          - Reboot (requires {"confirm": true})
POST /api/system/shutdown        - Shutdown (requires {"confirm": true})
GET  /api/reports/weekly         - Weekly detection report
GET  /api/reports/weekly/export  - CSV export (?format=csv|ebird)
```

**Effort:** 1 day (actual)

---

### Phase 2: Backup System ✅ COMPLETE

**Goal:** Full backup/restore without PHP

**Status:** Completed 2026-02-03 | Backend API complete, Backup.tsx added

**Tasks:**
- [x] Add `POST /api/backup/create` - streaming tar.gz directly to response
- [x] Add `POST /api/backup/restore` - multipart upload with atomic extraction
- [x] Add `GET /api/backup/status` - progress tracking with WebSocket broadcasts
- [x] Path traversal protection (validates destination within allowed directories)
- [x] Symlink/hardlink rejection (security)
- [x] Critical file fail-fast (birdnet.conf, birds.db must succeed)
- [x] Two-pass restore (count files first for accurate progress)
- [x] Atomic restore (extract to temp dir, then move to final location)
- [x] Restore state cleanup (TTL-based, prevents memory leak)
- [x] Add BackupRestore.tsx component (Backup.tsx created)

**Files Created:**
- `internal/api/backup.go` - ~550 lines, full backup/restore implementation

**Files Modified:**
- `internal/api/handlers.go` - Added homeDir field to Handlers struct
- `cmd/server/main.go` - Registered 3 new routes, passed homeDir to NewHandlers
- `internal/ws/messages.go` - Added TypeRestoreProgress constant

**New API Endpoints:**
```
POST /api/backup/create     - Stream backup as tar.gz download
POST /api/backup/restore    - Upload and restore backup file
GET  /api/backup/status     - Get restore progress by ID
```

**Security Features:**
- Path traversal blocked via absolute path validation
- Symlinks and hardlinks rejected
- 500MB upload limit
- Critical files fail-fast on extraction error

**Effort:** 3-5 days estimated, 1 day actual

---

### Phase 3: Image Provider Enhancement ✅ COMPLETE

**Goal:** Full Flickr/Wikipedia caching in Go

**Status:** Completed 2026-02-10

**Tasks:**
- [x] Implement Flickr API client in Go
- [x] Implement Wikipedia API client in Go
- [x] Add image cache SQLite table (data/db/images.db)
- [x] Add cache expiration logic (20-day fixed expiration)
- [x] Add blacklist management (DB table + in-memory set)
- [x] Add image service orchestrator with provider selection
- [x] Add 4 HTTP API endpoints
- [x] Update BirdImage.tsx to use server-side API
- [x] Add frontend TypeScript types and API functions

**Files Created:**
- `internal/images/types.go` - Shared types: ImageResult, CacheStats, provider constants
- `internal/images/cache.go` - SQLite cache (data/db/images.db), blacklist, expiration
- `internal/images/wikipedia.go` - Wikipedia REST + Commons API client
- `internal/images/flickr.go` - Flickr REST API client (search, info, licenses, user lookup)
- `internal/images/service.go` - Orchestrator: provider selection, cache-then-fetch
- `internal/api/images.go` - HTTP handlers for 4 image endpoints

**Files Modified:**
- `internal/api/handlers.go` - Added imageService field + SetImageService()
- `cmd/server/main.go` - Initialize image service, register 4 routes, close on shutdown
- `web/src/types/api.ts` - Added SpeciesImageResponse, BlacklistImageResponse, ImageCacheStatsResponse
- `web/src/hooks/useApi.ts` - Added fetchSpeciesImage(), blacklistSpeciesImage()
- `web/src/components/BirdImage.tsx` - Replaced client-side Wikipedia fetch with /api/species/{name}/image

**New API Endpoints:**
```
GET  /api/species/{name}/image          - Fetch species image (cached or fresh)
POST /api/species/{name}/image/blacklist - Blacklist image + get replacement
GET  /api/images/cache/stats            - Cache statistics
POST /api/images/cache/refresh          - Refresh expired cache entries
```

**Key Design Decisions:**
- Single `images.db` with provider column (simpler than PHP's two-DB approach)
- Fixed 20-day cache expiration (vs PHP's random 15-25 days)
- Blacklist stored in DB table (migrated from txt file on first run)
- Flickr gracefully returns nil when API key is empty

**Effort:** 2-3 days estimated, 1 day actual

---

### Phase 4: Final Cleanup ✅ COMPLETE

**Goal:** Remove all PHP files and dependencies

**Status:** Completed 2026-02-10

**Tasks:**
- [x] Relocate fonts from `src/web/public/assets/fonts/` to `data/fonts/`
- [x] Update `notifications.py` to use Go image API instead of PHP endpoint
- [x] Remove `/legacy*` route from Caddyfile
- [x] Remove `php_fastcgi` directive from Caddyfile
- [x] Remove `php-fpm` from install_services.sh
- [x] Remove `php-*` packages from apt install
- [x] Delete `src/web/app/` directory (20 files)
- [x] Delete `src/web/public/` directory (fonts moved, rest deleted)
- [x] Delete `src/web/vendor/adminer/` (3 files)
- [x] Delete orphaned templates (phpsysinfo.ini, index_bootstrap.html)
- [x] Rewrite `internal/config/caddy.go` template (PHP → Preact SPA)
- [x] Rewrite `install_services.sh` Caddyfile generation
- [x] Remove `configure_caddy_php()` function
- [x] Update `uninstall.sh` (remove PHP filter)
- [x] Update `update_birdnet_snippets.sh` (remove php7.4 migration)
- [x] Update README.md, CLAUDE.md, deployment/README.md
- [x] Update LICENSE (remove phpSysInfo GPL, remove Adminer from Apache)
- [x] Clean up PHP references in code comments

**Files Deleted:**
```
src/web/                # Entire directory removed (app/, public/, vendor/)
templates/phpsysinfo.ini
templates/index_bootstrap.html
```

**Files Moved:**
```
src/web/public/assets/fonts/*.ttf → data/fonts/
```

**Effort:** 1 day (actual)

---

## Success Criteria

- [x] All Go API endpoints implemented for every PHP page
- [x] Backup/restore works via Go API
- [x] Image provider caching works via Go API (Flickr + Wikipedia)
- [x] BirdImage.tsx uses server-side API instead of client-side Wikipedia
- [x] All Preact pages functional without /legacy fallback
- [x] No PHP files remain in codebase
- [x] No PHP dependencies in install scripts
- [x] No php_fastcgi directives in Caddyfile or caddy.go

---

## Timeline

| Phase | Estimated | Actual | Status |
|-------|-----------|--------|--------|
| Phase 1: Quick Wins | 2-3 days | 1 day | ✅ Complete (2026-02-03) |
| Phase 2: Backup System | 3-5 days | 1 day | ✅ Complete (2026-02-03) |
| Phase 3: Image Provider | 2-3 days | 1 day | ✅ Complete (2026-02-10) |
| Phase 4: Final Cleanup | 1 day | 1 day | ✅ Complete (2026-02-10) |

**All phases complete. BirdNET-Pi now runs on Go + Preact + Python only, with no PHP dependency.**
