# Settings Page Migration Research

> **STATUS: COMPLETED** — Kept for historical reference.

## Executive Summary

This document captures research findings for migrating the PHP settings pages to Preact (Phase 4.2 and Phase 5 of the infrastructure upgrade). The migration involves two main PHP pages with ~70KB of combined code handling 60+ configuration parameters.

---

## PHP Settings Architecture

### Source Files

| File | Size | Purpose |
|------|------|---------|
| `src/web/app/pages/config.php` | 31KB | Basic settings (location, model, notifications, UI) |
| `src/web/app/pages/advanced.php` | 40KB | Advanced settings (audio, storage, accessibility) |
| `src/web/app/pages/service_controls.php` | 7KB | Start/stop/restart services |
| `src/web/app/pages/system_controls.php` | 5KB | Reboot, backup, restore, update |
| `src/web/app/bootstrap.php` | 6KB | Config loading, schema validation |
| `data/config_schema.json` | 10KB | JSON Schema for all 60+ parameters |

### Form Submission Pattern

PHP uses **GET-based form submission**:
```php
// config.php
if(isset($_GET["latitude"])) {
    // Validate all inputs against schema
    // Update /etc/birdnet/birdnet.conf via regex
    // Execute shell commands (service restarts)
    // Reload config cache
}
```

**Why this matters:** We need to use proper REST patterns (GET for fetch, PUT for update) in Go.

### Configuration Storage

**Primary file:** `/etc/birdnet/birdnet.conf` (INI-style)
```ini
SITE_NAME="My Bird Station"
LATITUDE=39.4836
LONGITUDE=-74.0101
MODEL=BirdNET_GLOBAL_6K_V2.4_Model_FP16
CONFIDENCE=0.7
```

**Persistence mechanism:**
```php
$contents = file_get_contents("/etc/birdnet/birdnet.conf");
$contents = preg_replace("/LATITUDE=.*/", "LATITUDE=$latitude", $contents);
fwrite($fh, $contents);
```

**Why this matters:** Go needs a config file reader/writer that preserves comments and ordering.

---

## Configuration Parameters (Complete List)

### Basic Settings (config.php)

| Parameter | Type | Range/Enum | Default | UI Control |
|-----------|------|------------|---------|------------|
| `LATITUDE` | number | -90 to 90 | required | Number input (4 decimal places) |
| `LONGITUDE` | number | -180 to 180 | required | Number input (4 decimal places) |
| `SITE_NAME` | string | - | "" | Text input |
| `MODEL` | enum | FP16, FP32 | FP16 | Radio buttons |
| `SF_THRESH` | number | 0.0005-0.99 | 0.03 | Number input |
| `BIRDWEATHER_ID` | string | - | "" | Text input |
| `DATABASE_LANG` | enum | 40+ languages | "en" | Dropdown |
| `COLOR_SCHEME` | enum | light, dark | light | Toggle switch |
| `IMAGE_PROVIDER` | enum | WIKIPEDIA, FLICKR, "" | WIKIPEDIA | Radio buttons |
| `FLICKR_API_KEY` | string | 32 chars | "" | Text input (conditional) |
| `FLICKR_FILTER_EMAIL` | email | - | "" | Email input (conditional) |
| `INFO_SITE` | enum | ALLABOUTBIRDS, EBIRD | ALLABOUTBIRDS | Radio buttons |

**Notifications section:**
| Parameter | Type | Default | Notes |
|-----------|------|---------|-------|
| `APPRISE_INPUT` | textarea | "" | Multi-line notification URLs |
| `APPRISE_NOTIFICATION_TITLE` | string | "New Detection" | |
| `APPRISE_NOTIFICATION_BODY` | textarea | template | Supports placeholders |
| `MINIMUM_TIME_LIMIT` | number | 0 | Seconds between notifications |
| `ONLY_NOTIFY_SPECIES_NAMES` | string | "" | Comma-separated filter |
| `APPRISE_NOTIFY_EACH_DETECTION` | bool | false | |
| `APPRISE_NOTIFY_NEW_SPECIES` | bool | false | |
| `APPRISE_NOTIFY_NEW_SPECIES_EACH_DAY` | bool | false | |
| `APPRISE_WEEKLY_REPORT` | bool | false | |

