package monitor

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// MemoryMonitor tracks memory usage across components.
type MemoryMonitor struct {
	components map[string]func() uint64
	mu         sync.RWMutex
}

// MemoryStats contains memory usage statistics.
type MemoryStats struct {
	// Go runtime memory stats
	GoHeapAlloc    uint64 `json:"go_heap_alloc"`
	GoHeapSys      uint64 `json:"go_heap_sys"`
	GoHeapInUse    uint64 `json:"go_heap_in_use"`
	GoStackInUse   uint64 `json:"go_stack_in_use"`
	GoTotalAlloc   uint64 `json:"go_total_alloc"`
	GoNumGoroutine int    `json:"go_num_goroutine"`

	// System memory stats (from /proc/meminfo)
	SystemTotal     uint64 `json:"system_total"`
	SystemFree      uint64 `json:"system_free"`
	SystemAvailable uint64 `json:"system_available"`
	SystemUsed      uint64 `json:"system_used"`

	// Component-specific memory (registered getters)
	Components map[string]uint64 `json:"components"`

	// Calculated totals
	TotalComponentMemory uint64 `json:"total_component_memory"`
}

// NewMemoryMonitor creates a new MemoryMonitor instance.
func NewMemoryMonitor() *MemoryMonitor {
	return &MemoryMonitor{
		components: make(map[string]func() uint64),
	}
}

// Register adds a component memory getter function.
func (m *MemoryMonitor) Register(name string, getter func() uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.components[name] = getter
}

// Unregister removes a component memory getter.
func (m *MemoryMonitor) Unregister(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.components, name)
}

// GetUsage returns memory usage for all registered components.
func (m *MemoryMonitor) GetUsage() map[string]uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	usage := make(map[string]uint64)
	for name, getter := range m.components {
		usage[name] = getter()
	}
	return usage
}

// GetTotal returns total memory usage across all components.
func (m *MemoryMonitor) GetTotal() uint64 {
	usage := m.GetUsage()
	var total uint64
	for _, mem := range usage {
		total += mem
	}
	return total
}

// GetStats returns comprehensive memory statistics.
func (m *MemoryMonitor) GetStats() *MemoryStats {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	stats := &MemoryStats{
		// Go runtime stats
		GoHeapAlloc:    memStats.HeapAlloc,
		GoHeapSys:      memStats.HeapSys,
		GoHeapInUse:    memStats.HeapInuse,
		GoStackInUse:   memStats.StackInuse,
		GoTotalAlloc:   memStats.TotalAlloc,
		GoNumGoroutine: runtime.NumGoroutine(),

		// Component stats
		Components: m.GetUsage(),
	}

	// Calculate total component memory
	for _, mem := range stats.Components {
		stats.TotalComponentMemory += mem
	}

	// Get system memory stats (Linux-specific)
	if sysStats := getSystemMemory(); sysStats != nil {
		stats.SystemTotal = sysStats.Total
		stats.SystemFree = sysStats.Free
		stats.SystemAvailable = sysStats.Available
		stats.SystemUsed = sysStats.Used
	}

	return stats
}

// ShouldUnloadLLM checks if LLM should be unloaded based on memory threshold.
// Part 2: Used for memory-aware model management.
func (m *MemoryMonitor) ShouldUnloadLLM(threshold uint64) bool {
	stats := m.GetStats()
	if stats.SystemAvailable == 0 {
		// Couldn't get system memory, be conservative
		return false
	}
	return stats.SystemAvailable < threshold
}

// systemMemory holds parsed /proc/meminfo values.
type systemMemory struct {
	Total     uint64
	Free      uint64
	Available uint64
	Used      uint64
}

// getSystemMemory reads memory info from /proc/meminfo (Linux-specific).
func getSystemMemory() *systemMemory {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return nil
	}

	mem := &systemMemory{}
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		// Values in /proc/meminfo are in kB
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		value *= 1024 // Convert to bytes

		switch fields[0] {
		case "MemTotal:":
			mem.Total = value
		case "MemFree:":
			mem.Free = value
		case "MemAvailable:":
			mem.Available = value
		}
	}

	mem.Used = mem.Total - mem.Available
	return mem
}
