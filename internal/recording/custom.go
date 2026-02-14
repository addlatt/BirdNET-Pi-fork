// Package recording — custom.go implements time-windowed audio recording.
// It replaces the legacy scripts/runtime/custom_recording.sh with native Go process management.
package recording

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// TimeWindow defines a recording window as a range of hours [StartHour, EndHour].
// Both bounds are inclusive. A window where StartHour == EndHour records only during that hour.
type TimeWindow struct {
	StartHour int // 0-23
	EndHour   int // 0-23
}

// DefaultWindows are the original hardcoded time windows from custom_recording.sh.
var DefaultWindows = []TimeWindow{
	{StartHour: 0, EndHour: 0},
	{StartHour: 3, EndHour: 7},
	{StartHour: 15, EndHour: 19},
	{StartHour: 22, EndHour: 23},
}

// CustomRecordingConfig holds parameters for time-windowed recording.
type CustomRecordingConfig struct {
	Windows           []TimeWindow  // Time windows to record in
	RecordingDuration int           // Seconds to record per chunk (default 60)
	PauseDuration     int           // Seconds to pause between chunks (default 240)
	Channels          int           // Audio channels
	RecCard           string        // Sound card device (empty for default)
	ExtractedDir      string        // Base extracted directory (e.g. ~/BirdSongs/Extracted)
}

// CustomConfigFromManager extracts custom recording config from a RecordingConfig.
func CustomConfigFromManager(cfg RecordingConfig) CustomRecordingConfig {
	channels := cfg.Channels
	if channels <= 0 {
		channels = 2
	}
	return CustomRecordingConfig{
		Windows:           DefaultWindows,
		RecordingDuration: 60,
		PauseDuration:     240,
		Channels:          channels,
		RecCard:           cfg.RecCard,
		ExtractedDir:      filepath.Join(cfg.RecsDir, "Extracted"),
	}
}

// inWindow reports whether the given hour falls within any configured time window.
func inWindow(hour int, windows []TimeWindow) bool {
	for _, w := range windows {
		if hour >= w.StartHour && hour <= w.EndHour {
			return true
		}
	}
	return false
}

// nowFunc is overridable for testing.
var nowFunc = time.Now

// RunCustom starts time-windowed recording and blocks until the context is cancelled.
// It records audio in configured time windows, sleeping between chunks and
// checking windows each iteration.
func RunCustom(ctx context.Context, cfg CustomRecordingConfig) error {
	rawDir := filepath.Join(cfg.ExtractedDir, "Raw")
	if err := os.MkdirAll(rawDir, 0755); err != nil {
		return fmt.Errorf("creating raw directory: %w", err)
	}

	if err := ensurePulseAudio(ctx); err != nil {
		log.Printf("Warning: PulseAudio check failed: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		hour := nowFunc().Hour()
		if !inWindow(hour, cfg.Windows) {
			// Not in any window — sleep briefly and re-check.
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(30 * time.Second):
			}
			continue
		}

		// Record one chunk.
		log.Printf("Custom recording: hour %d is in window, recording %ds", hour, cfg.RecordingDuration)
		args := buildCustomArecordArgs(cfg, rawDir)
		cmd := exec.CommandContext(ctx, "arecord", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("Custom recording arecord exited: %v", err)
		}

		// Pause between recordings.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(cfg.PauseDuration) * time.Second):
		}
	}
}

// buildCustomArecordArgs constructs the arecord command-line arguments for custom recording.
// Output goes to Raw/<Month-Year>/<Day-Weekday>/<Date>-birdnet-<Time>.wav
// matching the original shell script's strftime pattern.
func buildCustomArecordArgs(cfg CustomRecordingConfig, rawDir string) []string {
	// Original shell pattern: ${EXTRACTED}/Raw/%B-%Y/%d-%A/%F-birdnet-${STAMP}.wav
	// %B = month name, %Y = year, %d = day of month, %A = weekday name, %F = YYYY-MM-DD, %H:%M:%S = time
	outputPattern := filepath.Join(rawDir, "%B-%Y", "%d-%A", "%F-birdnet-%H:%M:%S.wav")

	args := []string{
		"-f", "S16_LE",
		fmt.Sprintf("-c%d", cfg.Channels),
		"-r48000",
		"-t", "wav",
		"-d", fmt.Sprintf("%d", cfg.RecordingDuration),
		"--use-strftime", outputPattern,
	}

	if cfg.RecCard != "" && cfg.RecCard != "default" {
		args = append(args, "-D", cfg.RecCard)
	}

	return args
}
