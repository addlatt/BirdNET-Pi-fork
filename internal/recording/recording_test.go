package recording

import (
	"strings"
	"testing"

	"github.com/birdnet-pi/birdnet/internal/config"
)

func TestSplitStreams(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"single URL", "rtsp://host/stream", []string{"rtsp://host/stream"}},
		{"two URLs", "rtsp://a/1,rtsp://b/2", []string{"rtsp://a/1", "rtsp://b/2"}},
		{"with spaces", " rtsp://a/1 , rtsp://b/2 ", []string{"rtsp://a/1", "rtsp://b/2"}},
		{"empty string", "", nil},
		{"trailing comma", "rtsp://a/1,", []string{"rtsp://a/1"}},
		{"three URLs", "rtsp://a,http://b,rtsps://c", []string{"rtsp://a", "http://b", "rtsps://c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitStreams(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("splitStreams(%q) = %v (len %d), want %v (len %d)",
					tt.input, got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitStreams(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestBuildTimeoutParam(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		version int
		want    []string
	}{
		{"rtsp v4", "rtsp://host/stream", 4, []string{"-stimeout", "10000000"}},
		{"rtsp v5", "rtsp://host/stream", 5, []string{"-timeout", "10000000"}},
		{"rtsp v6", "rtsp://host/stream", 6, []string{"-timeout", "10000000"}},
		{"rtsps v5", "rtsps://host/stream", 5, []string{"-timeout", "10000000"}},
		{"rtsps v4", "rtsps://host/stream", 4, []string{"-stimeout", "10000000"}},
		{"http", "http://host/stream", 5, []string{"-rw_timeout", "10000000"}},
		{"no scheme", "/dev/video0", 5, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildTimeoutParam(tt.url, tt.version)
			if len(got) != len(tt.want) {
				t.Fatalf("buildTimeoutParam(%q, %d) = %v, want %v", tt.url, tt.version, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("buildTimeoutParam(%q, %d)[%d] = %q, want %q",
						tt.url, tt.version, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestBuildFFmpegArgs(t *testing.T) {
	cfg := RecordingConfig{
		LogLevel:        "error",
		RecordingLength: 15,
	}

	args := buildFFmpegArgs(cfg, "/tmp/stream", "rtsp://cam/feed", 1, 5)

	// Verify key arguments are present
	assertContains(t, args, "-hide_banner")
	assertContains(t, args, "-nostdin")
	assertContains(t, args, "-loglevel")
	assertContains(t, args, "error")
	assertContains(t, args, "-timeout")
	assertContains(t, args, "10000000")
	assertContains(t, args, "-i")
	assertContains(t, args, "rtsp://cam/feed")
	assertContains(t, args, "-vn")
	assertContains(t, args, "-acodec")
	assertContains(t, args, "pcm_s16le")
	assertContains(t, args, "-ac")
	assertContains(t, args, "2")
	assertContains(t, args, "-ar")
	assertContains(t, args, "48000")
	assertContains(t, args, "-f")
	assertContains(t, args, "segment")
	assertContains(t, args, "-segment_format")
	assertContains(t, args, "wav")
	assertContains(t, args, "-segment_time")
	assertContains(t, args, "15")
	assertContains(t, args, "-strftime")
	assertContains(t, args, "1")

	// Verify output pattern includes stream number
	last := args[len(args)-1]
	if !containsStr(last, "RTSP_1") {
		t.Errorf("output pattern should contain RTSP_1, got %q", last)
	}
}

func TestBuildFFmpegArgsOldVersion(t *testing.T) {
	cfg := RecordingConfig{
		LogLevel:        "info",
		RecordingLength: 30,
	}

	args := buildFFmpegArgs(cfg, "/tmp/stream", "rtsp://cam/feed", 2, 4)

	// Old ffmpeg should use -stimeout
	assertContains(t, args, "-stimeout")
	assertNotContains(t, args, "-timeout")
}

func TestBuildArecordArgs(t *testing.T) {
	cfg := RecordingConfig{
		Channels:        2,
		RecordingLength: 15,
		RecCard:         "",
	}

	args := buildArecordArgs(cfg, "/tmp/stream")

	assertContains(t, args, "-f")
	assertContains(t, args, "S16_LE")
	assertContains(t, args, "-c2")
	assertContains(t, args, "-r48000")
	assertContains(t, args, "-t")
	assertContains(t, args, "wav")
	assertContains(t, args, "--max-file-time")
	assertContains(t, args, "15")
	assertContains(t, args, "--use-strftime")

	// No -D when RecCard is empty
	assertNotContains(t, args, "-D")
}

func TestBuildArecordArgsWithDevice(t *testing.T) {
	cfg := RecordingConfig{
		Channels:        1,
		RecordingLength: 10,
		RecCard:         "hw:1,0",
	}

	args := buildArecordArgs(cfg, "/tmp/stream")

	assertContains(t, args, "-c1")
	assertContains(t, args, "-D")
	assertContains(t, args, "hw:1,0")
}

func TestBuildArecordArgsDefaultDevice(t *testing.T) {
	cfg := RecordingConfig{
		Channels:        2,
		RecordingLength: 15,
		RecCard:         "default",
	}

	args := buildArecordArgs(cfg, "/tmp/stream")

	// "default" device should not add -D flag (same as shell script behavior)
	assertNotContains(t, args, "-D")
}

func TestConfigFromManager(t *testing.T) {
	cfg := &config.Config{
		RTSPStream:                      "rtsp://a,rtsp://b",
		RecordingLength:                 20,
		Channels:                        1,
		RecCard:                         "hw:0",
		LogLevelBirdnetRecordingService: "debug",
		RecsDir:                         "/home/pi/BirdSongs",
	}

	rc := ConfigFromManager(cfg)

	if rc.RTSPStream != "rtsp://a,rtsp://b" {
		t.Errorf("RTSPStream = %q, want %q", rc.RTSPStream, "rtsp://a,rtsp://b")
	}
	if rc.RecordingLength != 20 {
		t.Errorf("RecordingLength = %d, want 20", rc.RecordingLength)
	}
	if rc.Channels != 1 {
		t.Errorf("Channels = %d, want 1", rc.Channels)
	}
	if rc.RecCard != "hw:0" {
		t.Errorf("RecCard = %q, want %q", rc.RecCard, "hw:0")
	}
	if rc.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", rc.LogLevel, "debug")
	}
	if rc.RecsDir != "/home/pi/BirdSongs" {
		t.Errorf("RecsDir = %q, want %q", rc.RecsDir, "/home/pi/BirdSongs")
	}
}

func TestConfigFromManagerDefaults(t *testing.T) {
	cfg := &config.Config{} // All zero values

	rc := ConfigFromManager(cfg)

	if rc.LogLevel != "error" {
		t.Errorf("default LogLevel = %q, want %q", rc.LogLevel, "error")
	}
	if rc.RecordingLength != 15 {
		t.Errorf("default RecordingLength = %d, want 15", rc.RecordingLength)
	}
	if rc.Channels != 2 {
		t.Errorf("default Channels = %d, want 2", rc.Channels)
	}
}

// helpers

func assertContains(t *testing.T, args []string, want string) {
	t.Helper()
	if !contains(args, want) {
		t.Errorf("args %v should contain %q", args, want)
	}
}

func assertNotContains(t *testing.T, args []string, unwanted string) {
	t.Helper()
	if contains(args, unwanted) {
		t.Errorf("args %v should not contain %q", args, unwanted)
	}
}

func contains[T comparable](slice []T, item T) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func containsStr(s, substr string) bool {
	return strings.Contains(s, substr)
}
