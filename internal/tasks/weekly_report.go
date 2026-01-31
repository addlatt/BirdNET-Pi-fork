package tasks

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"time"
)

// WeeklyReportTask generates and sends a weekly bird detection summary.
// This calls the existing Python script for report generation and Apprise notifications.
type WeeklyReportTask struct {
	scriptsDir string
}

// NewWeeklyReportTask creates a new weekly report task.
func NewWeeklyReportTask(scriptsDir string) *WeeklyReportTask {
	return &WeeklyReportTask{
		scriptsDir: scriptsDir,
	}
}

func (t *WeeklyReportTask) Name() string {
	return "weekly_report"
}

func (t *WeeklyReportTask) Description() string {
	return "Generates and sends weekly bird detection summary via Apprise"
}

func (t *WeeklyReportTask) DefaultSchedule() string {
	return "0 9 * * 6" // Saturday at 9 AM
}

func (t *WeeklyReportTask) Timeout() time.Duration {
	return 10 * time.Minute
}

func (t *WeeklyReportTask) Run(ctx context.Context) error {
	scriptPath := filepath.Join(t.scriptsDir, "tools", "weekly_report.py")

	log.Printf("Weekly report: running %s", scriptPath)

	cmd := exec.CommandContext(ctx, "python3", scriptPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("weekly report script failed: %w\nOutput: %s", err, string(output))
	}

	log.Printf("Weekly report: completed successfully\n%s", string(output))
	return nil
}
