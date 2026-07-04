package steps

import (
	"os/exec"

	"github.com/paulo-amaral/dockup/internal/sysinfo"
)

// Podman ships in the default repositories of every distro dockup supports,
// so installation is a plain package manager call — no third-party repo.
func podmanStep() Step {
	return Step{
		ID:        "podman",
		Title:     "Install Podman",
		Desc:      "Daemonless, rootless-friendly container engine from the distro repos",
		NeedsRoot: true,
		Command: func(i sysinfo.Info) *exec.Cmd {
			if i.Family == "rpm" {
				return shell("(command -v dnf >/dev/null 2>&1 && dnf install -y podman) || yum install -y podman; podman --version")
			}
			return shell("apt-get update && apt-get install -y podman && podman --version")
		},
	}
}
