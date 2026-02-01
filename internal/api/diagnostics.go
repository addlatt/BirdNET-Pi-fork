package api

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// DiskUsageResponse represents disk usage information.
type DiskUsageResponse struct {
	Size      string `json:"size"`
	Used      string `json:"used"`
	Available string `json:"available"`
	UsedPct   int    `json:"used_percent"`
	Path      string `json:"path"`
}

// MostRecentResponse represents the most recent detection.
type MostRecentResponse struct {
	Species    string    `json:"species"`
	Time       string    `json:"time"`
	Date       string    `json:"date"`
	TimeAgo    string    `json:"time_ago"`
	DetectedAt time.Time `json:"detected_at"`
}

// PiDiagnosticsResponse represents Raspberry Pi diagnostics.
type PiDiagnosticsResponse struct {
	IPs           IPInfo           `json:"ips"`
	Throttling    *ThrottleInfo    `json:"throttling,omitempty"`
	ClockSpeeds   map[string]int64 `json:"clock_speeds,omitempty"`
	Voltages      map[string]float64 `json:"voltages,omitempty"`
	CaddyConfig   string           `json:"caddy_config,omitempty"`
	CrontabActive string           `json:"crontab_active,omitempty"`
	IsPi          bool             `json:"is_pi"`
}

// IPInfo contains network IP information.
type IPInfo struct {
	LAN    string `json:"lan"`
	Public string `json:"public"`
}

// ThrottleInfo represents Pi throttling status.
type ThrottleInfo struct {
	Raw                       string `json:"raw"`
	Binary                    string `json:"binary"`
	UnderVoltageDetected      bool   `json:"under_voltage_detected"`
	ArmFrequencyCapped        bool   `json:"arm_frequency_capped"`
	CurrentlyThrottled        bool   `json:"currently_throttled"`
	SoftTempLimitActive       bool   `json:"soft_temp_limit_active"`
	UnderVoltageOccurred      bool   `json:"under_voltage_occurred"`
	ArmFrequencyCappedOccurred bool   `json:"arm_frequency_capped_occurred"`
	ThrottlingOccurred        bool   `json:"throttling_occurred"`
	SoftTempLimitOccurred     bool   `json:"soft_temp_limit_occurred"`
}

// SystemDiagnosticsResponse contains full system diagnostics.
type SystemDiagnosticsResponse struct {
	Services    []ServiceDiagnostic `json:"services"`
	DiskUsage   DiskUsageResponse   `json:"disk_usage"`
	Memory      MemoryInfo          `json:"memory"`
	LoadAverage LoadInfo            `json:"load_average"`
	CPU         CPUInfo             `json:"cpu"`
	Temperature TemperatureInfo     `json:"temperature"`
	Microphones []string            `json:"microphones"`
	DateTime    string              `json:"date_time"`
	PiInfo      *PiDiagnosticsResponse `json:"pi_info,omitempty"`
}

// ServiceDiagnostic represents a service's diagnostic info.
type ServiceDiagnostic struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Details string `json:"details,omitempty"`
}

// MemoryInfo represents memory usage.
type MemoryInfo struct {
	Total     string `json:"total"`
	Used      string `json:"used"`
	Free      string `json:"free"`
	Available string `json:"available"`
}

// LoadInfo represents system load averages.
type LoadInfo struct {
	Load1  float64 `json:"load_1"`
	Load5  float64 `json:"load_5"`
	Load15 float64 `json:"load_15"`
	Uptime string  `json:"uptime"`
}

// CPUInfo represents CPU information.
type CPUInfo struct {
	Model     string `json:"model"`
	Cores     int    `json:"cores"`
	Frequency string `json:"frequency,omitempty"`
}

// TemperatureInfo represents system temperature.
type TemperatureInfo struct {
	Celsius    float64 `json:"celsius"`
	Fahrenheit float64 `json:"fahrenheit"`
}

// SpeciesCountResponse represents species file count information.
type SpeciesCountResponse struct {
	Location      string                 `json:"location"`
	FreeSpace     string                 `json:"free_space"`
	TotalSpecies  int                    `json:"total_species"`
	TotalFiles    int                    `json:"total_files"`
	TotalSize     string                 `json:"total_size"`
	SpeciesCounts []SpeciesFileCount     `json:"species_counts"`
}

