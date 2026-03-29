# birdnet-pi fork

a modernized fork of [Nachtzuster's BirdNET-Pi](https://github.com/Nachtzuster/BirdNET-Pi) with a Go API, Preact frontend, and Python services.

the new preact app runs at `/`.

![overview](docs/overview.png)

## what's different

this fork replaces the original web interface and shell scripts with:

- **go api server** - REST api, websocket streaming, task scheduler, auto-restart
- **preact frontend** - modern typescript ui, real-time updates via websocket
- **python services** - spectrogram generation, livestreaming (replaces shell scripts)
- **same ml backend** - birdnet analysis unchanged

the bird detection stuff is identical to upstream. this is a frontend/api/service rewrite.

## architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                              Browser                                │
└──────────────────────────────────┬──────────────────────────────────┘
                                   │
                              ┌────▼────┐
                              │  Caddy  │ :80
                              └────┬────┘
           ┌───────────┬──────────┼──────────┬───────────┐
           │           │          │          │           │
      ┌────▼────┐ ┌────▼────┐ ┌───▼───┐ ┌────▼────┐ ┌────▼────┐
      │ Go API  │ │ Preact  │ │Icecast│ │ gotty   │ │Streamlit│
      │  :8080  │ │ static  │ │ :8000 │ │  :8888  │ │  :8501  │
      └────┬────┘ └─────────┘ └───▲───┘ └─────────┘ └─────────┘
           │                      │
    ┌──────┼──────┐               │
    │      │      │               │
┌───▼──┐ ┌─▼──┐ ┌─▼────────┐ ┌────┴─────────┐
│SQLite│ │ ML │ │WebSocket │ │ livestream.py│
│      │ │Svc │ │  Hub     │ └──────────────┘
└──────┘ └────┘ └──────────┘
                     │
          ┌──────────┼──────────┐
          │          │          │
   ┌──────▼───┐ ┌────▼────┐ ┌───▼────────────┐
   │ /ws/logs │ │   /ws   │ │ spectrogram.py │
   │ streaming│ │ updates │ └────────────────┘
   └──────────┘ └─────────┘
```

**services:**
| service | technology | purpose |
|---------|------------|---------|
| birdnet-api | Go | REST API, WebSocket, task scheduling |
| birdnet_analysis | Python | ML inference with BirdNET model |
| birdnet-recording | Go | Audio capture from microphone or RTSP (replaces shell scripts) |
| spectrogram_viewer | Python | Generates live spectrogram images |
| livestream | Python | Streams audio to Icecast via ffmpeg |
| birdnet_stats | Python/Streamlit | Statistics dashboard |

## prerequisites

### hardware

- **raspberry pi 5** (recommended), 4b, 400, 3b+, or 0w2
- 4GB+ RAM recommended (8GB for best performance)
- NVMe SSD recommended for Pi 5 (improves database and recording I/O)
- 64-bit raspios (trixie recommended)
- usb microphone or sound card
- x86_64 linux also works for development

### software

| dependency | minimum version | source |
|-----------|----------------|--------|
| Go | 1.21+ | `go.mod` |
| Node.js | 18+ | Vite 5 requires it |
| Python | 3.9+ | `pyproject.toml` |

### system packages

installed via `apt` on the pi (see `scripts/install/install_services.sh`):

- **caddy** — reverse proxy
- **sqlite3** — database
- **ffmpeg** — audio processing and streaming
- **alsa-utils** — microphone input (arecord)
- **sox**, **libsox-fmt-mp3** — audio conversion
- **pulseaudio** — audio routing
- **icecast2** — live audio streaming
- **avahi-utils** — mDNS hostname resolution
- **python3-pip**, **python3-venv** — python package management
- **inotify-tools** — file change monitoring

## installation

fresh install - use the upstream installer, then pull this branch:

```bash
# install base birdnet-pi first
curl -s https://raw.githubusercontent.com/Nachtzuster/BirdNET-Pi/main/newinstaller.sh | bash

# switch to this fork
cd ~/BirdNET-Pi
git remote set-url origin https://github.com/addlatt/BirdNET-Pi-fork.git
git fetch origin
git checkout main
git pull

# build go server
make build

# build preact app (optional - pre-built in web/dist)
cd web && npm install && npm run build && cd ..

# install and start the go service
bash deployment/install-api-service.sh
```

### cross-compilation

build for pi from another machine:

```bash
make build-arm64    # pi 3/4/5 (64-bit)
make build-arm      # pi zero/older (32-bit)
make build-pi       # alias for arm64
```

## usage

access from any browser on your network:
- `http://birdnetpi.local` or your pi's ip
- preact ui at `/`

### service management

```bash
# check all services
systemctl status birdnet-api spectrogram_viewer livestream

# view api logs
sudo journalctl -u birdnet-api -f

# restart everything
sudo systemctl restart birdnet-api spectrogram_viewer livestream
```

the go server auto-restarts if it crashes. rate limited to 5 restarts per minute.

## development

see [CLAUDE.md](CLAUDE.md) for the full dev guide.

### developer setup

```bash
# clone and set up everything (checks prereqs, installs deps, builds)
git clone https://github.com/addlatt/BirdNET-Pi-fork.git
cd BirdNET-Pi-fork
make install

# run tests
make test
```

### deploy to pi

```bash
# make changes locally, commit and push
git add -A && git commit -m "message" && git push

# on pi: pull, rebuild, restart
ssh user@birdnet "cd ~/BirdNET-Pi && git pull && make build && sudo systemctl restart birdnet-api"
```

## api

### REST endpoints

```
GET  /api/health              # health check
GET  /api/detections          # list detections (paginated)
GET  /api/species             # species with counts
GET  /api/settings            # current config
PUT  /api/settings            # update config
GET  /api/services            # service statuses
POST /api/services/{name}/{action}  # start/stop/restart services
GET  /api/diagnostics/disk    # disk usage info
GET  /api/diagnostics/system  # system info
GET  /api/logs/recent         # recent log entries
```

### WebSocket endpoints

```
WS   /ws                      # live detection updates
WS   /ws/logs                 # streaming log output
WS   /ws/logs/detections      # detection-only log stream
```


## license

same as upstream - [CC BY-NC-SA 4.0](LICENSE)

**you cannot use this for commercial products.**

## credits

- [kahst/BirdNET-Analyzer](https://github.com/kahst/BirdNET-Analyzer) - the ml model
- [mcguirepr89/BirdNET-Pi](https://github.com/mcguirepr89/BirdNET-Pi) - original project
- [Nachtzuster/BirdNET-Pi](https://github.com/Nachtzuster/BirdNET-Pi) - maintained fork this builds on
