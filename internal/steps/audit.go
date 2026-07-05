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
		Desc:  "CIS-inspired checks: socket, daemon flags, TLS, registries, running containers",
		Report: func(info sysinfo.Info) (string, error) {
			var b strings.Builder
			b.WriteString("dockup security audit — read-only, nothing was changed\n\n")
			for _, c := range Audit() {
				tag := c.Status
				if c.Severity != "" {
					tag += ":" + c.Severity
				}
				fmt.Fprintf(&b, "%-12s %-24s %s\n", "["+tag+"]", c.Name, c.Detail)
			}
			b.WriteString("\nReference: CIS Docker Benchmark (subset)\n")
			return b.String(), nil
		},
	}
}

// AuditResult is one security check outcome. Status is PASS, WARN or INFO;
// Severity (low, medium, high) is set only on WARN.
type AuditResult struct {
	Status   string `json:"status"`
	Severity string `json:"severity,omitempty"`
	Name     string `json:"name"`
	Detail   string `json:"detail"`
}

func pass(name, detail string) AuditResult { return AuditResult{"PASS", "", name, detail} }
func info(name, detail string) AuditResult { return AuditResult{"INFO", "", name, detail} }
func warn(sev, name, detail string) AuditResult {
	return AuditResult{"WARN", sev, name, detail}
}

// Audit runs all read-only security checks. Exported for --format json.
func Audit() []AuditResult {
	cfg := readDaemonConfig()
	results := []AuditResult{
		checkSocket(),
		checkDockerGroup(),
		checkDaemonKey(cfg, "log-opts", "log rotation", "low", "log-opts with max-size limits container log growth"),
		checkDaemonKey(cfg, "live-restore", "live-restore", "low", "containers survive daemon restarts"),
		checkDaemonKey(cfg, "no-new-privileges", "no-new-privileges", "medium", "blocks privilege escalation via setuid binaries"),
		checkUsernsRemap(cfg),
		checkICC(cfg),
		checkDaemonTLS(cfg),
		checkInsecureRegistries(cfg),
		checkExperimental(cfg),
		checkContentTrust(),
	}
	return append(results, containerChecks()...)
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
		return info("docker socket", "not found (daemon not running?)")
	}
	mode := fi.Mode().Perm()
	if mode&0o007 != 0 {
		return warn("high", "docker socket", fmt.Sprintf("%s is world-accessible (%o); expected 660 root:docker", sock, mode))
	}
	if st, ok := fi.Sys().(*syscall.Stat_t); ok && st.Uid != 0 {
		return warn("high", "docker socket", fmt.Sprintf("owner uid %d, expected root", st.Uid))
	}
	return pass("docker socket", fmt.Sprintf("permissions %o", mode))
}

func checkDockerGroup() AuditResult {
	out, err := exec.Command("getent", "group", "docker").Output()
	if err != nil {
		return info("docker group", "no docker group found")
	}
	parts := strings.Split(strings.TrimSpace(string(out)), ":")
	members := ""
	if len(parts) == 4 {
		members = parts[3]
	}
	if members == "" {
		return pass("docker group", "no extra members")
	}
	return warn("medium", "docker group", "members are root-equivalent: "+members)
}

func checkDaemonKey(cfg map[string]any, key, name, sev, why string) AuditResult {
	if _, ok := cfg[key]; ok {
		return pass(name, "configured in daemon.json")
	}
	return warn(sev, name, "not set — "+why+" (run: dockup harden)")
}

func checkUsernsRemap(cfg map[string]any) AuditResult {
	if _, ok := cfg["userns-remap"]; ok {
		return pass("userns-remap", "user namespace remapping active")
	}
	return info("userns-remap", "not set; consider it for multi-tenant hosts (breaks some workloads)")
}

func checkICC(cfg map[string]any) AuditResult {
	if v, ok := cfg["icc"].(bool); ok && !v {
		return pass("inter-container traffic", "icc disabled; containers must opt in via networks")
	}
	return warn("low", "inter-container traffic", "icc not restricted; all containers on the default bridge can talk to each other")
}

func checkDaemonTLS(cfg map[string]any) AuditResult {
	hosts, _ := cfg["hosts"].([]any)
	for _, h := range hosts {
		s, _ := h.(string)
		if strings.HasPrefix(s, "tcp://") {
			if v, ok := cfg["tlsverify"].(bool); ok && v {
				return pass("daemon TCP endpoint", s+" with tlsverify")
			}
			return warn("high", "daemon TCP endpoint", s+" exposed without tlsverify — remote root access")
		}
	}
	return pass("daemon TCP endpoint", "daemon not exposed on TCP")
}

func checkInsecureRegistries(cfg map[string]any) AuditResult {
	regs, _ := cfg["insecure-registries"].([]any)
	if len(regs) == 0 {
		return pass("insecure registries", "none configured")
	}
	var names []string
	for _, r := range regs {
		if s, ok := r.(string); ok {
			names = append(names, s)
		}
	}
	return warn("high", "insecure registries", "plaintext/unverified pulls allowed from: "+strings.Join(names, ", "))
}

func checkExperimental(cfg map[string]any) AuditResult {
	if v, ok := cfg["experimental"].(bool); ok && v {
		return info("experimental features", "enabled in daemon.json; disable on production hosts")
	}
	return pass("experimental features", "disabled")
}

func checkContentTrust() AuditResult {
	if os.Getenv("DOCKER_CONTENT_TRUST") == "1" {
		return pass("content trust", "DOCKER_CONTENT_TRUST=1 in this environment")
	}
	return info("content trust", "DOCKER_CONTENT_TRUST not set; image signatures are not enforced")
}

// containerChecks inspects running containers once and derives privilege and
// sensitive-mount findings from the same pass.
func containerChecks() []AuditResult {
	out, err := exec.Command("docker", "ps", "--quiet").Output()
	if err != nil {
		return []AuditResult{info("running containers", "cannot query daemon")}
	}
	ids := strings.Fields(string(out))
	if len(ids) == 0 {
		return []AuditResult{pass("running containers", "none")}
	}
	args := append([]string{"inspect", "--format",
		"{{.Name}}|{{.HostConfig.Privileged}}|{{range .HostConfig.Binds}}{{.}};{{end}}"}, ids...)
	insp, err := exec.Command("docker", args...).Output()
	if err != nil {
		return []AuditResult{info("running containers", "inspect failed")}
	}

	var privileged, sensitive []string
	for _, line := range strings.Split(strings.TrimSpace(string(insp)), "\n") {
		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 {
			continue
		}
		name := strings.TrimPrefix(parts[0], "/")
		if parts[1] == "true" {
			privileged = append(privileged, name)
		}
		for _, bind := range strings.Split(parts[2], ";") {
			src, _, _ := strings.Cut(bind, ":")
			switch src {
			case "/", "/etc", "/boot", "/var/run/docker.sock", "/run/docker.sock":
				sensitive = append(sensitive, name+" mounts "+src)
			}
		}
	}

	results := make([]AuditResult, 0, 2)
	if len(privileged) > 0 {
		results = append(results, warn("high", "privileged containers", strings.Join(privileged, ", ")))
	} else {
		results = append(results, pass("privileged containers", fmt.Sprintf("%d running, none privileged", len(ids))))
	}
	if len(sensitive) > 0 {
		results = append(results, warn("high", "sensitive mounts", strings.Join(sensitive, ", ")))
	} else {
		results = append(results, pass("sensitive mounts", "no host-critical paths or docker.sock mounted"))
	}
	return results
}