// SpeciesFileCount represents file count for a species.
type SpeciesFileCount struct {
	Species  string `json:"species"`
	Count    int    `json:"count"`
	CountStr string `json:"count_str"`
}

// DiskUsage handles GET /api/diagnostics/disk requests.
// Returns disk usage information similar to disk_usage.sh.
func (h *Handlers) DiskUsage(w http.ResponseWriter, r *http.Request) {
	info, err := getDiskUsage("/")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get disk usage: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, info)
}

// MostRecent handles GET /api/diagnostics/most-recent requests.
// Returns the most recent bird detection similar to most_recent.sh.
func (h *Handlers) MostRecent(w http.ResponseWriter, r *http.Request) {
	row := h.db.Conn().QueryRow(`
		SELECT Com_Name, Time, Date FROM detections
		ORDER BY Date DESC, Time DESC
		LIMIT 1
	`)

	var species, timeStr, dateStr string
	if err := row.Scan(&species, &timeStr, &dateStr); err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusOK, map[string]string{
				"message": "No detections yet",
			})
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to query database: "+err.Error())
		return
	}

	// Parse the date and time
	detectedAt, _ := time.Parse("2006-01-02 15:04:05", dateStr+" "+timeStr)
	timeAgo := formatTimeAgo(detectedAt)

	// Format time nicely
	formattedTime := detectedAt.Format("3:04PM")

	response := MostRecentResponse{
		Species:    species,
		Time:       formattedTime,
		Date:       dateStr,
		TimeAgo:    timeAgo,
		DetectedAt: detectedAt,
	}

	writeJSON(w, http.StatusOK, response)
}

// PiDiagnostics handles GET /api/diagnostics/pi requests.
// Returns Raspberry Pi specific diagnostics similar to extra_info.sh.
func (h *Handlers) PiDiagnostics(w http.ResponseWriter, r *http.Request) {
	response := PiDiagnosticsResponse{
		IsPi: isPi(),
	}

	// Get IPs
	response.IPs = IPInfo{
		LAN:    getLANIP(),
		Public: getPublicIP(),
	}

	// Get Pi-specific info if on a Pi
	if response.IsPi {
		response.Throttling = getThrottleInfo()
		response.ClockSpeeds = getClockSpeeds()
		response.Voltages = getVoltages()
	}

	// Get Caddyfile content (without passwords)
	caddyContent, _ := os.ReadFile("/etc/caddy/Caddyfile")
	response.CaddyConfig = sanitizeCaddyConfig(string(caddyContent))

	// Get crontab (non-comment lines)
	crontabContent, _ := os.ReadFile("/etc/crontab")
	response.CrontabActive = filterCrontab(string(crontabContent))

	writeJSON(w, http.StatusOK, response)
}

// SystemDiagnostics handles GET /api/diagnostics/system requests.
// Returns comprehensive system diagnostics similar to print_diagnostic_info.sh.
func (h *Handlers) SystemDiagnostics(w http.ResponseWriter, r *http.Request) {
	response := SystemDiagnosticsResponse{
		DateTime: time.Now().Format(time.RFC3339),
	}

	// Get service status
	services := []string{
		"caddy", "birdnet_analysis", "birdnet_log", "birdnet_recording",
		"birdnet_stats", "chart_viewer", "extraction", "web_terminal",
		"spectrogram_viewer", "livestream",
	}
	for _, svc := range services {
		response.Services = append(response.Services, getServiceDiagnostic(svc))
	}

	// Get disk usage
	response.DiskUsage, _ = getDiskUsage("/")

	// Get memory info
	response.Memory = getMemoryInfo()

	// Get load averages
	response.LoadAverage = getLoadInfo()

	// Get CPU info
	response.CPU = getCPUInfo()

	// Get temperature
	response.Temperature = getTemperature()

	// Get microphone devices
	response.Microphones = getMicrophoneDevices()

	// Get Pi-specific info if on a Pi
	if isPi() {
		piInfo := &PiDiagnosticsResponse{
			IsPi:        true,
			IPs:         IPInfo{LAN: getLANIP(), Public: getPublicIP()},
			Throttling:  getThrottleInfo(),
			ClockSpeeds: getClockSpeeds(),
			Voltages:    getVoltages(),
		}
		response.PiInfo = piInfo
	}

	writeJSON(w, http.StatusOK, response)
}