### Advanced Settings (advanced.php)

| Parameter | Type | Range | Default | UI Control |
|-----------|------|-------|---------|------------|
| `PRIVACY_THRESHOLD` | int | 0-3 | 0 | Slider |
| `FULL_DISK` | enum | purge, keep | purge | Radio buttons |
| `PURGE_THRESHOLD` | int | 20-99 | 95 | Number input (%) |
| `MAX_FILES_SPECIES` | int | 0+ | 0 | Number (0=unlimited) |
| `REC_CARD` | string | - | "default" | Text/dropdown |
| `CHANNELS` | int | 1-32 | 2 | Number input |
| `RECORDING_LENGTH` | int | 3-60 | 15 | Number (seconds) |
| `EXTRACTION_LENGTH` | int | 3-60 | 6 | Number (seconds) |
| `AUDIOFMT` | enum | mp3,wav,flac,ogg,opus | mp3 | Dropdown |
| `RTSP_STREAM` | string[] | - | [] | Dynamic URL list |
| `CADDY_PWD` | password | alphanumeric | "" | Password input |
| `BIRDNETPI_URL` | url | - | "" | URL input |
| `CONFIDENCE` | number | 0.01-0.99 | 0.7 | Slider |
| `SENSITIVITY` | number | 0.5-1.5 | 1.25 | Slider |
| `OVERLAP` | number | 0.0-2.9 | 0.0 | Slider |

**Frequency Shifting (accessibility):**
| Parameter | Type | Range | Default |
|-----------|------|-------|---------|
| `FREQSHIFT_TOOL` | enum | sox, ffmpeg | sox |
| `FREQSHIFT_HI` | int | 1000-20000 | 12000 |
| `FREQSHIFT_LO` | int | 100-10000 | 1000 |
| `FREQSHIFT_PITCH` | int | -4000 to 4000 | -2000 |
| `FREQSHIFT_RECONNECT_DELAY` | int | ms | 5000 |

**Logging:**
| Parameter | Type | Options |
|-----------|------|---------|
| `LogLevel_BirdnetRecordingService` | enum | debug, info, warning, error |
| `LogLevel_LiveAudioStreamService` | enum | debug, info, warning, error |
| `LogLevel_SpectrogramViewerService` | enum | debug, info, warning, error |

---

## Service Control Integration

### Managed Services

| Service | Actions | Notes |
|---------|---------|-------|
| `birdnet_analysis.service` | stop, start, restart | Core analysis daemon |
| `birdnet_recording.service` | stop, start, restart | Audio recording |
| `livestream.service` | stop, start, restart, enable, disable | Icecast livestream |
| `spectrogram_viewer.service` | stop, start, restart | Spectrogram generation |
| `chart_viewer.service` | stop, start, restart | Chart generation |
| `birdnet_stats.service` | stop, start, restart | Statistics |
| `web_terminal.service` | stop, start, restart | ttyd web terminal |
| `birdnet_log.service` | stop, start, restart | Log viewer |

### PHP Service Control Implementation

```php
function service_status($name) {
    $op = shell_exec("sudo systemctl status ".$name." | grep Active");
    // Returns: "active (running)" | "inactive" | error state
    // Color coded in UI
}

// Actions executed via shell_exec:
shell_exec("sudo systemctl stop " . $service);
shell_exec("sudo systemctl start " . $service);
shell_exec("sudo systemctl restart " . $service);
shell_exec("sudo restart_services.sh");  // Batch restart
```

### Post-Save Service Restarts

Different settings trigger different service restarts:
- **Timezone change:** timedatectl, PHP restart
- **Model/Language change:** install_language_label.sh, restart analysis
- **Site name/Color scheme:** restart chart_viewer
- **Audio config change:** restart recording, livestream
- **General changes:** restart_services.sh (batch)

---

## Existing Go/Preact Infrastructure

### What Exists (Can Reuse)

**Go Backend:**
- `internal/api/handlers.go` - Handler struct pattern with DI
- `internal/api/species_lists.go` - File-based list CRUD (similar pattern needed)
- Response pattern: `writeJSON(w, statusCode, data)`, `writeError(w, statusCode, msg)`
- Chi router with middleware
- Read-only SQLite via sqlc

