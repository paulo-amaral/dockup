// Package steps defines every action dockup can perform. Each step is either
// an exec step (a real terminal command, streamed to the user) or a report
// step (pure Go logic that returns text). The TUI and the headless --yes mode
// share this registry so behavior never diverges between the two.
package steps

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/paulo-amaral/dockup/internal/sysinfo"
)

type Step struct {
	ID          string
	Title       string
	Desc        string
	NeedsRoot   bool
	Destructive bool // TUI asks for confirmation before running
	// Exactly one of Command or Report is set.
	Command func(sysinfo.Info) *exec.Cmd
	Report  func(sysinfo.Info) (string, error)
}

// All returns the steps that make sense on this host, in menu order.
func All(info sysinfo.Info) []Step {
	var s []Step
	if info.OS == "linux" {
		s = append(s, dockerInstallStep(), nvidiaStep(info), podmanStep(), hardenStep(), auditStep())
	}
	if info.OS == "darwin" {
		s = append(s, appleSteps(info)...)
	}
	if info.Docker != "" || info.OS == "linux" {
		s = append(s, statusStep(), pruneStep())
	}
	return s
}

// RunHeadless executes a step by ID without the TUI (--yes mode).
func RunHeadless(id string, info sysinfo.Info) error {
	for _, st := range All(info) {
		if st.ID != id {
			continue
		}
		if st.NeedsRoot && !info.Root {
			return fmt.Errorf("%s requires root, run: sudo dockup --yes %s", st.ID, st.ID)
		}
		if st.Command != nil {
			cmd := st.Command(info)
			cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
			return cmd.Run()
		}
		out, err := st.Report(info)
		if out != "" {
			fmt.Println(out)
		}
		return err
	}
	var ids []string
	for _, st := range All(info) {
		ids = append(ids, st.ID)
	}
	return fmt.Errorf("unknown action %q on this platform, available: %s", id, strings.Join(ids, ", "))
}

// shell wraps a script in `sh -c`. SECURITY CONTRACT: callers must pass only
// compile-time constant strings. Never interpolate user input, env values or
// detected system data into the script; pass those as exec.Command arguments
// in a dedicated step instead.
func shell(script string) *exec.Cmd {
	return exec.Command("sh", "-c", script)
}
