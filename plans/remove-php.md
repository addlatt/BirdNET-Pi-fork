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
│   │   └── common.php         # DB functions, image providers
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
    ├── adminer/               # Database admin
    └── filemanager/           # File browser

data/
└── ebird.php                  # eBird species codes (~5000 lines)

Total: 25 PHP files, ~5,400 lines
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

**Coverage: ~70% of UI functionality migrated**

---

## Gaps to Close

### Gap 1: Weekly Report & eBird Export

**PHP Files:** `weekly_report.php`, `data/ebird.php`

**Current Functionality:**
- Generate weekly detection summary
- Compare current week vs previous week
- Export to eBird CSV format (species code mapping)
- Hourly detection aggregation

**Required Go Endpoints:**
```
GET /api/reports/weekly
    ?week=2024-W05           # ISO week format
    Response: {
      week: string,
      total_detections: int,
      unique_species: int,
      hourly_breakdown: [],
      top_species: [],
      comparison: { prev_week_total, change_pct }
    }

GET /api/reports/weekly/export
    ?format=ebird|csv
    Response: text/csv download
```

**Required Files:**
- `internal/api/reports.go` - Report generation endpoints
- `data/ebird.json` - Convert ebird.php to JSON (or embed in Go)

**Effort:** LOW - Straightforward query aggregation

---

### Gap 2: System Controls

**PHP File:** `system_controls.php`

**Current Functionality:**
- Reboot system (`sudo reboot`)
- Shutdown system (`sudo shutdown -h now`)
- Check for updates (`git fetch && git status`)
- Run system update (`update_birdnet.sh`)
- Clear all data (`clear_all_data.sh`)

**Required Go Endpoints:**
```
POST /api/system/reboot
    Requires: confirmation token
    Action: sudo reboot

POST /api/system/shutdown
    Requires: confirmation token
    Action: sudo shutdown -h now

GET /api/system/update-check
    Response: {
      current_commit: string,
      latest_commit: string,
      behind_count: int,
      update_available: bool
    }

POST /api/system/update
    Action: git pull && rebuild
    Response: streamed output via WebSocket
```

**Security Considerations:**
- Require authentication for all endpoints
- Add confirmation tokens for destructive actions
- Rate limit reboot/shutdown

**Effort:** MEDIUM - Need careful sudo handling

---

### Gap 3: Backup System

**PHP Files:** `backup.php`, `restore.php`

**Current Functionality:**
- Create tar.gz backup of:
  - ~/BirdSongs/ (recordings)
  - ~/BirdNET-Pi/data/db/birds.db
  - ~/BirdNET-Pi/birdnet.conf
  - Species list files
- Stream download to browser
- Chunked upload for restore (named pipes)
- Progress tracking during restore

**Required Go Endpoints:**
```
POST /api/backup/create
    Response: application/gzip stream
    Headers: Content-Disposition: attachment

POST /api/backup/restore
    Content-Type: multipart/form-data
    Body: backup.tar.gz (chunked upload)
    Response: WebSocket for progress updates

GET /api/backup/status
    Response: { in_progress: bool, percent: int, stage: string }
```