**Preact Frontend:**
- TypeScript types in `web/src/types/api.ts`
- API hooks in `web/src/hooks/useApi.ts`
- Form patterns: `SpeciesListEditor.tsx` (modal, dual-list, save/cancel)
- `SearchFilters.tsx` (controlled inputs, callbacks)
- Confirmation modals before destructive actions
- Loading/error states
- Tailwind CSS styling with dark mode support

### What's Missing (Need to Build)

**Go Backend:**
```
internal/config/
├── config.go       # Config struct, loader, writer
├── schema.go       # Validation rules
└── watcher.go      # File change detection (optional)

internal/api/
├── settings.go     # GET/PUT /api/settings endpoints
└── services.go     # Service control endpoints
```

**Preact Frontend:**
```
web/src/
├── types/api.ts    # Add Settings types
├── hooks/useSettings.ts  # Settings-specific hooks
├── components/
│   ├── SettingsForm.tsx
│   ├── SettingsSection.tsx
│   ├── ServiceControls.tsx
│   └── form/
│       ├── NumberInput.tsx
│       ├── SliderInput.tsx
│       ├── SelectInput.tsx
│       ├── ToggleSwitch.tsx
│       └── TextArea.tsx
└── pages/
    ├── Settings.tsx        # Basic settings
    └── AdvancedSettings.tsx # Advanced settings
```

---

## Proposed API Design

### Settings Endpoints

```
GET  /api/settings
     Response: { settings: SettingsObject, schema: SchemaObject }

PUT  /api/settings
     Body: { settings: Partial<SettingsObject> }
     Response: { status: "ok", restarts: ["service1", "service2"] }
     Note: Only changed fields sent; server tracks what changed

GET  /api/settings/schema
     Response: { schema: JSONSchema }

POST /api/settings/test-notification
     Body: { config: AppriseConfig, title: string, body: string }
     Response: { success: boolean, output: string }
```

### Service Control Endpoints

```
GET  /api/services
     Response: { services: [{ name, status, enabled }] }

POST /api/services/{name}/start
POST /api/services/{name}/stop
POST /api/services/{name}/restart
POST /api/services/{name}/enable
POST /api/services/{name}/disable
     Response: { status: "ok" | "error", message: string }
```

### System Control Endpoints

```
POST /api/system/reboot
     Response: { status: "rebooting" }

POST /api/system/backup
     Response: { url: "/api/system/backup/download/{id}" }

POST /api/system/restore
     Body: multipart/form-data with backup file
     Response: { status: "ok", message: "Restore complete, rebooting..." }

POST /api/system/update
     Response: { status: "updating", message: "..." }
```

---

## TypeScript Types (Proposed)

```typescript
// Settings types
export interface Settings {
  // Location
  latitude: number;
  longitude: number;
  site_name: string;

  // Analysis
  model: 'BirdNET_GLOBAL_6K_V2.4_Model_FP16' | 'BirdNET_6K_GLOBAL_MODEL';
  confidence: number;
  sensitivity: number;
  overlap: number;
  sf_thresh: number;
  privacy_threshold: number;

  // Recording
  rec_card: string;
  channels: number;
  recording_length: number;
  extraction_length: number;
  audiofmt: 'mp3' | 'wav' | 'flac' | 'ogg' | 'opus';

  // Storage
  full_disk: 'purge' | 'keep';
  purge_threshold: number;
  max_files_species: number;

  // Notifications
  apprise: AppriseSettings;
  birdweather_id: string;

  // UI
  color_scheme: 'light' | 'dark';
  image_provider: 'WIKIPEDIA' | 'FLICKR' | '';
  flickr_api_key?: string;
  flickr_filter_email?: string;
  info_site: 'ALLABOUTBIRDS' | 'EBIRD';
  database_lang: string;

  // Advanced
  caddy_pwd: string;
  birdnetpi_url: string;
  rtsp_streams: string[];
  freqshift: FreqShiftSettings;
  log_levels: LogLevelSettings;
}

export interface AppriseSettings {
  urls: string;  // Multi-line
  title: string;
  body: string;
  minimum_time_limit: number;
  only_notify_species: string;
  notify_each_detection: boolean;
  notify_new_species: boolean;
  notify_new_species_each_day: boolean;
  weekly_report: boolean;
}

export interface FreqShiftSettings {
  tool: 'sox' | 'ffmpeg';
  hi: number;
  lo: number;
  pitch: number;
  reconnect_delay: number;
  activate_in_livestream: boolean;
}

export interface LogLevelSettings {
  recording: 'debug' | 'info' | 'warning' | 'error';
  livestream: 'debug' | 'info' | 'warning' | 'error';
  spectrogram: 'debug' | 'info' | 'warning' | 'error';
}

// Schema types for form generation
export interface SettingsSchema {
  properties: Record<string, PropertySchema>;
  required: string[];
}

export interface PropertySchema {
  type: 'string' | 'number' | 'integer' | 'boolean';
  enum?: (string | number)[];
  minimum?: number;
  maximum?: number;
  default?: any;
  description?: string;
}

// Service types
export interface ServiceStatus {
  name: string;
  status: 'active' | 'inactive' | 'failed' | 'unknown';
  enabled: boolean;
  description?: string;
}

export interface ServicesResponse {
  services: ServiceStatus[];
}
```

