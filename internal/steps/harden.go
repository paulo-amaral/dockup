package steps

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/paulo-amaral/dockup/internal/sysinfo"
)

const daemonJSONPath = "/etc/docker/daemon.json"

// hardenSettings are the security defaults dockup applies. Tune here.
var hardenSettings = map[string]any{
	"log-driver":        "json-file",
	"log-opts":          map[string]any{"max-size": "10m", "max-file": "3"},
	"live-restore":      true,
	"no-new-privileges": true,
}

func hardenStep() Step {
	return Step{
		ID:        "harden",
		Title:     "Apply security hardening",
		Desc:      "daemon.json: log rotation, live-restore, no-new-privileges (with backup)",
		NeedsRoot: true,
		Report: func(info sysinfo.Info) (string, error) {
			changed, backup, err := mergeDaemonJSON(daemonJSONPath)
			if err != nil {
				return "", err
			}
			var b strings.Builder
			if backup != "" {
				fmt.Fprintf(&b, "Backup saved: %s\n", backup)
			}
			if len(changed) == 0 {
				b.WriteString("All hardening settings were already in place. Nothing to do.\n")
				return b.String(), nil
			}
			fmt.Fprintf(&b, "Applied to %s:\n", daemonJSONPath)
			for _, c := range changed {
				fmt.Fprintf(&b, "  + %s\n", c)
			}
			b.WriteString("\nRestarting docker daemon...\n")
			if out, err := exec.Command("systemctl", "restart", "docker").CombinedOutput(); err != nil {
				return b.String(), fmt.Errorf("docker restart failed: %s: %w", strings.TrimSpace(string(out)), err)
			}
			b.WriteString("Docker restarted. Hardening active.\n")
			return b.String(), nil
		},
	}
}

// mergeDaemonJSON applies hardenSettings on top of the existing config,
// returning which keys changed and the backup path (if a file existed).
// An unparsable existing file aborts the operation; overwriting a config we
// cannot read is worse than doing nothing.
func mergeDaemonJSON(path string) (changed []string, backup string, err error) {
	cfg := map[string]any{}
	raw, readErr := os.ReadFile(path)
	exists := readErr == nil
	if exists {
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return nil, "", fmt.Errorf("%s exists but is not valid JSON, fix it manually first: %w", path, err)
		}
	}

	for k, v := range hardenSettings {
		if fmt.Sprintf("%v", cfg[k]) == fmt.Sprintf("%v", v) {
			continue
		}
		cfg[k] = v
		changed = append(changed, fmt.Sprintf("%s = %s", k, mustJSON(v)))
	}
	if len(changed) == 0 {
		return nil, "", nil
	}

	if exists {
		backup = path + ".bak." + time.Now().Format("20060102-150405")
		if err := os.WriteFile(backup, raw, 0o644); err != nil {
			return nil, "", fmt.Errorf("could not write backup: %w", err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, backup, err
	}
	out, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(path, append(out, '\n'), 0o644); err != nil {
		return nil, backup, err
	}
	return changed, backup, nil
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
