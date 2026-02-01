package tasks

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"time"
)

// CoreServices lists the core BirdNET-Pi services.
var CoreServices = []string{
	"birdnet_recording.service",
	"birdnet_analysis.service",
	"chart_viewer.service",
	"spectrogram_viewer.service",
}

// AllServices lists all BirdNET-Pi services.
var AllServices = []string{
	"chart_viewer.service",
	"spectrogram_viewer.service",
	"icecast2.service",
	"birdnet_recording.service",
	"birdnet_analysis.service",
	"birdnet_log.service",
	"birdnet_stats.service",
}

// RestartServicesTask restarts all BirdNET-Pi services.
// This replaces the restart_services.sh script.
type RestartServicesTask struct{}

// NewRestartServicesTask creates a new restart services task.
func NewRestartServicesTask() *RestartServicesTask {
	return &RestartServicesTask{}
}

func (t *RestartServicesTask) Name() string {
	return "restart_services"
}

func (t *RestartServicesTask) Description() string {
	return "Restarts all BirdNET-Pi services"
}

func (t *RestartServicesTask) DefaultSchedule() string {
	return "" // Manual only
}

func (t *RestartServicesTask) Timeout() time.Duration {
	return 5 * time.Minute
}

func (t *RestartServicesTask) Run(ctx context.Context) error {
	log.Println("Restart services: stopping recording service first")

	// Stop recording service first
	if err := stopService(ctx, "birdnet_recording.service"); err != nil {
		log.Printf("Restart services: warning - failed to stop recording: %v", err)
	}

	// Restart all services
	for _, svc := range AllServices {
		log.Printf("Restart services: restarting %s", svc)
		if err := restartService(ctx, svc); err != nil {
			log.Printf("Restart services: warning - failed to restart %s: %v", svc, err)
		}
	}

	// Wait for analysis service to be running
	log.Println("Restart services: waiting for analysis service to start")
	for i := 0; i < 5; i++ {
		if isServiceActive(ctx, "birdnet_analysis.service") {
			log.Println("Restart services: analysis service is running")
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}

	log.Println("Restart services: completed")
	return nil
}

// StopCoreServicesTask stops the core BirdNET-Pi services.
// This replaces the stop_core_services.sh script.
type StopCoreServicesTask struct{}

// NewStopCoreServicesTask creates a new stop core services task.
func NewStopCoreServicesTask() *StopCoreServicesTask {
	return &StopCoreServicesTask{}
}

func (t *StopCoreServicesTask) Name() string {
	return "stop_core_services"
}

func (t *StopCoreServicesTask) Description() string {
	return "Stops core BirdNET-Pi services (recording, analysis, viewers)"
}

func (t *StopCoreServicesTask) DefaultSchedule() string {
	return "" // Manual only
}

func (t *StopCoreServicesTask) Timeout() time.Duration {
	return 2 * time.Minute
}

func (t *StopCoreServicesTask) Run(ctx context.Context) error {
	log.Println("Stop core services: stopping services")

	for _, svc := range CoreServices {
		log.Printf("Stop core services: stopping %s", svc)
		if err := stopService(ctx, svc); err != nil {
			log.Printf("Stop core services: warning - failed to stop %s: %v", svc, err)
		}
	}

	// Also stop custom_recording if it exists
	stopService(ctx, "custom_recording.service")

	log.Println("Stop core services: completed")
	return nil
}

// Helper functions

func stopService(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "sudo", "systemctl", "stop", name)
	return cmd.Run()
}

func restartService(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "sudo", "systemctl", "restart", name)
	return cmd.Run()
}

func isServiceActive(ctx context.Context, name string) bool {
	cmd := exec.CommandContext(ctx, "systemctl", "is-active", "--quiet", name)
	return cmd.Run() == nil
}

// ServiceController provides service control operations.
// This can be used directly by API handlers.
type ServiceController struct{}

// NewServiceController creates a new service controller.
func NewServiceController() *ServiceController {
	return &ServiceController{}
}

// RestartAll restarts all BirdNET-Pi services.
func (c *ServiceController) RestartAll(ctx context.Context) error {
	task := NewRestartServicesTask()
	return task.Run(ctx)
}

// StopCore stops the core BirdNET-Pi services.
func (c *ServiceController) StopCore(ctx context.Context) error {
	task := NewStopCoreServicesTask()
	return task.Run(ctx)
}

// Restart restarts a specific service.
func (c *ServiceController) Restart(ctx context.Context, serviceName string) error {
	if !isValidServiceName(serviceName) {
		return fmt.Errorf("invalid service name: %s", serviceName)
	}
	return restartService(ctx, serviceName)
}

// Stop stops a specific service.
func (c *ServiceController) Stop(ctx context.Context, serviceName string) error {
	if !isValidServiceName(serviceName) {
		return fmt.Errorf("invalid service name: %s", serviceName)
	}
	return stopService(ctx, serviceName)
}

// Start starts a specific service.
func (c *ServiceController) Start(ctx context.Context, serviceName string) error {
	if !isValidServiceName(serviceName) {
		return fmt.Errorf("invalid service name: %s", serviceName)
	}
	cmd := exec.CommandContext(ctx, "sudo", "systemctl", "start", serviceName)
	return cmd.Run()
}

// IsActive checks if a service is active.
func (c *ServiceController) IsActive(ctx context.Context, serviceName string) bool {
	return isServiceActive(ctx, serviceName)
}

func isValidServiceName(name string) bool {
	validServices := map[string]bool{
		"birdnet_recording.service":    true,
		"birdnet_analysis.service":     true,
		"birdnet_log.service":          true,
		"birdnet_stats.service":        true,
		"chart_viewer.service":         true,
		"spectrogram_viewer.service":   true,
		"livestream.service":           true,
		"web_terminal.service":         true,
		"icecast2.service":             true,
		"custom_recording.service":     true,
	}
	return validServices[name]
}
