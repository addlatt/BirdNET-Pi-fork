package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/birdnet-pi/birdnet/internal/config"
	"github.com/birdnet-pi/birdnet/internal/recording"
)

func main() {
	mode := flag.String("mode", "standard", "Recording mode: 'standard' (RTSP/microphone) or 'custom' (time-windowed)")
	flag.Parse()

	if *mode != "standard" && *mode != "custom" {
		fmt.Fprintf(os.Stderr, "Error: --mode must be 'standard' or 'custom', got %q\n", *mode)
		os.Exit(1)
	}

	// Load configuration
	configPath := getEnv("BIRDNET_CONFIG", config.DefaultConfigPath)
	homeDir := getEnv("HOME", expandHome("~"))

	configMgr := config.NewManager(configPath, homeDir)
	if err := configMgr.Load(); err != nil {
		log.Fatalf("Failed to load configuration from %s: %v", configPath, err)
	}

	cfg := recording.ConfigFromManager(configMgr.Get())

	// Set up context with signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-quit
		log.Printf("Received signal %v, shutting down...", sig)
		cancel()
	}()

	switch *mode {
	case "standard":
		if cfg.RTSPStream != "" {
			log.Printf("Starting standard recording (RTSP mode, streams: %s)", cfg.RTSPStream)
		} else {
			device := cfg.RecCard
			if device == "" {
				device = "default"
			}
			log.Printf("Starting standard recording (microphone mode, device: %s, channels: %d)", device, cfg.Channels)
		}

		if err := recording.Run(ctx, cfg); err != nil && ctx.Err() == nil {
			log.Fatalf("Recording failed: %v", err)
		}

	case "custom":
		customCfg := recording.CustomConfigFromManager(cfg)
		log.Printf("Starting custom recording (windows: %d, record: %ds, pause: %ds)",
			len(customCfg.Windows), customCfg.RecordingDuration, customCfg.PauseDuration)

		if err := recording.RunCustom(ctx, customCfg); err != nil && ctx.Err() == nil {
			log.Fatalf("Custom recording failed: %v", err)
		}
	}

	log.Println("Recording stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func expandHome(path string) string {
	if len(path) > 1 && path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home + path[1:]
	}
	return path
}
