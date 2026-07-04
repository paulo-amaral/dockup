package steps

import (
	"os/exec"
	"strings"

	"github.com/paulo-amaral/Easy-Install-docker-ce-docker-compose/internal/sysinfo"
)

// Apple's open source `container` tool (github.com/apple/container) runs
// Linux containers in lightweight VMs on Apple Silicon. Requires macOS 15.5+.
// Homebrew must not run as root, so these steps are intentionally NeedsRoot=false.
func appleSteps(info sysinfo.Info) []Step {
	if info.Arch != "arm64" {
		return []Step{{
			ID:    "apple-info",
			Title: "Apple container (unavailable)",
			Desc:  "Requires Apple Silicon; this Mac is " + info.Arch,
			Report: func(sysinfo.Info) (string, error) {
				return "Apple's container tool only supports Apple Silicon (arm64).\n" +
					"On Intel Macs use Docker Desktop or Colima instead.\n", nil
			},
		}}
	}
	return []Step{
		{
			ID:    "apple-install",
			Title: "Install Apple container",
			Desc:  "brew install --cask container (Apple's native Linux container runtime)",
			Command: func(sysinfo.Info) *exec.Cmd {
				return shell(`set -e
command -v brew >/dev/null 2>&1 || { echo "Homebrew not found. Install it from https://brew.sh or grab the signed .pkg from https://github.com/apple/container/releases"; exit 1; }
brew install --cask container
container --version`)
			},
		},
		{
			ID:    "apple-start",
			Title: "Start container system",
			Desc:  "container system start (offers to fetch a Linux kernel on first run)",
			Command: func(sysinfo.Info) *exec.Cmd {
				return exec.Command("container", "system", "start")
			},
		},
		{
			ID:    "apple-stop",
			Title: "Stop container system",
			Desc:  "container system stop",
			Command: func(sysinfo.Info) *exec.Cmd {
				return exec.Command("container", "system", "stop")
			},
		},
		{
			ID:    "apple-status",
			Title: "Apple container status",
			Desc:  "Version and running containers",
			Report: func(sysinfo.Info) (string, error) {
				if _, err := exec.LookPath("container"); err != nil {
					return "Apple container is not installed. Use \"Install Apple container\" first.\n", nil
				}
				var b strings.Builder
				section(&b, "version", "container", "--version")
				section(&b, "containers", "container", "list", "--all")
				return b.String(), nil
			},
		},
	}
}