// SpeciesCount handles GET /api/diagnostics/species-count requests.
// Returns species file count information similar to disk_species_count.sh.
func (h *Handlers) SpeciesCount(w http.ResponseWriter, r *http.Request) {
	baseDir := filepath.Join(h.birdsongsDir, "Extracted", "By_Date")

	// Get species from database
	rows, err := h.db.Conn().Query("SELECT DISTINCT Com_Name FROM detections")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to query species: "+err.Error())
		return
	}
	defer rows.Close()

	var species []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			species = append(species, name)
		}
	}

	// Count files for each species
	var counts []SpeciesFileCount
	totalFiles := 0

	for _, sp := range species {
		// Sanitize species name for filesystem
		sanitized := strings.ReplaceAll(sp, " ", "_")
		sanitized = strings.ReplaceAll(sanitized, "'", "")

		count := countSpeciesFiles(baseDir, sanitized)
		totalFiles += count

		counts = append(counts, SpeciesFileCount{
			Species:  sp,
			Count:    count,
			CountStr: formatCount(count),
		})
	}

	// Sort by count descending
	sort.Slice(counts, func(i, j int) bool {
		return counts[i].Count > counts[j].Count
	})

	// Get disk info
	diskInfo, _ := getDiskUsage(baseDir)
	totalSize := getDirSize(baseDir)

	response := SpeciesCountResponse{
		Location:      baseDir,
		FreeSpace:     diskInfo.Available,
		TotalSpecies:  len(species),
		TotalFiles:    totalFiles,
		TotalSize:     totalSize,
		SpeciesCounts: counts,
	}

	writeJSON(w, http.StatusOK, response)
}

// DumpLogs handles GET /api/diagnostics/logs requests.
// Returns a tar.gz of system logs similar to dump_logs.sh.
func (h *Handlers) DumpLogs(w http.ResponseWriter, r *http.Request) {
	// Create a buffer to write our archive to
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", "attachment; filename=birdnet-logs.tar.gz")

	gzWriter := gzip.NewWriter(w)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Add service logs
	services := []string{
		"birdnet_analysis", "birdnet_recording", "birdnet_log",
		"chart_viewer", "spectrogram_viewer", "livestream",
	}

	for _, svc := range services {
		logContent := getServiceLog(svc, 100)
		if logContent != "" {
			addToTar(tarWriter, svc+".log", []byte(logContent))
		}
	}

	// Add config file (without passwords)
	configContent, _ := os.ReadFile("/etc/birdnet/birdnet.conf")
	sanitizedConfig := sanitizeConfig(string(configContent))
	addToTar(tarWriter, "birdnet.conf", []byte(sanitizedConfig))

	// Add Caddyfile (without passwords)
	caddyContent, _ := os.ReadFile("/etc/caddy/Caddyfile")
	sanitizedCaddy := sanitizeCaddyConfig(string(caddyContent))
	addToTar(tarWriter, "Caddyfile", []byte(sanitizedCaddy))

	// Add system info
	sysInfo := collectSystemInfo()
	addToTar(tarWriter, "sysinfo.txt", []byte(sysInfo))

	// Add sound card info
	soundInfo := getSoundCardInfo()
	addToTar(tarWriter, "soundcard.txt", []byte(soundInfo))
}

// Helper functions

func getDiskUsage(path string) (DiskUsageResponse, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return DiskUsageResponse{}, err
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bfree * uint64(stat.Bsize)
	used := total - free
	avail := stat.Bavail * uint64(stat.Bsize)

	usedPct := 0
	if total > 0 {
		usedPct = int((used * 100) / total)
	}

	return DiskUsageResponse{
		Size:      formatBytes(total),
		Used:      formatBytes(used),
		Available: formatBytes(avail),
		UsedPct:   usedPct,
		Path:      path,
	}, nil
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatTimeAgo(t time.Time) string {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterday := today.AddDate(0, 0, -1)
	twoDaysAgo := today.AddDate(0, 0, -2)

	tDate := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())

	if tDate.Equal(today) {
		return "today"
	} else if tDate.Equal(yesterday) {
		return "yesterday"
	} else if tDate.Equal(twoDaysAgo) {
		return "two days ago"
	}
	return "on " + t.Format("2006-01-02")
}

