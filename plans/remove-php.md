# PHP Deprecation Plan

## Goal

Remove all PHP dependencies from BirdNET-Pi, leaving only:
- **Go** - API server, WebSocket, task scheduling
- **Python** - ML inference, services (spectrogram, livestream)
- **Preact** - Frontend UI

## Current PHP Footprint

```
src/web/
├── public/
│   └── index.php              # Front controller
├── app/
│   ├── bootstrap.php          # Config loader
│   ├── router.php             # View router, AJAX handlers
│   ├── lib/
│   │   └── common.php         # DB functions, image providers (updated: uses ebird.json)
│   └── pages/
│       ├── overview.php       # Dashboard
│       ├── todays_detections.php
│       ├── spectrogram.php
│       ├── stats.php
│       ├── species_tools.php
│       ├── species_list.php
│       ├── play.php           # Recordings
│       ├── history.php        # Daily charts
│       ├── weekly_report.php
│       ├── config.php         # Settings
│       ├── advanced.php
│       ├── system_controls.php
│       ├── service_controls.php
│       ├── backup.php
│       ├── restore.php
│       └── api.php            # Image API
└── vendor/
    ├── adminer/               # Database admin (keeping until Phase 4)
    └── filemanager/           # REMOVED in Phase 1

data/
└── ebird.php                  # REMOVED - converted to ebird.json

Total: 21 PHP files remaining, ~4,900 lines (was 25 files, ~10,400 lines)
```

## Already Migrated

| PHP File | Replacement | Status |
|----------|-------------|--------|
| overview.php | Overview.tsx + /api/detections | ✓ Complete |
| todays_detections.php | Detections.tsx | ✓ Complete |
| stats.php | Stats.tsx + /api/species | ✓ Complete |
| spectrogram.php | Spectrogram.tsx + /api/spectrogram | ✓ Complete |
| species_list.php | SpeciesManagement.tsx + /api/species-lists | ✓ Complete |
| species_tools.php | SpeciesManagement.tsx + /api/reclassify | ✓ Complete |
| play.php | Recordings.tsx + /api/recordings | ✓ Complete |
| config.php | Settings.tsx + /api/settings | ✓ Complete |
| advanced.php | AdvancedSettings.tsx | ✓ Complete |
| service_controls.php | ServiceControls.tsx + /api/services | ✓ Complete |
| history.php | Stats.tsx (charts) | ✓ Complete |
| backup.php | /api/backup/create | ✓ API Complete |
| restore.php | /api/backup/restore + /api/backup/status | ✓ API Complete |
| weekly_report.php | /api/reports/weekly | ✓ API Complete |

**Coverage: ~85% of UI functionality migrated (APIs complete, backup frontend added, weekly report pending)**

---

## Gaps to Close

### Gap 1: Weekly Report & eBird Export ✅ CLOSED

**PHP Files:** `weekly_report.php`, `data/ebird.php`

**Status:** Go endpoints implemented in Phase 1. PHP `weekly_report.php` still exists but can be removed once frontend component is added.

**Implemented Go Endpoints:**
```
GET /api/reports/weekly
    ?week=2024-W05           # ISO week format (optional, defaults to current)
    Response: {
      week: "2024-W05",
      start_date: "2024-01-28",
      end_date: "2024-02-03",
      total_detections: int,
      unique_species: int,
      comparison: { prev_total, change_pct },
      top_species: [{ sci_name, com_name, count, change_pct }],
      new_species: [{ sci_name, com_name, count, first_detected }]
    }

GET /api/reports/weekly/export
    ?format=csv|ebird&week=2024-W05
    Response: text/csv download with species codes
```

**Files Created:**
- `internal/api/reports.go` - Report generation + export endpoints
- `data/ebird.json` - 6,523 eBird species codes (converted from PHP)

**Remaining:** Add WeeklyReport.tsx component to replace PHP page

---

### Gap 2: System Controls ⚡ PARTIALLY CLOSED

**PHP File:** `system_controls.php`

**Status:** Core endpoints implemented in Phase 1. Update/clear data endpoints deferred.

**Implemented Go Endpoints:**
```
GET /api/system/update-check
    Response: {
      current_commit: string,
      latest_commit: string,
      behind_count: int,
      update_available: bool
    }

POST /api/system/reboot
    Body: { "confirm": true }
    Response: { "status": "rebooting" }

POST /api/system/shutdown
    Body: { "confirm": true }
    Response: { "status": "shutting_down" }
```

**Remaining Endpoints (deferred):**
```
POST /api/system/update
    Action: git pull && rebuild
    Response: streamed output via WebSocket

POST /api/system/clear-data
    Action: clear_all_data.sh
    Requires: confirmation + careful handling
```

**Security:** Confirmation required via `{"confirm": true}` in request body

**Effort:** LOW for remaining (update script execution)

---

### Gap 3: Backup System ✅ CLOSED

**PHP Files:** `backup.php`, `restore.php`

**Status:** Go endpoints implemented in Phase 2. PHP files still exist but can be removed once frontend component is added.

**Implemented Go Endpoints:**
```
POST /api/backup/create
    Response: application/gzip stream (streamed directly, no temp file)
    Headers: Content-Disposition: attachment; filename="birdnet-backup_YYYY-MM-DD.tar.gz"

POST /api/backup/restore
    Content-Type: multipart/form-data
    Body: backup field with .tar.gz file (max 500MB)
    Response: { restore_id: string, status: "started" }

    Features:
    - Two-pass processing (count files, then extract)
    - Atomic restore (extract to temp dir, then move)
    - WebSocket progress broadcasts on "tasks" channel
    - Path traversal protection
    - Symlink/hardlink rejection
    - Critical file fail-fast (birdnet.conf, birds.db)

GET /api/backup/status?id={restore_id}
    Response: {
      id: string,
      status: "uploading" | "extracting" | "completed" | "failed",
      progress: 0-100,
      stage: string,
      error: string (if failed),
      started_at: ISO8601
    }
```

