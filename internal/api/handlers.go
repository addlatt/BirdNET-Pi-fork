package api

import (
	"github.com/birdnet-pi/birdnet/internal/db"
	"github.com/birdnet-pi/birdnet/internal/mlclient"
	"github.com/birdnet-pi/birdnet/internal/monitor"
	"github.com/birdnet-pi/birdnet/internal/ws"
)

// Handlers contains all API handler dependencies.
type Handlers struct {
	db           *db.DB
	hub          *ws.Hub
	monitor      *monitor.MemoryMonitor
	mlClient     *mlclient.Client
	scriptsDir   string
	dataDir      string
	birdsongsDir string
}

// NewHandlers creates a new Handlers instance with all dependencies.
func NewHandlers(db *db.DB, hub *ws.Hub, monitor *monitor.MemoryMonitor, mlClient *mlclient.Client, scriptsDir, dataDir, birdsongsDir string) *Handlers {
	return &Handlers{
		db:           db,
		hub:          hub,
		monitor:      monitor,
		mlClient:     mlClient,
		scriptsDir:   scriptsDir,
		dataDir:      dataDir,
		birdsongsDir: birdsongsDir,
	}
}
