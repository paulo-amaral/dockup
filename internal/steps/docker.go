package steps

import (
	"os/exec"

	"github.com/paulo-amaral/dockup/internal/sysinfo"
)

// Docker's official convenience script installs the engine, the buildx and
// compose v2 plugins, and sets up the distro repository for future updates.
// The standalone python docker-compose (v1) is dead; nothing here installs it.
func dockerInstallStep() Step {
	return Step{
		ID:        "install",
		Title:     "Install / update Docker Engine + Compose v2",
		Desc:      "Official get.docker.com script: engine, CLI, buildx and compose plugins",
		NeedsRoot: true,
		Command: func(sysinfo.Info) *exec.Cmd {
			return shell("curl -fsSL https://get.docker.com | sh && systemctl enable --now docker && docker version")
		},
	}
}
