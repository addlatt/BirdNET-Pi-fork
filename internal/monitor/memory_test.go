package monitor

import (
	"testing"
)

func TestNewMemoryMonitor(t *testing.T) {
	m := NewMemoryMonitor()
	if m == nil {
		t.Fatal("NewMemoryMonitor returned nil")
	}
	if m.components == nil {
		t.Error("components map is nil")
	}
}

func TestMemoryMonitor_Register(t *testing.T) {
	m := NewMemoryMonitor()

	// Register a component
	m.Register("test", func() uint64 { return 1024 })

	usage := m.GetUsage()
	if val, ok := usage["test"]; !ok {
		t.Error("registered component not found in usage")
	} else if val != 1024 {
		t.Errorf("usage[test] = %d, want 1024", val)
	}
}

func TestMemoryMonitor_Unregister(t *testing.T) {
	m := NewMemoryMonitor()

	m.Register("test", func() uint64 { return 1024 })
	m.Unregister("test")

	usage := m.GetUsage()
	if _, ok := usage["test"]; ok {
		t.Error("unregistered component still in usage")
	}
}

func TestMemoryMonitor_GetUsage(t *testing.T) {
	m := NewMemoryMonitor()

	// Register multiple components
	m.Register("component1", func() uint64 { return 100 })
	m.Register("component2", func() uint64 { return 200 })
	m.Register("component3", func() uint64 { return 300 })

	usage := m.GetUsage()

	if len(usage) != 3 {
		t.Errorf("len(usage) = %d, want 3", len(usage))
	}

	tests := map[string]uint64{
		"component1": 100,
		"component2": 200,
		"component3": 300,
	}

	for name, want := range tests {
		if got := usage[name]; got != want {
			t.Errorf("usage[%s] = %d, want %d", name, got, want)
		}
	}
}

func TestMemoryMonitor_GetTotal(t *testing.T) {
	m := NewMemoryMonitor()

	// Empty monitor
	if total := m.GetTotal(); total != 0 {
		t.Errorf("empty GetTotal() = %d, want 0", total)
	}

	// Register components
	m.Register("c1", func() uint64 { return 100 })
	m.Register("c2", func() uint64 { return 200 })

	if total := m.GetTotal(); total != 300 {
		t.Errorf("GetTotal() = %d, want 300", total)
	}
}

func TestMemoryMonitor_GetStats(t *testing.T) {
	m := NewMemoryMonitor()
	m.Register("test", func() uint64 { return 500 })

	stats := m.GetStats()

	if stats == nil {
		t.Fatal("GetStats returned nil")
	}

	// Go runtime stats should be populated
	if stats.GoNumGoroutine <= 0 {
		t.Error("GoNumGoroutine should be > 0")
	}

	// Component stats
	if stats.Components == nil {
		t.Error("Components map is nil")
	}
	if val, ok := stats.Components["test"]; !ok || val != 500 {
		t.Errorf("Components[test] = %d, want 500", val)
	}

	// Total component memory
	if stats.TotalComponentMemory != 500 {
		t.Errorf("TotalComponentMemory = %d, want 500", stats.TotalComponentMemory)
	}
}

func TestMemoryMonitor_ShouldUnloadLLM(t *testing.T) {
	m := NewMemoryMonitor()

	tests := []struct {
		name      string
		threshold uint64
		// We can't predict system memory, but we can test the function doesn't panic
	}{
		{name: "low threshold", threshold: 0},
		{name: "high threshold", threshold: 1024 * 1024 * 1024 * 100}, // 100GB - should always return true
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just ensure it doesn't panic
			_ = m.ShouldUnloadLLM(tt.threshold)
		})
	}
}

func TestMemoryMonitor_ConcurrentAccess(t *testing.T) {
	m := NewMemoryMonitor()

	// Test concurrent registration and usage
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			m.Register("concurrent", func() uint64 { return uint64(i) })
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			_ = m.GetUsage()
			_ = m.GetTotal()
		}
		done <- true
	}()

	<-done
	<-done
}

func TestMemoryStats_Fields(t *testing.T) {
	stats := &MemoryStats{
		GoHeapAlloc:          1000,
		GoHeapSys:            2000,
		GoHeapInUse:          1500,
		GoStackInUse:         500,
		GoTotalAlloc:         3000,
		GoNumGoroutine:       10,
		SystemTotal:          8000000000,
		SystemFree:           4000000000,
		SystemAvailable:      5000000000,
		SystemUsed:           3000000000,
		Components:           map[string]uint64{"test": 100},
		TotalComponentMemory: 100,
	}

	// Verify all fields are accessible
	if stats.GoHeapAlloc != 1000 {
		t.Errorf("GoHeapAlloc = %d, want 1000", stats.GoHeapAlloc)
	}
	if stats.GoNumGoroutine != 10 {
		t.Errorf("GoNumGoroutine = %d, want 10", stats.GoNumGoroutine)
	}
	if stats.SystemTotal != 8000000000 {
		t.Errorf("SystemTotal = %d, want 8000000000", stats.SystemTotal)
	}
	if stats.Components["test"] != 100 {
		t.Errorf("Components[test] = %d, want 100", stats.Components["test"])
	}
}

func BenchmarkGetUsage(b *testing.B) {
	m := NewMemoryMonitor()
	m.Register("c1", func() uint64 { return 100 })
	m.Register("c2", func() uint64 { return 200 })
	m.Register("c3", func() uint64 { return 300 })

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.GetUsage()
	}
}

func BenchmarkGetStats(b *testing.B) {
	m := NewMemoryMonitor()
	m.Register("test", func() uint64 { return 100 })

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.GetStats()
	}
}
