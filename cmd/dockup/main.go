// dockup — interactive TUI to install, harden and maintain container
// runtimes: Docker Engine + Compose v2 (Linux), NVIDIA Container Toolkit,
// and Apple's container tool (macOS / Apple Silicon).
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/paulo-amaral/Easy-Install-docker-ce-docker-compose/internal/steps"
	"github.com/paulo-amaral/Easy-Install-docker-ce-docker-compose/internal/sysinfo"
	"github.com/paulo-amaral/Easy-Install-docker-ce-docker-compose/internal/tui"
)

// version is stamped by GoReleaser via -ldflags.
var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	yes := flag.Bool("yes", false, "non-interactive mode: run ACTION without the TUI")
	flag.Usage = usage
	flag.Parse()

	if *showVersion {
		fmt.Println("dockup", version)
		return
	}

	info := sysinfo.Detect()

	if *yes {
		if flag.NArg() != 1 {
			usage()
			os.Exit(2)
		}
		if err := steps.RunHeadless(flag.Arg(0), info); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		return
	}

	p := tea.NewProgram(tui.New(info, version), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `dockup %s — install & maintain container runtimes

usage:
  dockup                 launch the interactive TUI
  dockup --yes ACTION    run one action without the TUI (servers, CI)
  dockup --version       print version

actions on this host:
`, version)
	for _, st := range steps.All(sysinfo.Detect()) {
		fmt.Fprintf(os.Stderr, "  %-14s %s\n", st.ID, st.Title)
	}
}