**Files Created:**
- `internal/api/backup.go` - Backup download, restore upload, progress tracking

**Files Modified:**
- `internal/api/handlers.go` - Added homeDir field to Handlers struct
- `cmd/server/main.go` - Registered 3 new routes, passed homeDir
- `internal/ws/messages.go` - Added TypeRestoreProgress constant

**Remaining:** Add BackupRestore.tsx component to replace PHP pages

**Effort:** HIGH (actual: 1 day)

---

### Gap 4: Image Provider Caching ✅ CLOSED

**PHP File:** `common.php` (getFlickrImage, getWikipediaImage functions)

**Status:** Go implementation complete in Phase 3. Full Flickr + Wikipedia support with SQLite caching, blacklist management, and cache expiration.

**Implemented Go Endpoints:**
```
GET /api/species/{name}/image
    Query: ?provider=flickr|wikipedia|auto&com_name=Common+Name
    Response: { sci_name, com_name, provider, image_url, title, source_id, author_url, license_url, photos_url, cached_at }

POST /api/species/{name}/image/blacklist
    Body: { provider, com_name }
    Action: Mark current image as bad, fetch new one

GET /api/images/cache/stats
    Response: { flickr_count, wikipedia_count, total_count, expired_count }

POST /api/images/cache/refresh
    Action: Refresh expired images (background)
```

**Effort:** MEDIUM (actual: Phase 3)

---

### Gap 5: Vendor Tool Replacements

#### Adminer (Database Admin) - KEEPING FOR NOW

**Current:** PHP-based SQLite browser at /adminer

**Options:**
1. Keep Adminer (requires PHP) - simplest
2. Replace with sqlite-web (Python) - adds Python dep
3. Add basic DB viewer to Preact - most work
4. Remove entirely - users can use CLI sqlite3

**Decision:** Keep until Phase 4 (final PHP removal)

#### File Manager ✅ REMOVED

**Status:** Removed in Phase 1

**Changes Made:**
- Deleted `src/web/vendor/filemanager/` directory
- Removed `/filemanager/*` route from `deployment/Caddyfile`
- Removed from `internal/config/caddy.go` template
- Removed from `scripts/install/install_services.sh` Caddyfile generation
- Added deprecation message in `src/web/app/router.php`

**Replacement:** Recordings.tsx + /api/recordings endpoints

#### phpsysinfo ✅ REMOVED

**Status:** Removed in Phase 1

**Changes Made:**
- Removed `/phpsysinfo/*` route from `deployment/Caddyfile`
- Removed from `internal/config/caddy.go` template
- Removed `install_phpsysinfo()` function from install script
- Removed phpsysinfo symlink creation from install script
- Added deprecation message in `src/web/app/router.php`

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
- [ ] Add WeeklyReport.tsx component (optional - deferred)

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
- [x] Add image cache SQLite table
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

**Effort:** 2-3 days estimated

---

### Phase 4: Final Cleanup

**Goal:** Remove all PHP

**Tasks:**
- [ ] Remove `/legacy*` route from Caddyfile
- [ ] Remove `php-fpm` from install_services.sh
- [ ] Remove `php-*` packages from apt install
- [ ] Delete `src/web/app/` directory
- [ ] Delete `src/web/public/` directory
- [ ] Delete `src/web/vendor/adminer/` (last vendor tool)
- [ ] Update README.md
- [ ] Test fresh install without PHP

**Files to Delete:**
```
src/web/app/           # ~3,500 lines
src/web/public/        # ~100 lines
src/web/vendor/adminer/ # ~500 lines (filemanager already removed)
```

**Already Removed in Phase 1:**
```
data/ebird.php         # Converted to ebird.json
src/web/vendor/filemanager/  # Replaced by Recordings.tsx
```

**Effort:** 1 day

---

## Dependencies to Remove

After full migration, these can be removed from `install_services.sh`:

```bash
# Remove from apt install:
php-sqlite3
php-fpm
php-curl
php-xml
php-zip
php

# Remove from Caddyfile:
php_fastcgi unix//run/php/php-fpm.sock
```

**Disk savings:** ~50MB of PHP packages

---

## Risk Assessment

| Risk | Mitigation |
|------|------------|
| Backup restore fails | Test extensively before removing PHP |
| Missing edge case in migration | Keep /legacy route until Phase 4 |
| User has custom PHP modifications | Document in release notes |
| Adminer users lose DB access | Document sqlite3 CLI usage |

---

## Success Criteria

- [ ] All Preact pages functional without /legacy fallback
- [x] Backup/restore works via Go API
- [ ] Fresh install completes without PHP packages
- [ ] All systemd services start correctly
- [ ] No PHP processes running after install

---

## Timeline Estimate

| Phase | Estimated | Actual | Status |
|-------|-----------|--------|--------|
| Phase 1 | 2-3 days | 1 day | ✅ Complete |
| Phase 2 | 3-5 days | 1 day | ✅ Complete |
| Phase 3 | 2-3 days | 1 day | ✅ Complete |
| Phase 4 | 1 day | - | Pending |

**Remaining: ~1 day of development (Phase 4 only)**
