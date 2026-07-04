// Package sysinfo detects the host environment: distro, architecture,
// privileges, GPU presence and the state of the Docker stack.
package sysinfo

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type Info struct {
	OS         string // runtime.GOOS
	Distro     string // os-release ID (ubuntu, debian, fedora, rhel, ...)
	DistroName string // os-release PRETTY_NAME
	Family     string // "deb", "rpm" or "" when unsupported
	Arch       string
	Root       bool
	HasGPU     bool   // NVIDIA GPU visible on the host
	Docker     string // server (or client) version, "" if absent
	Compose    string // compose v2 plugin version, "" if absent
	NvidiaCTK  bool   // nvidia-ctk binary present
}

func Detect() Info {
	info := Info{OS: runtime.GOOS, Arch: runtime.GOARCH}
	info.Root = os.Geteuid() == 0

	if info.OS == "linux" {
		info.Distro, info.DistroName, info.Family = readOSRelease("/etc/os-release")
		info.HasGPU = detectNvidiaGPU()
	}

	info.Docker = dockerVersion()
	info.Compose = composeVersion()
	_, err := exec.LookPath("nvidia-ctk")
	info.NvidiaCTK = err == nil
	return info
}

func readOSRelease(path string) (id, pretty, family string) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", "", ""
	}
	var idLike string
	for _, line := range strings.Split(string(b), "\n") {
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		v = strings.Trim(v, `"`)
		switch k {
		case "ID":
			id = v
		case "ID_LIKE":
			idLike = v
		case "PRETTY_NAME":
			pretty = v
		}
	}
	all := id + " " + idLike
	switch {
	case containsAny(all, "debian", "ubuntu"):
		family = "deb"
	case containsAny(all, "rhel", "fedora", "centos", "amzn", "rocky", "almalinux"):
		family = "rpm"
	}
	return id, pretty, family
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func detectNvidiaGPU() bool {
	if _, err := os.Stat("/proc/driver/nvidia"); err == nil {
		return true
	}
	// Driver may not be loaded yet; fall back to PCI device class.
	out, err := exec.Command("lspci").Output()
	return err == nil && strings.Contains(strings.ToLower(string(out)), "nvidia")
}

func dockerVersion() string {
	out, err := exec.Command("docker", "version", "--format", "{{.Server.Version}}").Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}
	// Daemon may be down or unreachable without root; report the client.
	out, err = exec.Command("docker", "version", "--format", "{{.Client.Version}}").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out)) + " (client)"
}

func composeVersion() string {
	out, err := exec.Command("docker", "compose", "version", "--short").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
