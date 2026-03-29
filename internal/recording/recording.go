// Package recording manages audio recording from RTSP streams or local microphones.
// It replaces the legacy scripts/runtime/birdnet_recording.sh with native Go process management.
package recording

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/birdnet-pi/birdnet/internal/config"
)

// RecordingConfig holds the parameters needed for recording.
type RecordingConfig struct {
	RTSPStream      string // Comma-separated RTSP URLs (empty for microphone mode)
	RecordingLength int    // Chunk duration in seconds
	Channels        int    // Audio channels (microphone mode)
	RecCard         string // Sound card device (microphone mode, empty for default)
	LogLevel        string // ffmpeg/arecord log level
	RecsDir         string // Base recordings directory
}

// ConfigFromManager extracts recording config from a config.Config.
func ConfigFromManager(cfg *config.Config) RecordingConfig {
	logLevel := cfg.LogLevelBirdnetRecordingService
	if logLevel == "" {
		logLevel = "error"
	}
	recordingLength := cfg.RecordingLength
	if recordingLength <= 0 {
		recordingLength = 15
	}
	channels := cfg.Channels
	if channels <= 0 {
		channels = 2
	}
	return RecordingConfig{
		RTSPStream:      cfg.RTSPStream,
		RecordingLength: recordingLength,
		Channels:        channels,
		RecCard:         cfg.RecCard,
		LogLevel:        logLevel,
		RecsDir:         cfg.RecsDir,
	}
}

// Run starts recording and blocks until the context is cancelled.
// In RTSP mode, it launches one ffmpeg process per stream URL.
// In microphone mode, it launches arecord with PulseAudio.
func Run(ctx context.Context, cfg RecordingConfig) error {
	streamDir := filepath.Join(cfg.RecsDir, "StreamData")
	if err := os.MkdirAll(streamDir, 0755); err != nil {
		return fmt.Errorf("creating stream directory: %w", err)
	}

	if cfg.RTSPStream != "" {
		return runRTSP(ctx, cfg, streamDir)
	}
	return runMicrophone(ctx, cfg, streamDir)
}

// runRTSP launches one ffmpeg subprocess per RTSP stream URL.
// Each stream runs in a goroutine with automatic restart on failure.
func runRTSP(ctx context.Context, cfg RecordingConfig, streamDir string) error {
	streams := splitStreams(cfg.RTSPStream)
	if len(streams) == 0 {
		return fmt.Errorf("RTSP_STREAM is set but contains no valid URLs")
	}

	ffmpegVersion, err := getFFmpegMajorVersion(ctx)
	if err != nil {
		log.Printf("Warning: could not determine ffmpeg version, assuming >=5: %v", err)
		ffmpegVersion = 5
	}

	var wg sync.WaitGroup
	for i, url := range streams {
		wg.Add(1)
		go func(streamURL string, streamNum int) {
			defer wg.Done()
			loopFFmpeg(ctx, cfg, streamDir, streamURL, streamNum, ffmpegVersion)
		}(url, i+1)
	}

	wg.Wait()
	return ctx.Err()
}

// loopFFmpeg runs ffmpeg in a retry loop for a single RTSP stream.
// It restarts ffmpeg on failure until the context is cancelled.
func loopFFmpeg(ctx context.Context, cfg RecordingConfig, streamDir, streamURL string, streamNum, ffmpegVersion int) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		args := buildFFmpegArgs(cfg, streamDir, streamURL, streamNum, ffmpegVersion)
		log.Printf("Starting ffmpeg for stream %d: %s", streamNum, streamURL)

		cmd := exec.CommandContext(ctx, "ffmpeg", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			if ctx.Err() != nil {
				return // Context cancelled, graceful shutdown
			}
			log.Printf("ffmpeg stream %d exited: %v, restarting...", streamNum, err)
		}
	}
}

