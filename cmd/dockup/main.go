// dockup — interactive TUI to install, harden and maintain container
// runtimes: Docker Engine + Compose v2 (Linux), NVIDIA Container Toolkit,
// and Apple's container tool (macOS / Apple Silicon).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime/debug"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/paulo-amaral/dockup/v2/internal/steps"
	"github.com/paulo-amaral/dockup/v2/internal/sysinfo"
	"github.com/paulo-amaral/dockup/v2/internal/tui"
)

// version is stamped by GoReleaser via -ldflags. Builds installed with
// `go install` skip ldflags, so fall back to the module version embedded
// by the Go toolchain.
var version = "dev"

func resolveVersion() string {
	if version != "dev" {
		return version
	}
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	return version
}

func main() {
	version = resolveVersion()
	showVersion := flag.Bool("version", false, "print version and exit")
	yes := flag.Bool("yes", false, "non-interactive mode: run ACTION without the TUI")
	format := flag.String("format", "text", `output format for the audit action: "text" or "json"`)
	flag.Usage = usage
	flag.Parse()

	if *showVersion {
		fmt.Println("dockup", version)
		return
	}

	info := sysinfo.Detect()

	if *yes {
		if flag.NArg() == 0 {
			usage()
			os.Exit(2)
		}
		action := flag.Arg(0)
		// Go's flag package stops at the first positional argument, so flags
		// written after ACTION (dockup --yes audit --format json) need a
		// second parse of the remainder.
		if flag.NArg() > 1 {
			_ = flag.CommandLine.Parse(flag.Args()[1:])
		}
		// JSON audit is CI-oriented: machine-readable and exit 3 on any WARN.
		if action == "audit" && *format == "json" {
			results := steps.Audit()
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(results); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			for _, r := range results {
				if r.Status == "WARN" {
					os.Exit(3)
				}
			}
			return
		}
		if err := steps.RunHeadless(action, info); err != nil {
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
  dockup                            launch the interactive TUI
  dockup --yes ACTION               run one action without the TUI (servers, CI)
  dockup --yes audit --format json  machine-readable audit; exit code 3 if any WARN
  dockup --version                  print version

actions on this host:
`, version)
	for _, st := range steps.All(sysinfo.Detect()) {
		fmt.Fprintf(os.Stderr, "  %-14s %s\n", st.ID, st.Title)
	}
}