func isPi() bool {
	_, err := exec.LookPath("vcgencmd")
	return err == nil
}

func getLANIP() string {
	out, err := exec.Command("hostname", "-I").Output()
	if err != nil {
		return "unknown"
	}
	parts := strings.Fields(string(out))
	if len(parts) > 0 {
		return parts[0]
	}
	return "unknown"
}

func getPublicIP() string {
	out, err := exec.Command("curl", "-s4", "--max-time", "5", "ifconfig.co").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func getThrottleInfo() *ThrottleInfo {
	out, err := exec.Command("vcgencmd", "get_throttled").Output()
	if err != nil {
		return nil
	}

	raw := strings.TrimSpace(string(out))
	// Parse throttled=0x0 format
	parts := strings.Split(raw, "=")
	if len(parts) != 2 {
		return nil
	}

	hexStr := strings.TrimPrefix(parts[1], "0x")
	val, err := strconv.ParseUint(hexStr, 16, 32)
	if err != nil {
		return nil
	}

	return &ThrottleInfo{
		Raw:                       raw,
		Binary:                    fmt.Sprintf("%b", val),
		UnderVoltageDetected:      (val & (1 << 0)) != 0,
		ArmFrequencyCapped:        (val & (1 << 1)) != 0,
		CurrentlyThrottled:        (val & (1 << 2)) != 0,
		SoftTempLimitActive:       (val & (1 << 3)) != 0,
		UnderVoltageOccurred:      (val & (1 << 16)) != 0,
		ArmFrequencyCappedOccurred: (val & (1 << 17)) != 0,
		ThrottlingOccurred:        (val & (1 << 18)) != 0,
		SoftTempLimitOccurred:     (val & (1 << 19)) != 0,
	}
}

func getClockSpeeds() map[string]int64 {
	clocks := make(map[string]int64)
	clockTypes := []string{"arm", "core", "h264", "isp", "v3d", "uart", "pwm", "emmc", "pixel", "vec", "hdmi", "dpi"}

	for _, ct := range clockTypes {
		out, err := exec.Command("vcgencmd", "measure_clock", ct).Output()
		if err != nil {
			continue
		}
		// Parse frequency=600000000 format
		parts := strings.Split(strings.TrimSpace(string(out)), "=")
		if len(parts) == 2 {
			if val, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
				clocks[ct] = val
			}
		}
	}

	return clocks
}

func getVoltages() map[string]float64 {
	volts := make(map[string]float64)
	voltTypes := []string{"core", "sdram_c", "sdram_i", "sdram_p"}

	for _, vt := range voltTypes {
		out, err := exec.Command("vcgencmd", "measure_volts", vt).Output()
		if err != nil {
			continue
		}
		// Parse volt=1.2000V format
		re := regexp.MustCompile(`volt=(\d+\.?\d*)V`)
		matches := re.FindStringSubmatch(string(out))
		if len(matches) == 2 {
			if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
				volts[vt] = val
			}
		}
	}

	return volts
}

func sanitizeCaddyConfig(content string) string {
	// Remove basicauth blocks (3 lines starting with basicauth)
	re := regexp.MustCompile(`(?m)^\s*basicauth\s*\{[^}]*\}`)
	return re.ReplaceAllString(content, "[basicauth block removed]")
}

func filterCrontab(content string) string {
	var active []string
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			active = append(active, line)
		}
	}
	return strings.Join(active, "\n")
}

func getServiceDiagnostic(name string) ServiceDiagnostic {
	diag := ServiceDiagnostic{Name: name}

	out, err := exec.Command("systemctl", "is-active", name+".service").Output()
	if err != nil {
		diag.Status = "unknown"
	} else {
		diag.Status = strings.TrimSpace(string(out))
	}

	return diag
}

func getMemoryInfo() MemoryInfo {
	out, _ := exec.Command("free", "-h").Output()
	lines := strings.Split(string(out), "\n")
	if len(lines) < 2 {
		return MemoryInfo{}
	}

	fields := strings.Fields(lines[1])
	if len(fields) < 7 {
		return MemoryInfo{}
	}

	return MemoryInfo{
		Total:     fields[1],
		Used:      fields[2],
		Free:      fields[3],
		Available: fields[6],
	}
}