---

## Form Component Design

### Settings Page Layout

```
┌────────────────────────────────────────────────────────┐
│  Settings                    [Save] [Cancel] [Revert] │
├────────────────────────────────────────────────────────┤
│ ┌─ Location ──────────────────────────────────────────┐│
│ │ Latitude: [_____] Longitude: [_____] Site: [____]  ││
│ └─────────────────────────────────────────────────────┘│
│ ┌─ BirdNET Model ─────────────────────────────────────┐│
│ │ ○ V2.4 FP16 (Recommended)  ○ V2.4 FP32             ││
│ │ Species Filter Threshold: [_0.03_]                  ││
│ └─────────────────────────────────────────────────────┘│
│ ┌─ Notifications ──────────────────── [Test] ────────┐│
│ │ Apprise URLs: [____________________________]       ││
│ │ Title: [______________] Body: [_______________]    ││
│ │ ☐ Each detection  ☐ New species  ☐ Weekly report   ││
│ │ BirdWeather ID: [_______________]                   ││
│ └─────────────────────────────────────────────────────┘│
│ ┌─ Display ───────────────────────────────────────────┐│
│ │ Theme: [Light ▼]  Images: ○ Wikipedia ○ Flickr     ││
│ │ Info Site: ○ AllAboutBirds ○ eBird                 ││
│ │ Language: [English ▼]                               ││
│ └─────────────────────────────────────────────────────┘│
└────────────────────────────────────────────────────────┘
```

### Form Component Pattern

```tsx
interface SettingsFormProps {
  initialSettings: Settings;
  schema: SettingsSchema;
  onSave: (settings: Partial<Settings>) => Promise<void>;
}

function SettingsForm({ initialSettings, schema, onSave }: SettingsFormProps) {
  const [settings, setSettings] = useState(initialSettings);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [dirty, setDirty] = useState(false);
  const [saving, setSaving] = useState(false);

  // Track changes for partial update
  const changedFields = useMemo(() => {
    return Object.keys(settings).filter(
      key => settings[key] !== initialSettings[key]
    );
  }, [settings, initialSettings]);

  const handleChange = useCallback((field: keyof Settings, value: any) => {
    setSettings(prev => ({ ...prev, [field]: value }));
    setDirty(true);
    // Validate immediately
    const error = validateField(field, value, schema);
    setErrors(prev => ({ ...prev, [field]: error }));
  }, [schema]);

  const handleSave = async () => {
    // Validate all changed fields
    const allErrors = validateSettings(settings, schema);
    if (Object.keys(allErrors).length > 0) {
      setErrors(allErrors);
      return;
    }

    setSaving(true);
    try {
      // Only send changed fields
      const partial = pick(settings, changedFields);
      await onSave(partial);
      setDirty(false);
    } catch (err) {
      setErrors({ _form: err.message });
    } finally {
      setSaving(false);
    }
  };

  return (
    <form onSubmit={e => { e.preventDefault(); handleSave(); }}>
      <SettingsSection title="Location">
        <NumberInput
          label="Latitude"
          value={settings.latitude}
          onChange={v => handleChange('latitude', v)}
          min={-90}
          max={90}
          step={0.0001}
          error={errors.latitude}
        />
        {/* ... more fields */}
      </SettingsSection>
      {/* ... more sections */}
    </form>
  );
}
```

