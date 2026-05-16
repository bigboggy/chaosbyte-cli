package reportcli

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// autoThrottle bounds how often the local TUI re-fires the report
// subprocess. Matches the scheduler cadence installed by scripts/install.sh
// (one minute) so the in-TUI auto-trigger keeps the leaderboard equally
// fresh for users who run `vibespace` locally between scheduler ticks.
const autoThrottle = 1 * time.Minute

// KickBackground launches `vibespace report` as a detached subprocess so the
// user doesn't have to remember to upload manually. Best-effort: any failure
// (binary not on disk, SSH key not linked, network down) is swallowed —
// the marker file still gets touched on completion so we don't retry on
// every TUI launch.
//
// Returns immediately; the subprocess runs in the background. Callers must
// not rely on its completion for anything user-facing.
//
// cfgDir is where the timestamp marker lives — typically the same directory
// the local store sits in (e.g. ~/.config/vibespace).
func KickBackground(cfgDir string) {
	if cfgDir == "" {
		return
	}
	marker := filepath.Join(cfgDir, "last-report.txt")
	if recentlyReported(marker, autoThrottle) {
		return
	}
	self, err := os.Executable()
	if err != nil {
		return
	}
	cmd := exec.Command(self, "report")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	cmd.Stdin = nil
	if err := cmd.Start(); err != nil {
		return
	}
	go func() {
		_ = cmd.Wait()
		// Touch the marker regardless of success — otherwise a persistently
		// failing config (no /auth link, offline, etc.) re-fires the
		// subprocess on every launch. The throttle window backs off
		// uniformly whether or not the upload actually wrote anything.
		_ = os.WriteFile(marker, []byte(time.Now().UTC().Format(time.RFC3339)), 0o644)
	}()
}

func recentlyReported(marker string, within time.Duration) bool {
	data, err := os.ReadFile(marker)
	if err != nil {
		return false
	}
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(string(data)))
	if err != nil {
		return false
	}
	return time.Since(t) < within
}