func getLoadInfo() LoadInfo {
	out, _ := exec.Command("uptime").Output()
	uptimeStr := strings.TrimSpace(string(out))

	info := LoadInfo{Uptime: uptimeStr}

	// Parse load averages
	re := regexp.MustCompile(`load average:\s*(\d+\.?\d*),\s*(\d+\.?\d*),\s*(\d+\.?\d*)`)
	matches := re.FindStringSubmatch(uptimeStr)
	if len(matches) == 4 {
		info.Load1, _ = strconv.ParseFloat(matches[1], 64)
		info.Load5, _ = strconv.ParseFloat(matches[2], 64)
		info.Load15, _ = strconv.ParseFloat(matches[3], 64)
	}

	return info
}

func getCPUInfo() CPUInfo {
	info := CPUInfo{}

	content, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return info
	}

	lines := strings.Split(string(content), "\n")
	cores := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				info.Model = strings.TrimSpace(parts[1])
			}
		}
		if strings.HasPrefix(line, "processor") {
			cores++
		}
	}
	info.Cores = cores

	return info
}

func getTemperature() TemperatureInfo {
	content, err := os.ReadFile("/sys/class/thermal/thermal_zone0/temp")
	if err != nil {
		return TemperatureInfo{}
	}

	milliC, err := strconv.ParseFloat(strings.TrimSpace(string(content)), 64)
	if err != nil {
		return TemperatureInfo{}
	}

	celsius := milliC / 1000.0
	fahrenheit := celsius*9/5 + 32

	return TemperatureInfo{
		Celsius:    celsius,
		Fahrenheit: fahrenheit,
	}
}

func getMicrophoneDevices() []string {
	out, err := exec.Command("arecord", "-l").Output()
	if err != nil {
		return nil
	}

	var devices []string
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, "card") {
			devices = append(devices, strings.TrimSpace(line))
		}
	}

	return devices
}

func countSpeciesFiles(baseDir, species string) int {
	count := 0
	filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		// Check if this file is in a species directory
		dir := filepath.Base(filepath.Dir(path))
		if dir == species {
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".mp3" || ext == ".wav" || ext == ".flac" || ext == ".ogg" || ext == ".opus" {
				count++
			}
		}
		return nil
	})
	return count
}

func formatCount(count int) string {
	if count >= 1000 {
		return fmt.Sprintf("%.1fk", float64(count)/1000)
	}
	return strconv.Itoa(count)
}

func getDirSize(path string) string {
	out, err := exec.Command("du", "-sh", path).Output()
	if err != nil {
		return "unknown"
	}
	fields := strings.Fields(string(out))
	if len(fields) > 0 {
		return fields[0]
	}
	return "unknown"
}

func getServiceLog(service string, lines int) string {
	out, err := exec.Command("journalctl", "-u", service+".service", "-n", strconv.Itoa(lines), "--no-pager").Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func sanitizeConfig(content string) string {
	// Remove lines containing PWD
	var sanitized []string
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "PWD=") {
			sanitized = append(sanitized, line)
		}
	}
	return strings.Join(sanitized, "\n")
}

func collectSystemInfo() string {
	var info strings.Builder

	// Disk usage
	out, _ := exec.Command("df", "-h").Output()
	info.WriteString("=== Disk Usage ===\n")
	info.Write(out)
	info.WriteString("\n")

	// Memory
	out, _ = exec.Command("free", "-h").Output()
	info.WriteString("=== Memory ===\n")
	info.Write(out)
	info.WriteString("\n")

	// Network
	out, _ = exec.Command("ip", "addr").Output()
	info.WriteString("=== Network ===\n")
	info.Write(out)
	info.WriteString("\n")

	return info.String()
}

func getSoundCardInfo() string {
	var info strings.Builder

	out, _ := exec.Command("arecord", "-L").Output()
	info.WriteString("=== Recording Devices ===\n")
	info.Write(out)
	info.WriteString("\n")

	out, _ = exec.Command("arecord", "-l").Output()
	info.WriteString("=== Sound Cards ===\n")
	info.Write(out)

	return info.String()
}

func addToTar(tw *tar.Writer, name string, data []byte) error {
	header := &tar.Header{
		Name:    name,
		Mode:    0644,
		Size:    int64(len(data)),
		ModTime: time.Now(),
	}

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	_, err := io.Copy(tw, strings.NewReader(string(data)))
	return err
}
