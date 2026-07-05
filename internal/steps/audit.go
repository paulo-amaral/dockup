package steps

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/paulo-amaral/dockup/v2/internal/sysinfo"
)

// auditStep runs read-only checks inspired by the CIS Docker Benchmark.
// It changes nothing on the host; it only reports.
func auditStep() Step {
	return Step{
		ID:    "audit",
		Title: "Security audit (read-only)",
		Desc:  "CIS-inspired checks: socket perms, docker group, daemon.json, privileged containers",
		Report: func(info sysinfo.Info) (string, error) {
			var b strings.Builder
			b.WriteString("dockup security audit — read-only, nothing was changed\n\n")
			for _, c := range Audit() {
				fmt.Fprintf(&b, "[%s] %-28s %s\n", c.Status, c.Name, c.Detail)
			}
			b.WriteString("\nReference: CIS Docker Benchmark (basic subset)\n")
			return b.String(), nil
		},
	}
}

// AuditResult is one security check outcome. Status is PASS, WARN or INFO.
type AuditResult struct {
	Status string `json:"status"`
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

// Audit runs all read-only security checks. Exported for --format json.
func Audit() []AuditResult {
	cfg := readDaemonConfig()
	return []AuditResult{
		checkSocket(),
		checkDockerGroup(),
		checkDaemonKey(cfg, "log-opts", "log rotation", "log-opts with max-size limits container log growth"),
		checkDaemonKey(cfg, "live-restore", "live-restore", "containers survive daemon restarts"),
		checkDaemonKey(cfg, "no-new-privileges", "no-new-privileges", "blocks privilege escalation via setuid binaries"),
		checkUsernsRemap(cfg),
		checkPrivilegedContainers(),
	}
}

func readDaemonConfig() map[string]any {
	cfg := map[string]any{}
	if raw, err := os.ReadFile(daemonJSONPath); err == nil {
		_ = json.Unmarshal(raw, &cfg)
	}
	return cfg
}

func checkSocket() AuditResult {
	const sock = "/var/run/docker.sock"
	fi, err := os.Stat(sock)
	if err != nil {
		return AuditResult{"INFO", "docker socket", "not found (daemon not running?)"}
	}
	mode := fi.Mode().Perm()
	if mode&0o007 != 0 {
		return AuditResult{"WARN", "docker socket", fmt.Sprintf("%s is world-accessible (%o); expected 660 root:docker", sock, mode)}
	}
	if st, ok := fi.Sys().(*syscall.Stat_t); ok && st.Uid != 0 {
		return AuditResult{"WARN", "docker socket", fmt.Sprintf("owner uid %d, expected root", st.Uid)}
	}
	return AuditResult{"PASS", "docker socket", fmt.Sprintf("permissions %o", mode)}
}

func checkDockerGroup() AuditResult {
	out, err := exec.Command("getent", "group", "docker").Output()
	if err != nil {
		return AuditResult{"INFO", "docker group", "no docker group found"}
	}
	parts := strings.Split(strings.TrimSpace(string(out)), ":")
	members := ""
	if len(parts) == 4 {
		members = parts[3]
	}
	if members == "" {
		return AuditResult{"PASS", "docker group", "no extra members"}
	}
	return AuditResult{"WARN", "docker group", fmt.Sprintf("members are root-equivalent: %s", members)}
}

func checkDaemonKey(cfg map[string]any, key, name, why string) AuditResult {
	if _, ok := cfg[key]; ok {
		return AuditResult{"PASS", name, "configured in daemon.json"}
	}
	return AuditResult{"WARN", name, "not set — " + why + " (run: dockup harden)"}
}

func checkUsernsRemap(cfg map[string]any) AuditResult {
	if _, ok := cfg["userns-remap"]; ok {
		return AuditResult{"PASS", "userns-remap", "user namespace remapping active"}
	}
	return AuditResult{"INFO", "userns-remap", "not set; consider it for multi-tenant hosts (breaks some workloads)"}
}

func checkPrivilegedContainers() AuditResult {
	out, err := exec.Command("docker", "ps", "--quiet").Output()
	if err != nil {
		return AuditResult{"INFO", "privileged containers", "cannot query daemon"}
	}
	ids := strings.Fields(string(out))
	if len(ids) == 0 {
		return AuditResult{"PASS", "privileged containers", "no running containers"}
	}
	args := append([]string{"inspect", "--format", "{{.Name}} {{.HostConfig.Privileged}}"}, ids...)
	insp, err := exec.Command("docker", args...).Output()
	if err != nil {
		return AuditResult{"INFO", "privileged containers", "inspect failed"}
	}
	var bad []string
	for _, line := range strings.Split(strings.TrimSpace(string(insp)), "\n") {
		if strings.HasSuffix(line, " true") {
			bad = append(bad, strings.TrimPrefix(strings.Fields(line)[0], "/"))
		}
	}
	if len(bad) > 0 {
		return AuditResult{"WARN", "privileged containers", strings.Join(bad, ", ")}
	}
	return AuditResult{"PASS", "privileged containers", fmt.Sprintf("%d running, none privileged", len(ids))}
}