---

## Validation Strategy

### Client-Side (Immediate Feedback)

```typescript
function validateField(
  field: string,
  value: any,
  schema: SettingsSchema
): string | null {
  const prop = schema.properties[field];
  if (!prop) return null;

  // Type check
  if (prop.type === 'number' && typeof value !== 'number') {
    return 'Must be a number';
  }

  // Range check
  if (prop.minimum !== undefined && value < prop.minimum) {
    return `Minimum value is ${prop.minimum}`;
  }
  if (prop.maximum !== undefined && value > prop.maximum) {
    return `Maximum value is ${prop.maximum}`;
  }

  // Enum check
  if (prop.enum && !prop.enum.includes(value)) {
    return `Must be one of: ${prop.enum.join(', ')}`;
  }

  return null;
}
```

### Server-Side (Final Validation)

```go
func (h *Handlers) validateSettings(settings map[string]interface{}) []ValidationError {
    var errors []ValidationError

    // Use JSON Schema validation
    result, err := h.schema.Validate(gojsonschema.NewGoLoader(settings))
    if err != nil {
        return []ValidationError{{Field: "_form", Message: err.Error()}}
    }

    for _, err := range result.Errors() {
        errors = append(errors, ValidationError{
            Field:   err.Field(),
            Message: err.Description(),
        })
    }

    return errors
}
```

---

## Migration Phases

### Phase 4.2: Basic Settings Page

**Tasks:**
1. Create Go config loader (read INI file)
2. Implement `GET /api/settings` and `GET /api/settings/schema`
3. Create Preact Settings page with form components
4. Implement `PUT /api/settings` with validation
5. Add service restart logic (call existing shell scripts)
6. Test alongside PHP version

### Phase 4.3: Advanced Settings Page

**Tasks:**
1. Extend Go config for advanced parameters
2. Add frequency shifting UI section
3. Add log level controls
4. Add RTSP stream configuration (dynamic list)
5. Implement password change flow

### Phase 5: Service & System Controls

**Tasks:**
1. Implement service status endpoint
2. Add service control actions (start/stop/restart)
3. Create Service Controls component
4. Implement system actions (backup/restore/reboot)
5. Add update mechanism

---

## Security Considerations

1. **Authentication:** Settings endpoints require authentication
   - Currently HTTP Basic Auth (`birdnet` / `CADDY_PWD`)
   - Go middleware to check auth header

2. **Input Validation:** Server-side validation required
   - Never trust client validation alone
   - Use schema validation library

3. **Command Injection:** Service names must be whitelisted
   - Never execute user input directly
   - Use predefined service list

4. **Password Handling:**
   - Never return password in GET response
   - Use separate endpoint for password change
   - Require current password to change

---

## Open Questions

1. **Config Format Migration:** Should we migrate to YAML (as planned) or keep INI for backward compatibility?

2. **Service Restart UX:** Show progress/status during restarts? WebSocket notifications?

3. **Validation Timing:** Validate on blur vs on change vs on submit?

4. **Password Flow:** Require re-authentication after password change?

5. **Conditional Fields:** How to handle fields that depend on other settings (e.g., Flickr API key only when Flickr selected)?

---

## Files to Create

### Go Backend
- [ ] `internal/config/config.go`
- [ ] `internal/config/schema.go`
- [ ] `internal/config/ini.go` (INI file parser/writer)
- [ ] `internal/api/settings.go`
- [ ] `internal/api/services.go`

### Preact Frontend
- [ ] `web/src/types/settings.ts`
- [ ] `web/src/hooks/useSettings.ts`
- [ ] `web/src/components/form/NumberInput.tsx`
- [ ] `web/src/components/form/SliderInput.tsx`
- [ ] `web/src/components/form/SelectInput.tsx`
- [ ] `web/src/components/form/ToggleSwitch.tsx`
- [ ] `web/src/components/form/TextArea.tsx`
- [ ] `web/src/components/SettingsSection.tsx`
- [ ] `web/src/components/SettingsForm.tsx`
- [ ] `web/src/components/ServiceControls.tsx`
- [ ] `web/src/pages/Settings.tsx`
- [ ] `web/src/pages/AdvancedSettings.tsx`