// buildFFmpegArgs constructs the ffmpeg command-line arguments for an RTSP stream.
func buildFFmpegArgs(cfg RecordingConfig, streamDir, streamURL string, streamNum, ffmpegVersion int) []string {
	outputPattern := filepath.Join(streamDir,
		fmt.Sprintf("%%F-birdnet-RTSP_%d-%%H:%%M:%%S.wav", streamNum))

	args := []string{
		"-hide_banner",
		"-loglevel", cfg.LogLevel,
		"-nostdin",
	}

	// Add timeout parameter based on URL scheme and ffmpeg version
	timeoutParam := buildTimeoutParam(streamURL, ffmpegVersion)
	if len(timeoutParam) > 0 {
		args = append(args, timeoutParam...)
	}

	args = append(args,
		"-i", streamURL,
		"-vn",
		"-map", "a:0",
		"-acodec", "pcm_s16le",
		"-ac", "2",
		"-ar", "48000",
		"-f", "segment",
		"-segment_format", "wav",
		"-segment_time", fmt.Sprintf("%d", cfg.RecordingLength),
		"-strftime", "1",
		outputPattern,
	)

	return args
}

// buildTimeoutParam returns the appropriate timeout argument for the stream URL scheme.
func buildTimeoutParam(streamURL string, ffmpegVersion int) []string {
	lower := strings.ToLower(streamURL)
	switch {
	case strings.HasPrefix(lower, "rtsp://") || strings.HasPrefix(lower, "rtsps://"):
		if ffmpegVersion < 5 {
			return []string{"-stimeout", "10000000"}
		}
		return []string{"-timeout", "10000000"}
	case strings.Contains(lower, "://"):
		// Other URL schemes (http, etc.)
		return []string{"-rw_timeout", "10000000"}
	default:
		return nil
	}
}

// runMicrophone starts recording from a local microphone via arecord.
func runMicrophone(ctx context.Context, cfg RecordingConfig, streamDir string) error {
	if err := ensurePulseAudio(ctx); err != nil {
		log.Printf("Warning: PulseAudio check failed: %v", err)
	}

	args := buildArecordArgs(cfg, streamDir)
	log.Printf("Starting arecord with %d channel(s)", cfg.Channels)

	cmd := exec.CommandContext(ctx, "arecord", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err() // Context cancelled, graceful shutdown
		}
		return fmt.Errorf("arecord exited: %w", err)
	}
	return nil
}

// buildArecordArgs constructs the arecord command-line arguments.
func buildArecordArgs(cfg RecordingConfig, streamDir string) []string {
	outputPattern := filepath.Join(streamDir, "%F-birdnet-%H:%M:%S.wav")

	args := []string{
		"-f", "S16_LE",
		fmt.Sprintf("-c%d", cfg.Channels),
		"-r48000",
		"-t", "wav",
		"--max-file-time", fmt.Sprintf("%d", cfg.RecordingLength),
		"--use-strftime", outputPattern,
	}

	if cfg.RecCard != "" && cfg.RecCard != "default" {
		args = append(args, "-D", cfg.RecCard)
	}

	return args
}

// ensurePulseAudio checks if a PulseAudio-compatible server (PulseAudio or
// PipeWire-Pulse) is running and starts native PulseAudio only if neither is found.
func ensurePulseAudio(ctx context.Context) error {
	check := exec.CommandContext(ctx, "pactl", "info")
	if err := check.Run(); err != nil {
		log.Printf("No PulseAudio-compatible server detected, starting PulseAudio...")
		start := exec.CommandContext(ctx, "pulseaudio", "--start")
		return start.Run()
	}
	return nil
}

// splitStreams splits a comma-separated list of stream URLs, trimming whitespace.
func splitStreams(s string) []string {
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// getFFmpegMajorVersion returns the major version number of the installed ffmpeg.
func getFFmpegMajorVersion(ctx context.Context) (int, error) {
	cmd := exec.CommandContext(ctx, "ffmpeg", "-version")
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	// Output format: "ffmpeg version N.x.x ..."
	var major int
	lines := strings.SplitN(string(out), "\n", 2)
	if len(lines) > 0 {
		fields := strings.Fields(lines[0])
		for i, f := range fields {
			if f == "version" && i+1 < len(fields) {
				versionParts := strings.SplitN(fields[i+1], ".", 2)
				if _, err := fmt.Sscanf(versionParts[0], "%d", &major); err == nil {
					return major, nil
				}
			}
		}
	}
	return 0, fmt.Errorf("could not parse ffmpeg version from output")
}
