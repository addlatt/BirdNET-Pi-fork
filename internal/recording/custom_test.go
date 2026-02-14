package recording

import (
	"path/filepath"
	"testing"
)

func TestInWindow(t *testing.T) {
	windows := []TimeWindow{
		{StartHour: 0, EndHour: 0},
		{StartHour: 3, EndHour: 7},
		{StartHour: 15, EndHour: 19},
		{StartHour: 22, EndHour: 23},
	}

	tests := []struct {
		hour int
		want bool
	}{
		{0, true},   // window 0-0
		{1, false},  // gap
		{2, false},  // gap
		{3, true},   // window 3-7
		{5, true},   // window 3-7
		{7, true},   // window 3-7 (inclusive)
		{8, false},  // gap
		{14, false}, // gap
		{15, true},  // window 15-19
		{17, true},  // window 15-19
		{19, true},  // window 15-19 (inclusive)
		{20, false}, // gap
		{21, false}, // gap
		{22, true},  // window 22-23
		{23, true},  // window 22-23 (inclusive)
	}

	for _, tt := range tests {
		got := inWindow(tt.hour, windows)
		if got != tt.want {
			t.Errorf("inWindow(%d) = %v, want %v", tt.hour, got, tt.want)
		}
	}
}

func TestInWindowEmpty(t *testing.T) {
	if inWindow(12, nil) {
		t.Error("inWindow with nil windows should return false")
	}
	if inWindow(12, []TimeWindow{}) {
		t.Error("inWindow with empty windows should return false")
	}
}

func TestBuildCustomArecordArgs(t *testing.T) {
	cfg := CustomRecordingConfig{
		Channels:          2,
		RecordingDuration: 60,
		RecCard:           "",
	}

	args := buildCustomArecordArgs(cfg, "/tmp/raw")

	assertContains(t, args, "-f")
	assertContains(t, args, "S16_LE")
	assertContains(t, args, "-c2")
	assertContains(t, args, "-r48000")
	assertContains(t, args, "-t")
	assertContains(t, args, "wav")
	assertContains(t, args, "-d")
	assertContains(t, args, "60")
	assertContains(t, args, "--use-strftime")

	// No -D when RecCard is empty
	assertNotContains(t, args, "-D")

	// Verify output pattern includes strftime directories
	last := args[len(args)-1]
	if !containsStr(last, "%B-%Y") {
		t.Errorf("output pattern should contain %%B-%%Y, got %q", last)
	}
	if !containsStr(last, "%d-%A") {
		t.Errorf("output pattern should contain %%d-%%A, got %q", last)
	}
	if !containsStr(last, "birdnet-%H:%M:%S.wav") {
		t.Errorf("output pattern should contain birdnet-%%H:%%M:%%S.wav, got %q", last)
	}
}

func TestBuildCustomArecordArgsWithDevice(t *testing.T) {
	cfg := CustomRecordingConfig{
		Channels:          1,
		RecordingDuration: 30,
		RecCard:           "hw:1,0",
	}

	args := buildCustomArecordArgs(cfg, "/tmp/raw")

	assertContains(t, args, "-c1")
	assertContains(t, args, "30")
	assertContains(t, args, "-D")
	assertContains(t, args, "hw:1,0")
}

func TestBuildCustomArecordArgsDefaultDevice(t *testing.T) {
	cfg := CustomRecordingConfig{
		Channels:          2,
		RecordingDuration: 60,
		RecCard:           "default",
	}

	args := buildCustomArecordArgs(cfg, "/tmp/raw")

	// "default" device should not add -D flag
	assertNotContains(t, args, "-D")
}

func TestCustomConfigFromManager(t *testing.T) {
	rc := RecordingConfig{
		Channels: 1,
		RecCard:  "hw:0",
		RecsDir:  "/home/pi/BirdSongs",
	}

	cfg := CustomConfigFromManager(rc)

	if cfg.Channels != 1 {
		t.Errorf("Channels = %d, want 1", cfg.Channels)
	}
	if cfg.RecCard != "hw:0" {
		t.Errorf("RecCard = %q, want %q", cfg.RecCard, "hw:0")
	}
	if cfg.RecordingDuration != 60 {
		t.Errorf("RecordingDuration = %d, want 60", cfg.RecordingDuration)
	}
	if cfg.PauseDuration != 240 {
		t.Errorf("PauseDuration = %d, want 240", cfg.PauseDuration)
	}
	if len(cfg.Windows) != 4 {
		t.Fatalf("Windows = %d, want 4", len(cfg.Windows))
	}
	if cfg.Windows[1].StartHour != 3 || cfg.Windows[1].EndHour != 7 {
		t.Errorf("Windows[1] = %v, want {3, 7}", cfg.Windows[1])
	}
	want := filepath.Join("/home/pi/BirdSongs", "Extracted")
	if cfg.ExtractedDir != want {
		t.Errorf("ExtractedDir = %q, want %q", cfg.ExtractedDir, want)
	}
}

func TestCustomConfigFromManagerDefaults(t *testing.T) {
	rc := RecordingConfig{} // Zero values

	cfg := CustomConfigFromManager(rc)

	if cfg.Channels != 2 {
		t.Errorf("default Channels = %d, want 2", cfg.Channels)
	}
}

func TestDefaultWindows(t *testing.T) {
	// Verify DefaultWindows match the original shell script
	expected := []TimeWindow{
		{0, 0},
		{3, 7},
		{15, 19},
		{22, 23},
	}

	if len(DefaultWindows) != len(expected) {
		t.Fatalf("DefaultWindows len = %d, want %d", len(DefaultWindows), len(expected))
	}

	for i, w := range DefaultWindows {
		if w.StartHour != expected[i].StartHour || w.EndHour != expected[i].EndHour {
			t.Errorf("DefaultWindows[%d] = %v, want %v", i, w, expected[i])
		}
	}
}