**Implementation Notes:**
- Use `archive/tar` and `compress/gzip` Go stdlib
- Stream directly to response (don't buffer in memory)
- For restore: write to temp file, then extract
- Consider using WebSocket for progress updates

**Effort:** HIGH - Complex streaming and progress tracking

---

### Gap 4: Image Provider Caching

**PHP File:** `common.php` (getFlickrImage, getWikipediaImage functions)

**Current Functionality:**
- Flickr API search with caching in flickr.db
- Wikipedia API image lookup with caching in wikipedia.db
- Image expiration (15-25 days)
- Blacklist management for bad images
- License URL extraction

**Current Go Status:**
- Basic `/api/species/{name}/image` exists
- May not have full caching logic

**Required Enhancements:**
```
GET /api/species/{name}/image
    Query: ?provider=flickr|wikipedia|auto
    Response: { url, license_url, attribution, cached_at }

POST /api/species/{name}/image/blacklist
    Action: Mark current image as bad, fetch new one

GET /api/images/cache/stats
    Response: { flickr_count, wikipedia_count, expired_count }

POST /api/images/cache/refresh
    Action: Refresh expired images
```

**Effort:** MEDIUM - API calls + caching logic

---

### Gap 5: Vendor Tool Replacements

#### Adminer (Database Admin)

**Current:** PHP-based SQLite browser at /adminer

**Options:**
1. Keep Adminer (requires PHP) - simplest
2. Replace with sqlite-web (Python) - adds Python dep
3. Add basic DB viewer to Preact - most work
4. Remove entirely - users can use CLI sqlite3

**Recommendation:** Remove from default install, document CLI usage

#### File Manager

**Current:** PHP file browser at /filemanager

**Status:** Already replaced by Recordings.tsx

**Action:** Remove vendor/filemanager/

#### phpsysinfo

**Current:** PHP system info at /phpsysinfo

**Status:** Replaced by /api/diagnostics/system

**Action:** Remove from Caddyfile, remove phpsysinfo install

---

## Migration Phases

### Phase 1: Quick Wins

**Goal:** Close easy gaps, remove unused vendor tools

**Tasks:**
- [ ] Add `GET /api/reports/weekly` endpoint
- [ ] Add `GET /api/system/update-check` endpoint
- [ ] Add `POST /api/system/reboot` endpoint (with confirmation)
- [ ] Add `POST /api/system/shutdown` endpoint (with confirmation)
- [ ] Convert `data/ebird.php` to `data/ebird.json`
- [ ] Remove filemanager from Caddyfile
- [ ] Remove phpsysinfo from Caddyfile and install script
- [ ] Add WeeklyReport.tsx component (optional)

**Files to Create:**
- `internal/api/reports.go`
- `internal/api/system_control.go`
- `data/ebird.json`

**Files to Modify:**
- `deployment/Caddyfile` - remove PHP routes
- `scripts/install/install_services.sh` - remove phpsysinfo

**Effort:** 2-3 days

---

### Phase 2: Backup System

**Goal:** Full backup/restore without PHP

**Tasks:**
- [ ] Add `POST /api/backup/create` - streaming tar.gz
- [ ] Add `POST /api/backup/restore` - chunked upload
- [ ] Add `GET /api/backup/status` - progress tracking
- [ ] Add BackupRestore.tsx component
- [ ] WebSocket progress updates during restore

**Files to Create:**
- `internal/api/backup.go`
- `web/src/pages/BackupRestore.tsx`

**Effort:** 3-5 days

---

### Phase 3: Image Provider Enhancement

**Goal:** Full Flickr/Wikipedia caching in Go

**Tasks:**
- [ ] Implement Flickr API client in Go
- [ ] Implement Wikipedia API client in Go
- [ ] Add image cache SQLite table
- [ ] Add cache expiration logic
- [ ] Add blacklist management

**Files to Create:**
- `internal/images/flickr.go`
- `internal/images/wikipedia.go`
- `internal/images/cache.go`

**Effort:** 2-3 days

---

### Phase 4: Final Cleanup

**Goal:** Remove all PHP

**Tasks:**
- [ ] Remove `/legacy*` route from Caddyfile
- [ ] Remove `php-fpm` from install_services.sh
- [ ] Remove `php-*` packages from apt install
- [ ] Delete `src/web/app/` directory
- [ ] Delete `src/web/public/` directory
- [ ] Delete `src/web/vendor/` directory
- [ ] Delete `data/ebird.php`
- [ ] Update README.md
- [ ] Test fresh install without PHP

**Files to Delete:**
```
src/web/app/           # ~3,500 lines
src/web/public/        # ~100 lines
src/web/vendor/        # ~1,500 lines
data/ebird.php         # ~5,000 lines
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
- [ ] Backup/restore works via Go API
- [ ] Fresh install completes without PHP packages
- [ ] All systemd services start correctly
- [ ] No PHP processes running after install

---

## Timeline Estimate

| Phase | Duration | Cumulative |
|-------|----------|------------|
| Phase 1 | 2-3 days | 2-3 days |
| Phase 2 | 3-5 days | 5-8 days |
| Phase 3 | 2-3 days | 7-11 days |
| Phase 4 | 1 day | 8-12 days |

**Total: ~2 weeks of development**
