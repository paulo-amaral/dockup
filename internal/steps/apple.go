package steps

import (
	"os/exec"
	"strings"

	"github.com/paulo-amaral/dockup/v2/internal/sysinfo"
)

// Apple's open source `container` tool (github.com/apple/container) runs
// Linux containers in lightweight VMs on Apple Silicon. Requires macOS 15.5+.
// Install preference: MacPorts, then Apple's signed .pkg from GitHub releases.
// Homebrew also works (brew install --cask container) but is deliberately not
// automated here; it is documented in the README as an alternative.
// Steps stay NeedsRoot=false: the script escalates only where needed via sudo.
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
			Desc:  "Via MacPorts, or Apple's signed .pkg (Apple's native Linux container runtime)",
			Command: func(sysinfo.Info) *exec.Cmd {
				return shell(`set -e
if command -v port >/dev/null 2>&1; then
    echo "Installing via MacPorts (sudo may prompt for your password)..."
    sudo port install container
else
    echo "MacPorts not found; installing Apple's signed .pkg from GitHub releases..."
    url=$(curl -fsSL https://api.github.com/repos/apple/container/releases/latest | grep -o 'https://[^"]*\.pkg' | head -n1)
    [ -n "$url" ] || { echo "could not locate a .pkg asset in the latest release"; exit 1; }
    tmp=$(mktemp -d)
    trap 'rm -rf "$tmp"' EXIT
    curl -fsSL -o "$tmp/container.pkg" "$url"
    sudo installer -pkg "$tmp/container.pkg" -target /
fi
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
