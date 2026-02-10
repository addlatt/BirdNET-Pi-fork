package api

import (
	"github.com/birdnet-pi/birdnet/internal/config"
	"github.com/birdnet-pi/birdnet/internal/db"
	"github.com/birdnet-pi/birdnet/internal/mlclient"
	"github.com/birdnet-pi/birdnet/internal/monitor"
	"github.com/birdnet-pi/birdnet/internal/scheduler"
	"github.com/birdnet-pi/birdnet/internal/ws"
)

// Handlers contains all API handler dependencies.
type Handlers struct {
	db           *db.DB
	hub          *ws.Hub
	monitor      *monitor.MemoryMonitor
	mlClient     *mlclient.Client
	configMgr    *config.Manager
	scheduler    *scheduler.Scheduler
	taskHistory  *scheduler.HistoryStore
	scriptsDir   string
	dataDir      string
	birdsongsDir string
	homeDir      string
}

// NewHandlers creates a new Handlers instance with all dependencies.
func NewHandlers(db *db.DB, hub *ws.Hub, monitor *monitor.MemoryMonitor, mlClient *mlclient.Client, configMgr *config.Manager, scriptsDir, dataDir, birdsongsDir, homeDir string) *Handlers {
	return &Handlers{
		db:           db,
		hub:          hub,
		monitor:      monitor,
		mlClient:     mlClient,
		configMgr:    configMgr,
		scriptsDir:   scriptsDir,
		dataDir:      dataDir,
		birdsongsDir: birdsongsDir,
		homeDir:      homeDir,
	}
}

// SetScheduler sets the scheduler dependency.
// This is called after NewHandlers because the scheduler depends on handlers existing.
func (h *Handlers) SetScheduler(s *scheduler.Scheduler, history *scheduler.HistoryStore) {
	h.scheduler = s
	h.taskHistory = history
}
