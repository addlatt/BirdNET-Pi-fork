# BirdNET-Pi Reorganization Plan

## Current State Analysis

### System Overview

BirdNET-Pi is a bird sound identification system running on a Raspberry Pi 5 (4GB RAM, NVMe SSD). It continuously records audio, analyzes it with a TensorFlow Lite model, and serves results via a PHP web interface.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           CURRENT ARCHITECTURE                               │
└─────────────────────────────────────────────────────────────────────────────┘

     RECORDING                ANALYSIS                 STORAGE              WEB UI
    ┌─────────┐            ┌───────────┐            ┌─────────┐          ┌─────────┐
    │ arecord │            │  Python   │            │ SQLite  │          │   PHP   │
    │   or    │──► .wav ──►│  TFLite   │──► data ──►│  + FS   │◄── read ─│  Caddy  │
    │  ffmpeg │            │  librosa  │            │         │          │         │
    └─────────┘            └───────────┘            └─────────┘          └─────────┘
         │                       │                       │                    │
    birdnet_              birdnet_                 birds.db            homepage/
    recording.sh          analysis.py              Extracted/          scripts/*.php
```

### Tech Stack

| Layer | Technology |
|-------|------------|
| OS | Raspberry Pi OS (Debian), Linux 6.12 |
| Init | systemd |
| Web Server | Caddy (reverse proxy + PHP) |
| Backend | PHP 8.4-FPM |
| Analysis | Python 3.13 + TensorFlow Lite |
| Database | SQLite |
| Audio | ALSA (arecord), FFmpeg, sox |
| Streaming | Icecast2 |
| mDNS | Avahi |

### File Inventory

| Type | Count | Location |
|------|-------|----------|
| Shell scripts (.sh) | 37 | scripts/, templates/ |
| PHP files (.php) | 23 | scripts/, homepage/ |
| Python files (.py) | 10 | scripts/, scripts/utils/ |
| Systemd services | 11 | templates/ |
| TFLite models | 4 | model/ (~100MB) |
| Binaries (gotty) | 2 | scripts/ (24MB) |
| Python wheel | 1 | root (50MB) |

---

## Problems with Current Structure

### 1. Flat Directory Structure
Everything dumped into `scripts/`:
- Core services (birdnet_analysis.py, birdnet_recording.sh)
- Web UI pages (23 PHP files)
- Install scripts
- Utility scripts
- Vendored tools (adminer, gotty)
- Runtime data (birds.db, wikipedia.db)

### 2. Binaries Committed to Git
```
scripts/gotty-aarch64     9.7 MB
scripts/gotty-x86_64      14 MB
tflite_runtime-*.whl      50 MB
────────────────────────────────
Total bloat:              ~74 MB
```

### 3. Runtime Data in Source Tree
- `scripts/birds.db` - SQLite database
- `scripts/wikipedia.db` - Cache database
- Various `.txt` state files at root

### 4. No Python Packaging
- No `pyproject.toml` or `setup.py`
- Only `scripts/utils/` is a proper module
- Hardcoded paths everywhere

### 5. Mixed Responsibilities
The `scripts/` directory contains:
- Installation logic
- Runtime services
- Web application
- CLI utilities
- Third-party tools

### 6. Hardcoded Paths

| Path | Defined In | Used By |
|------|------------|---------|
| `/etc/birdnet/birdnet.conf` | Multiple | All scripts |
| `~/BirdNET-Pi/scripts/birds.db` | helpers.py | Python + PHP |
| `~/BirdNET-Pi/model/` | helpers.py | Analysis |
| `~/BirdSongs/` | birdnet.conf | Recording + Web |
| `/usr/local/bin/` | systemd units | Services |
| `~/BirdNET-Pi/homepage/static/` | helpers.py | Fonts |

---

## Current Directory Structure

```
BirdNET-Pi/
├── birdnet/                      # Python venv (generated)
├── model/                        # ML models
│   ├── *.tflite                  # TensorFlow Lite models
│   ├── *_Labels.txt              # Species labels
│   └── l18n/                     # Translations
├── scripts/                      # ⚠️ JUNK DRAWER
│   ├── birdnet_analysis.py       # Core: analysis daemon
│   ├── birdnet_recording.sh      # Core: recording daemon
│   ├── birdnet_log.sh            # Core: log viewer
│   ├── utils/                    # ✓ Only organized part
│   │   ├── analysis.py
│   │   ├── models.py
│   │   ├── db.py
│   │   ├── helpers.py
│   │   ├── reporting.py
│   │   ├── notifications.py
│   │   ├── classes.py
│   │   └── maintainer.py
│   ├── *.php                     # Web UI (23 files)
│   ├── install_*.sh              # Installation (6 files)
│   ├── *.sh                      # Utilities (25+ files)
│   ├── gotty-aarch64             # ⚠️ Binary
│   ├── gotty-x86_64              # ⚠️ Binary
│   ├── birds.db                  # ⚠️ Runtime data
│   ├── wikipedia.db              # ⚠️ Runtime data
│   └── filemanager/              # Vendored PHP app
├── homepage/                     # Web UI entry
│   ├── index.php
│   ├── views.php
│   ├── style.css
│   ├── images/
│   └── static/
├── templates/                    # Systemd + config templates
│   ├── *.service
│   ├── *.cron
│   └── *.conf
├── tests/                        # Sparse tests
├── docs/                         # Documentation
├── .github/                      # CI/CD
├── birdnet.conf                  # Config template
├── requirements.txt              # Python deps
├── tflite_runtime-*.whl          # ⚠️ Vendored wheel (50MB)
└── *.txt                         # Species lists, state files
```

---

## Dependency Graph

### Python Analysis Chain
```
birdnet_analysis.py
├── inotify (watches StreamData/)
├── utils/helpers.py
│   └── reads /etc/birdnet/birdnet.conf
├── utils/analysis.py
│   ├── librosa (audio loading)
│   ├── numpy (signal processing)
│   └── utils/models.py
│       └── tflite_runtime (ML inference)
└── utils/reporting.py
    ├── subprocess → sox (audio extraction)
    ├── PIL (spectrogram images)
    ├── sqlite3 → birds.db
    ├── utils/notifications.py → apprise
    └── requests → BirdWeather API
```

### PHP Web Chain
```
Caddy → PHP-FPM
├── homepage/index.php (router)
├── scripts/common.php (shared lib)
│   ├── reads /etc/birdnet/birdnet.conf
│   └── queries birds.db
└── scripts/*.php (pages)
    ├── overview.php
    ├── config.php
    ├── stats.php
    └── ...
```

### Service Dependencies
```
systemd
├── birdnet_recording.service
│   └── writes → ~/BirdSongs/StreamData/*.wav
├── birdnet_analysis.service
│   ├── watches → ~/BirdSongs/StreamData/
│   ├── writes → ~/BirdSongs/Extracted/
│   └── writes → scripts/birds.db
├── caddy.service (http)
│   └── reads → ~/BirdSongs/Extracted/
├── livestream.service
│   └── ffmpeg → icecast2
└── avahi-daemon.service
    └── broadcasts → *.local
```

---

## Shared State Analysis

### Overview Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           SHARED STATE MAP                                   │
└─────────────────────────────────────────────────────────────────────────────┘

                    ┌──────────────────────────┐
                    │  /etc/birdnet/birdnet.conf │  ◄── SINGLE SOURCE OF TRUTH
                    │  (INI format, ~100 keys)  │
                    └────────────┬─────────────┘
                                 │
        ┌────────────────────────┼────────────────────────┐
        │                        │                        │
        ▼                        ▼                        ▼
┌───────────────┐      ┌────────────────┐       ┌────────────────┐
│    Python     │      │      PHP       │       │     Shell      │
│  get_settings()│      │  get_config()  │       │  source $conf  │
│  helpers.py   │      │  common.php    │       │  direct vars   │
└───────┬───────┘      └────────┬───────┘       └────────┬───────┘
        │                       │                        │
        │  Singleton w/         │  $_SESSION cache       │  Direct env
        │  global _settings     │  + mtime check         │  vars in shell
        │                       │                        │
        └───────────────────────┼────────────────────────┘
                                │
                                ▼
                    ┌──────────────────────────┐
                    │    ~/BirdNET-Pi/scripts/ │
                    │        birds.db          │  ◄── SHARED DATABASE
                    │      (SQLite, ~1MB)      │
                    └────────────┬─────────────┘
                                 │
        ┌────────────────────────┼────────────────────────┐
        │                        │                        │
        ▼                        ▼                        ▼
┌───────────────┐      ┌────────────────┐       ┌────────────────┐
│    Python     │      │      PHP       │       │     Shell      │
│   WRITE path  │      │   READ-ONLY    │       │   READ-ONLY    │
│ reporting.py  │      │   common.php   │       │   sqlite3 CLI  │
│   db.py (RO)  │      │   get_db()     │       │                │
└───────────────┘      └────────────────┘       └────────────────┘
```

### 1. Configuration State

#### `/etc/birdnet/birdnet.conf`
- **Format**: INI-style (parsed as PHP ini by both Python and PHP)
- **Owner**: root (requires sudo to modify)
- **Size**: ~100 configuration keys

| Access Method | Language | Caching | Location |
|---------------|----------|---------|----------|
| `get_settings()` | Python | Global singleton `_settings` | `helpers.py` |
| `get_config()` | PHP | `$_SESSION` + mtime check | `common.php` |
| `source $conf` | Shell | None (direct env vars) | Each script |

#### Python Config Access (helpers.py)
```python
_settings = None  # Module-level singleton

def get_settings(settings_path='/etc/birdnet/birdnet.conf', force_reload=False):
    global _settings
    if _settings is None or force_reload:
        # Parse INI with custom parser (strips quotes)
        parser = PHPConfigParser(interpolation=None)
        parser.read_file(chain(("[top]",), open(settings_path)))
        _settings = parser['top']
    return _settings
```

#### PHP Config Access (common.php)
```php
function get_config($force_reload = false) {
    $mtime = stat('/etc/birdnet/birdnet.conf')["mtime"];
    if ($_SESSION['my_config_version'] !== $mtime) {
        $force_reload = true;  // Auto-reload if file changed
    }
    if (!isset($_SESSION['my_config']) || $force_reload) {
        $source = preg_replace("~^#+.*$~m", "", file_get_contents('/etc/birdnet/birdnet.conf'));
        $_SESSION['my_config'] = parse_ini_string($source);
    }
    return $_SESSION['my_config'];
}
```

#### Shell Config Access
```bash
source /etc/birdnet/birdnet.conf  # Direct variable injection
echo $LATITUDE $LONGITUDE         # Used as env vars
```

---

### 2. Database State

#### `~/BirdNET-Pi/scripts/birds.db`
- **Format**: SQLite3
- **Size**: ~1MB (grows with detections)
- **Schema**: Single table `detections`

| Access | Language | Mode | Connection |
|--------|----------|------|------------|
| `reporting.py` | Python | **READ-WRITE** | `sqlite3.connect(DB_PATH)` |
| `db.py` | Python | READ-ONLY | `sqlite3.connect(f"file:{DB_PATH}?mode=ro", uri=True)` |
| `common.php` | PHP | READ-ONLY | `new SQLite3('./scripts/birds.db', SQLITE3_OPEN_READONLY)` |
| Shell scripts | Bash | READ-ONLY | `sqlite3 ~/BirdNET-Pi/scripts/birds.db` |

#### Write Path (only in reporting.py)
```python
def write_to_db(file: ParseFileName, detection: Detection):
    for attempt_number in range(3):  # Retry logic for busy DB
        try:
            con = sqlite3.connect(DB_PATH)
            cur = con.cursor()
            cur.execute("INSERT INTO detections VALUES (?, ?, ...)", (...))
            con.commit()
            break
        except sqlite3.OperationalError:
            sleep(0.3)
```

#### Read Paths
- **Python `db.py`**: Module-level singleton `_DB`, read-only URI mode
- **PHP `common.php`**: `$_db` with `SQLITE3_OPEN_READONLY`, 1000ms busy timeout
- **Shell**: Direct `sqlite3` CLI queries

---

### 3. Filesystem State

#### Shared Directories
```
~/BirdSongs/                      ◄── Configured via RECS_DIR in birdnet.conf
├── StreamData/                   ◄── Recording daemon writes here
│   ├── *.wav                     │   Live recordings (15-second chunks)
│   ├── *.wav.json                │   Analysis metadata
│   ├── analyzing_now.txt         │   Lock file: current file being analyzed
│   └── spectrogram.png           │   Live spectrogram for web UI
├── Extracted/                    ◄── Analysis daemon writes here
│   └── By_Date/YYYY-MM-DD/       │   Extracted bird clips + spectrograms
└── Processed/                    ◄── Completed recordings
```

#### State Files at Repo Root
| File | Purpose | Written By | Read By |
|------|---------|------------|---------|
| `analyzing_now.txt` | Lock: prevents double-analysis | `birdnet_analysis.py` | `helpers.py` |
| `exclude_species_list.txt` | Species to ignore | User/Web UI | Analysis |
| `whitelist_species_list.txt` | Species to always notify | User/Web UI | Notifications |
| `confirmed_species_list.txt` | Verified species | User/Web UI | Stats |
| `apprise.txt` | Notification URLs | User/Web UI | `notifications.py` |
| `body.txt` | Notification template | User/Web UI | `notifications.py` |
| `BirdDB.txt` | Wikipedia cache? | `maintainer.py` | Web UI |

---

### 4. Path Definitions (Centralized)

#### Python: `helpers.py` (THE source of truth for Python)
```python
BASE_PATH = os.path.abspath(os.path.join(os.path.dirname(__file__), '..', '..'))
# → ~/BirdNET-Pi

DB_PATH = os.path.join(BASE_PATH, 'scripts/birds.db')
# → ~/BirdNET-Pi/scripts/birds.db

MODEL_PATH = os.path.join(BASE_PATH, 'model')
# → ~/BirdNET-Pi/model

FONT_DIR = os.path.join(BASE_PATH, 'homepage/static')
# → ~/BirdNET-Pi/homepage/static

ANALYZING_NOW = os.path.expanduser('~/BirdSongs/StreamData/analyzing_now.txt')
# → ~/BirdSongs/StreamData/analyzing_now.txt (HARDCODED!)
```

#### PHP: `common.php`
```php
define('__ROOT__', dirname(dirname(__FILE__)));
// → ~/BirdNET-Pi

// DB path is RELATIVE (fragile!)
$_db = new SQLite3('./scripts/birds.db', ...);

// Home derived from config
$home = '/home/' . get_config()['BIRDNET_USER'];
```

#### Shell: Derived from config
```bash
source /etc/birdnet/birdnet.conf
RECS_DIR="$HOME/BirdSongs"  # From config
DB="$HOME/BirdNET-Pi/scripts/birds.db"  # Often hardcoded
```

---

### 5. Concurrency & Locking

#### Current Locking Mechanisms
| Resource | Lock Type | Implementation |
|----------|-----------|----------------|
| Database | SQLite busy timeout | 1000ms (PHP), retry loop (Python) |
| Analysis | File lock | `analyzing_now.txt` contains current file |
| Recording | Open file check | `lsof` to skip files still being written |

#### Race Conditions Possible
1. **Config reload**: Python singleton doesn't auto-reload; PHP uses mtime
2. **DB writes**: Only `reporting.py` writes, but no explicit locking
3. **StreamData**: Recording writes while analysis reads (handled by `lsof`)

---

### 6. Key Insight for Reorganization

**Centralization points** (touch these to update paths):

| Component | File | What to Change |
|-----------|------|----------------|
| Python paths | `helpers.py` | `BASE_PATH`, `DB_PATH`, `MODEL_PATH`, `FONT_DIR` |
| PHP paths | `common.php` | `__ROOT__`, DB path, config path |
| Shell paths | `birdnet.conf` | `RECS_DIR`, add new path vars |
| Services | `*.service` | `ExecStart`, `WorkingDirectory` |
| Web server | `Caddyfile` | `root` directive |

**The good news**: Paths are mostly centralized in `helpers.py` (Python) and `common.php` (PHP). The bad news: Some paths are hardcoded (e.g., `ANALYZING_NOW`, PHP's relative `./scripts/birds.db`).

---

## Detailed Import Map

### Python File Imports

#### Core Service: `birdnet_analysis.py`
```python
# Standard library
import logging
import os, os.path
import re
import signal
import sys
import threading
from queue import Queue
from subprocess import CalledProcessError

# Third-party
import inotify.adapters
from inotify.constants import IN_CLOSE_WRITE

# Internal
from utils.analysis import load_global_model, run_analysis
from utils.classes import ParseFileName
from utils.helpers import get_settings, get_wav_files, ANALYZING_NOW
from utils.reporting import extract_detection, summary, write_to_file, write_to_db, apprise, bird_weather, heartbeat, update_json_file
```

#### `utils/analysis.py` (Audio Processing)
```python
# Standard library
import logging
import os
import time

# Third-party
import librosa
import numpy as np

# Internal
from .classes import Detection, ParseFileName
from .helpers import get_settings, get_language
from .models import get_model
```

#### `utils/models.py` (ML Inference)
```python
# Standard library
import logging
import math
import os
import operator

# Third-party
import numpy as np
# tflite_runtime imported dynamically

# Internal
from .helpers import get_settings, get_model_labels, MODEL_PATH
```

#### `utils/helpers.py` (Configuration & Paths)
```python
# Standard library
import glob
import json
import os
import re
import subprocess
from collections import OrderedDict
from configparser import ConfigParser
from itertools import chain

# DEFINES CRITICAL PATHS:
# BASE_PATH = ~/BirdNET-Pi
# DB_PATH = ~/BirdNET-Pi/scripts/birds.db
# MODEL_PATH = ~/BirdNET-Pi/model
# FONT_DIR = ~/BirdNET-Pi/homepage/static
# ANALYZING_NOW = ~/BirdSongs/StreamData/analyzing_now.txt
```

#### `utils/reporting.py` (Post-Processing)
```python
# Standard library
import glob
import io
import json
import logging
import os
import sqlite3
import subprocess
import tempfile
from time import sleep

# Third-party
import requests
import soundfile
from PIL import Image, ImageDraw, ImageFont

# Internal
from .classes import Detection, ParseFileName
from .helpers import get_settings, get_font, DB_PATH
from .notifications import sendAppriseNotifications
```

#### `utils/notifications.py` (Alerts)
```python
# Standard library
import html
import os
import socket
import time

# Third-party
import apprise
import requests

# Internal
from .db import get_todays_count_for, get_this_weeks_count_for
from .helpers import get_settings
```

#### `utils/db.py` (Database)
```python
# Standard library
import sqlite3
import time as timeim
from datetime import datetime

# Internal
from .helpers import DB_PATH
```

#### `utils/classes.py` (Data Structures)
```python
# Standard library
import datetime
import os
import re

# Third-party
from tzlocal import get_localzone
```

#### `utils/maintainer.py` (Maintenance)
```python
# Standard library
import os
import re
import time

# Third-party
import requests

# Internal
from .helpers import MODEL_PATH, save_language, get_language
```

#### Visualization: `daily_plot.py`
```python
# Standard library
import argparse
import os
import sqlite3
import textwrap
from datetime import datetime
from time import sleep

# Third-party
import matplotlib.font_manager as font_manager
import matplotlib.pyplot as plt
import numpy as np
import pandas as pd
import seaborn as sns
from matplotlib import rcParams
from matplotlib.colors import LogNorm

# Internal
from utils.helpers import DB_PATH, FONT_DIR, get_settings, get_font
```

#### Visualization: `plotly_streamlit.py`
```python
# Standard library
import os
import sqlite3
from datetime import datetime, timedelta
from sqlite3 import Connection

# Third-party
import numpy as np
import pandas as pd
import plotly.express as px
import plotly.graph_objects as go
import plotly.io as pio
import streamlit as st
from dateutil import tz
from numpy import ma
from plotly.subplots import make_subplots
from sklearn.preprocessing import normalize
from suntime import Sun

# Internal
from utils.helpers import get_settings
```

#### CLI: `species.py`
```python
# Standard library
import argparse
import datetime
import os

# Internal
from utils.helpers import get_settings, MODEL_PATH
from utils.models import MDataModel1, MDataModel2
```

#### CLI: `send_test_notification.py`
```python
# Standard library
import argparse
import datetime
import logging
import sys

# Internal
from utils.db import get_latest
from utils.helpers import get_settings
from utils import notifications
```

---

### Python Internal Dependency Graph

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        PYTHON INTERNAL DEPENDENCIES                         │
└─────────────────────────────────────────────────────────────────────────────┘

birdnet_analysis.py
    │
    ├──► utils/helpers.py ◄──────────────────────────────────────────┐
    │        │                                                        │
    │        └──► [configparser, glob, json, os, re, subprocess]      │
    │                                                                 │
    ├──► utils/classes.py ◄───────────────────────────────────────────┤
    │        │                                                        │
    │        └──► [datetime, os, re, tzlocal]                         │
    │                                                                 │
    ├──► utils/analysis.py                                            │
    │        │                                                        │
    │        ├──► utils/helpers.py ────────────────────────────────►──┤
    │        ├──► utils/classes.py ────────────────────────────────►──┤
    │        └──► utils/models.py                                     │
    │                 │                                               │
    │                 ├──► utils/helpers.py ───────────────────────►──┤
    │                 └──► [tflite_runtime, numpy]                    │
    │                                                                 │
    └──► utils/reporting.py                                           │
             │                                                        │
             ├──► utils/helpers.py ────────────────────────────────►──┤
             ├──► utils/classes.py ────────────────────────────────►──┘
             ├──► utils/notifications.py
             │        │
             │        ├──► utils/helpers.py ───────────────────────►──┐
             │        └──► utils/db.py                                │
             │                 │                                      │
             │                 └──► utils/helpers.py ──────────────►──┘
             │
             └──► [PIL, requests, soundfile, sqlite3, subprocess→sox]
```

### Circular Dependency Check: ✅ NONE FOUND
The dependency graph is a clean DAG (directed acyclic graph).

---

### External Python Dependencies (by usage count)

| Package | Count | Used For |
|---------|-------|----------|
| os | 11 | File/path operations |
| time/datetime | 6 | Timestamps |
| sqlite3 | 5 | Database access |
| numpy | 5 | Numerical operations |
| logging | 5 | Logging |
| re | 4 | Regex |
| subprocess | 3 | Shell commands (sox, lsof) |
| requests | 3 | HTTP (BirdWeather, Wikipedia) |
| argparse | 3 | CLI parsing |
| pandas | 2 | Data analysis |
| json | 2 | JSON parsing |
| librosa | 1 | Audio loading |
| apprise | 1 | Notifications |
| streamlit | 1 | Dashboard |
| plotly | 1 | Charts |
| matplotlib | 1 | Plots |
| seaborn | 1 | Heatmaps |
| PIL | 1 | Image processing |
| soundfile | 1 | Audio I/O |
| tflite_runtime | 1 | ML inference |
| tzlocal | 1 | Timezone |
| suntime | 1 | Sunrise/sunset |
| sklearn | 1 | Normalization |
| inotify | 1 | File watching |

---

### PHP Include Map

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            PHP INCLUDE GRAPH                                 │
└─────────────────────────────────────────────────────────────────────────────┘

homepage/index.php (ENTRY POINT)
    │
    ├──► require_once 'scripts/common.php' ◄─────────────────────────────────┐
    │                                                                         │
    ├──► include_once 'scripts/api.php'                                       │
    │        └──► require_once 'scripts/common.php' ─────────────────────────►┤
    │                                                                         │
    └──► homepage/views.php                                                   │
             │                                                                │
             ├──► require_once 'scripts/common.php' ─────────────────────────►┤
             │                                                                │
             ├──► include('scripts/system_controls.php')                      │
             │        └──► require_once 'scripts/common.php' ────────────────►┤
             │                                                                │
             ├──► include('scripts/service_controls.php')                     │
             │        └──► require_once 'scripts/common.php' ────────────────►┤
             │                                                                │
             ├──► include('scripts/overview.php')                             │
             │        └──► require_once 'scripts/common.php' ────────────────►┤
             │                                                                │
             ├──► include('scripts/todays_detections.php')                    │
             │        └──► require_once 'scripts/common.php' ────────────────►┤
             │                                                                │
             ├──► include('scripts/stats.php')                                │
             │        └──► require_once 'scripts/common.php' ────────────────►┤
             │                                                                │
             ├──► include('scripts/history.php')                              │
             │        └──► require_once 'scripts/common.php' ────────────────►┤
             │                                                                │
             ├──► include('scripts/spectrogram.php')                          │
             │        └──► require_once 'scripts/common.php' ────────────────►┤
             │                                                                │
             └──► include('scripts/weekly_report.php')
                      └──► require_once 'scripts/common.php' ────────────────►┘

scripts/common.php (SHARED LIBRARY - hub of all PHP)
    │
    ├──► require 'scripts/ebird.php'
    │
    ├──► Reads: /etc/birdnet/birdnet.conf
    ├──► Queries: ~/BirdNET-Pi/scripts/birds.db
    └──► shell_exec: timedatectl, systemd-escape

Other pages with direct common.php dependency:
    ├──► scripts/config.php       → require_once 'scripts/common.php'
    ├──► scripts/advanced.php     → require_once 'scripts/common.php'
    ├──► scripts/play.php         → require_once 'scripts/common.php'
    ├──► scripts/backup.php       → require_once 'scripts/common.php'
    ├──► scripts/restore.php      → require_once 'scripts/common.php'
    └──► scripts/species_tools.php → require_once 'scripts/common.php'
```

### Key Insight: `common.php` is the Hub
Every PHP file depends on `common.php`. This is good for refactoring - we only need to update paths in one place.

---

### Shell Script Dependency Map

#### Scripts Sourcing `birdnet.conf` (24 scripts)
```
backup_data.sh          disk_check.sh           restart_services.sh
birdnet_changeidentification.sh   disk_species_clean.sh   species_notifier.sh
birdnet_recording.sh    disk_species_count.sh   spectrogram.sh
cleanup.sh              dump_logs.sh            uninstall.sh
clear_all_data.sh       install_birdnet.sh      update_birdnet.sh
createdb.sh             install_config.sh       update_birdnet_snippets.sh
custom_recording.sh     install_language_label.sh   update_caddyfile.sh
livestream.sh           update_species.sh       weekly_report.sh
```

#### Shell Script Call Graph
```
install_birdnet.sh (MAIN INSTALLER)
    ├──► install_config.sh
    │        └──► stop_core_services.sh
    ├──► install_helpers.sh
    │        └──► install.sh (external?)
    ├──► install_language_label.sh
    └──► install_services.sh
             ├──► createdb.sh
             ├──► birdnet_recording.sh (copies)
             ├──► birdnet_log.sh (copies)
             ├──► livestream.sh (copies)
             ├──► spectrogram.sh (copies)
             ├──► custom_recording.sh (copies)
             └──► install_helpers.sh

update_birdnet.sh
    ├──► pre_update.sh
    └──► update_birdnet_snippets.sh
             ├──► install_helpers.sh
             ├──► install_language_label.sh
             ├──► restart_services.sh
             └──► update_caddyfile.sh

backup_data.sh
    ├──► stop_core_services.sh
    ├──► install_language_label.sh
    ├──► restart_services.sh
    └──► update_caddyfile.sh

clear_all_data.sh
    ├──► createdb.sh
    └──► restart_services.sh

cleanup.sh ──► stop_core_services.sh
disk_check.sh ──► stop_core_services.sh
print_diagnostic_info.sh ──► extra_info.sh
species_notifier.sh ──► update_species.sh
```

---

### External Binary Dependencies

#### Called from Python (via subprocess)
| Binary | Called From | Purpose |
|--------|-------------|---------|
| `sox` | reporting.py | Audio extraction, spectrograms |
| `lsof` | helpers.py | Check open files |

#### Called from Shell Scripts
| Binary | Usage Count | Purpose |
|--------|-------------|---------|
| `systemctl` | 45 | Service management |
| `arecord` | 14 | ALSA audio recording |
| `sqlite3` | 10 | Database queries |
| `sox` | 9 | Audio processing |
| `ffmpeg` | 9 | Audio/video transcoding |
| `curl` | 9 | HTTP requests |
| `journalctl` | 3 | Log viewing |
| `wget` | 2 | File downloads |

#### Called from PHP (via shell_exec/exec)
| Binary/Command | Called From | Purpose |
|----------------|-------------|---------|
| `timedatectl` | common.php, config.php | Timezone management |
| `systemctl` | config.php, advanced.php | Service control |
| `sudo` | Multiple | Privilege escalation |
| `rm` | play.php | File deletion |
| Shell scripts | config.php, advanced.php | Various admin tasks |

---

### Configuration Variables Used

#### Python Config Keys (from birdnet.conf)
| Key | Usage Count | Purpose |
|-----|-------------|---------|
| RECS_DIR | 5 | Recording directory |
| DATABASE_LANG | 5 | Label language |
| MODEL | 4 | ML model name |
| COLOR_SCHEME | 4 | UI theme |
| LATITUDE | 3 | Location |
| LONGITUDE | 3 | Location |
| SENSITIVITY | 2 | Detection sensitivity |
| OVERLAP | 2 | Analysis overlap |
| HEARTBEAT_URL | 2 | Monitoring |
| CONFIDENCE | 2 | Min confidence |

#### PHP Config Keys (from birdnet.conf)
| Key | Usage Count | Purpose |
|-----|-------------|---------|
| RTSP_STREAM | 6 | RTSP input URL |
| RECORDING_LENGTH | 5 | Segment duration |
| RARE_SPECIES_THRESHOLD | 5 | Highlight threshold |
| FREQSHIFT_* | 5 | Audio frequency shift |
| MODEL | 4 | ML model |
| SF_THRESH | 3 | Score filter |
| IMAGE_PROVIDER | 3 | Wikipedia/Flickr |
| CADDY_PWD | 3 | Web auth |
| AUDIOFMT | 3 | Output format |

---

### Critical Path Definitions (helpers.py)

```python
BASE_PATH = os.path.abspath(os.path.join(os.path.dirname(__file__), '..', '..'))
# Resolves to: ~/BirdNET-Pi

DB_PATH = os.path.join(BASE_PATH, 'scripts/birds.db')
# Resolves to: ~/BirdNET-Pi/scripts/birds.db

MODEL_PATH = os.path.join(BASE_PATH, 'model')
# Resolves to: ~/BirdNET-Pi/model

FONT_DIR = os.path.join(BASE_PATH, 'homepage/static')
# Resolves to: ~/BirdNET-Pi/homepage/static

ANALYZING_NOW = os.path.expanduser('~/BirdSongs/StreamData/analyzing_now.txt')
# Resolves to: ~/BirdSongs/StreamData/analyzing_now.txt
```

### Path Update Strategy for Reorganization
When restructuring, these are the ONLY Python locations that need path updates:
1. `helpers.py` - Update `BASE_PATH`, `DB_PATH`, `MODEL_PATH`, `FONT_DIR`
2. `common.php` - Update config file path, DB path
3. Systemd service files - Update `ExecStart` paths
4. Caddyfile - Update document root

---

## Proposed New Structure

```
birdnet-pi/
├── src/
│   ├── birdnet/                   # Python package
│   │   ├── __init__.py
│   │   ├── analyzer/              # Analysis service
│   │   │   ├── __init__.py
│   │   │   ├── daemon.py          # Main daemon (was birdnet_analysis.py)
│   │   │   ├── audio.py           # Audio processing
│   │   │   ├── inference.py       # ML model inference
│   │   │   └── reporting.py       # Post-processing
│   │   ├── models/                # Model loading
│   │   │   ├── __init__.py
│   │   │   ├── loader.py
│   │   │   └── species.py
│   │   ├── db/                    # Database layer
│   │   │   ├── __init__.py
│   │   │   └── queries.py
│   │   ├── notifications/         # Alerts
│   │   │   ├── __init__.py
│   │   │   └── apprise.py
│   │   └── config.py              # Configuration management
│   │
│   ├── recording/                 # Recording service
│   │   ├── record.sh              # Main recording script
│   │   └── stream.sh              # Livestream script
│   │
│   └── web/                       # PHP web application
│       ├── public/                # Document root (served by Caddy)
│       │   ├── index.php          # Front controller
│       │   ├── assets/            # CSS, JS, images
│       │   │   ├── css/
│       │   │   ├── js/
│       │   │   └── images/
│       │   └── .htaccess          # (if needed for PHP-FPM)
│       ├── app/                   # Application logic (NOT in doc root)
│       │   ├── bootstrap.php      # App initialization
│       │   ├── pages/             # Page controllers
│       │   │   ├── overview.php
│       │   │   ├── config.php
│       │   │   ├── stats.php
│       │   │   ├── history.php
│       │   │   └── ...
│       │   └── lib/               # Shared PHP libraries
│       │       ├── common.php
│       │       ├── db.php
│       │       └── auth.php
│       └── views/                 # Templates (optional future)
│           └── layout.php
│
├── config/
│   ├── birdnet.conf.example       # Config template
│   ├── systemd/                   # Service files
│   │   ├── birdnet-analyzer.service
│   │   ├── birdnet-recorder.service
│   │   └── ...
│   └── caddy/                     # Web server config
│       └── Caddyfile
│
├── models/                        # ML models
│   ├── download.sh                # Download script (not committed)
│   └── .gitkeep
│
├── scripts/                       # Admin scripts only
│   ├── install.sh
│   ├── uninstall.sh
│   ├── backup.sh
│   └── update.sh
│
├── tools/                         # CLI utilities
│   ├── species-list.py
│   ├── test-notification.py
│   └── diagnostics.sh
│
├── tests/
│   ├── test_analyzer.py
│   ├── test_models.py
│   └── test_notifications.py
│
├── data/                          # Runtime data (.gitignored)
│   ├── db/
│   │   └── birds.db
│   ├── recordings/
│   │   ├── stream/
│   │   ├── extracted/
│   │   └── processed/
│   └── cache/
│
├── .gitignore
├── pyproject.toml                 # Modern Python packaging
├── requirements.txt               # For pip compatibility
├── LICENSE
└── README.md
```

---

## Current Web Architecture (Critical for Migration)

### The Symlink Problem

The current setup uses a **symlink-based document root** that mixes source code with runtime data:

```
~/BirdSongs/Extracted/  (Caddy document root)
├── index.php       → symlink → ~/BirdNET-Pi/homepage/index.php
├── views.php       → symlink → ~/BirdNET-Pi/homepage/views.php
├── scripts/        → symlink → ~/BirdNET-Pi/scripts/  (entire dir!)
├── images/         → symlink → ~/BirdNET-Pi/homepage/images/
├── static/         → symlink → ~/BirdNET-Pi/homepage/static/
├── style.css       → symlink → ~/BirdNET-Pi/homepage/style.css
├── By_Date/        ← ACTUAL DATA (bird recordings by date)
├── Charts/         ← ACTUAL DATA (generated chart images)
└── spectrogram.png → symlink → ~/BirdSongs/StreamData/spectrogram.png
```

**Why this exists**: The original developers wanted to:
1. Serve PHP files and static assets
2. Serve bird recordings with browseable directories
3. Use a single Caddy `root` directive for simplicity

**Problems**:
- Source code mixed with runtime data
- Symlinks created during install (fragile)
- Relative paths in PHP depend on this layout (e.g., `./scripts/birds.db`)
- Can't cleanly separate code from data

### Proposed: Route-Based Caddy Configuration

Instead of symlinks, use Caddy's routing to serve different content from different locations:

```caddyfile
http:// {
  # PHP application - clean document root
  handle /* {
    root * /home/{user}/birdnet-pi/src/web/public
    php_fastcgi unix//run/php/php-fpm.sock
    file_server
  }

  # Bird recordings - browseable file server
  handle /recordings/* {
    root * /home/{user}/BirdSongs/Extracted
    uri strip_prefix /recordings
    file_server browse
  }

  # Direct access for audio playback (legacy URL compatibility)
  handle /By_Date/* {
    root * /home/{user}/BirdSongs/Extracted
    file_server
  }

  handle /Charts/* {
    root * /home/{user}/BirdSongs/Extracted
    file_server
  }

  # Live spectrogram
  handle /spectrogram.png {
    root * /home/{user}/BirdSongs/StreamData
    file_server
  }

  # Reverse proxies (unchanged)
  reverse_proxy /stream localhost:8000
  reverse_proxy /log* localhost:8080
  reverse_proxy /stats* localhost:8501
  reverse_proxy /terminal* localhost:8888
}
```

### PHP Path Migration

Current relative paths must be updated to use configuration:

| Current Path | New Approach |
|--------------|--------------|
| `./scripts/birds.db` | `Config::get('DB_PATH')` or `__DIR__ . '/../data/birds.db'` |
| `./scripts/*.txt` | Move to `data/` directory, use config |
| `scripts/common.php` | `require __DIR__ . '/../app/lib/common.php'` |

The front controller pattern (`public/index.php`) will handle routing and include paths correctly.

---

## Migration Strategy

### Phase 1: Repository Cleanup (No Runtime Changes) ✅ COMPLETE

Removed 24MB of gotty binaries from git and created `scripts/download_gotty.sh` to fetch them on-demand from GitHub releases during installation. Updated `scripts/install_services.sh` to call the download script if binaries are missing. Reorganized `.gitignore` with clear sections for downloaded binaries, runtime data, Python artifacts, and IDE files.

- [x] Remove binaries from git (gotty, wheel)
- [x] Add download scripts for binaries
- [x] Move runtime data locations to .gitignore
- [x] Create proper .gitignore

### Phase 2: Python Restructure ✅ COMPLETE

Moved `scripts/utils/` to `src/birdnet/` and created a modern `pyproject.toml` with PEP 621 metadata, making the codebase an installable Python package. Updated all Python imports from `from utils.xxx` to `from birdnet.xxx` across scripts and tests, including mock patch paths. Modified `scripts/install_birdnet.sh` to install the package in editable mode (`pip install -e .`) after dependencies.

- [x] Create `src/birdnet/` package structure
- [x] Move `scripts/utils/` → `src/birdnet/`
- [x] Update imports
- [x] Create `pyproject.toml`
- [ ] Update systemd service paths (deferred - services use /usr/local/bin symlinks)

### Phase 3: Web Restructure (Most Complex - Requires Caddy Changes)
- [ ] Create `src/web/public/` structure (front controller pattern)
- [ ] Create `src/web/app/` for application logic
- [ ] Move PHP files with proper organization:
  - `homepage/index.php` → `src/web/public/index.php` (becomes front controller)
  - `homepage/views.php` → `src/web/app/router.php`
  - `scripts/common.php` → `src/web/app/lib/common.php`
  - `scripts/*.php` (pages) → `src/web/app/pages/`
  - `homepage/static/` → `src/web/public/assets/`
  - `homepage/images/` → `src/web/public/assets/images/`
- [ ] Update PHP include paths (remove reliance on symlink structure)
- [ ] Update `get_db()` to use absolute path from config, not `./scripts/birds.db`
- [ ] Create new Caddy config with route-based serving (see above)
- [ ] Remove symlink creation from install scripts
- [ ] Test all pages and audio playback

### Phase 4: Shell Script Organization
- [ ] Separate install vs runtime vs utility scripts
- [ ] Move to appropriate directories
- [ ] Update all path references

### Phase 5: Configuration
- [ ] Centralize path definitions
- [ ] Environment variable support
- [ ] Config file validation

---

## Open Questions

1. **Containerization later?** - Structure should support Docker but not require it
2. **Backwards compatibility?** - Do we need migration scripts for existing installs?
3. **Fork vs PR?** - Contributing back to upstream?
4. **Model storage?** - Keep in repo or download on install?
5. **Database migrations?** - Schema versioning needed?

---

## Next Steps

1. Create the new directory structure
2. Start with Python reorganization (cleanest, most isolated)
3. Update imports and test
4. Move to PHP/web reorganization
5. Update systemd services
6. Test full system
7. Document changes

---

## References

- Original repo: https://github.com/mcguirepr89/BirdNET-Pi
- BirdNET-Analyzer: https://github.com/kahst/BirdNET-Analyzer
- TensorFlow Lite: https://www.tensorflow.org/lite





