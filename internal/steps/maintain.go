package steps

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/paulo-amaral/Easy-Install-docker-ce-docker-compose/internal/sysinfo"
)

func statusStep() Step {
	return Step{
		ID:    "status",
		Title: "Status & disk usage",
		Desc:  "Engine health, versions and docker system df",
		Report: func(info sysinfo.Info) (string, error) {
			var b strings.Builder
			section(&b, "docker version", "docker", "version", "--format",
				"Server: {{.Server.Version}}  Client: {{.Client.Version}}")
			section(&b, "compose plugin", "docker", "compose", "version")
			section(&b, "disk usage", "docker", "system", "df")
			section(&b, "running containers", "docker", "ps", "--format",
				"table {{.Names}}\t{{.Image}}\t{{.Status}}")
			return b.String(), nil
		},
	}
}

func pruneStep() Step {
	return Step{
		ID:          "prune",
		Title:       "Prune unused data",
		Desc:        "docker system prune: removes stopped containers, dangling images, unused networks",
		Destructive: true,
		Command: func(sysinfo.Info) *exec.Cmd {
			return exec.Command("docker", "system", "prune", "--force")
		},
	}
}

func section(b *strings.Builder, title string, name string, args ...string) {
	fmt.Fprintf(b, "── %s\n", title)
	out, err := exec.Command(name, args...).CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil && text == "" {
		text = "unavailable: " + err.Error()
	}
	b.WriteString(text + "\n\n")
}
