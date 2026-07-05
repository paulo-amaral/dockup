// Package tui renders dockup's interactive terminal interface (Bubble Tea).
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/paulo-amaral/dockup/v2/internal/steps"
	"github.com/paulo-amaral/dockup/v2/internal/sysinfo"
)

var (
	accent    = lipgloss.AdaptiveColor{Light: "#B4552D", Dark: "#D97757"}
	dim       = lipgloss.AdaptiveColor{Light: "#8A8A8A", Dark: "#6C6C6C"}
	good      = lipgloss.AdaptiveColor{Light: "#2E7D32", Dark: "#7BC77E"}
	warn      = lipgloss.AdaptiveColor{Light: "#B26A00", Dark: "#E5B567"}
	titleSt   = lipgloss.NewStyle().Bold(true).Foreground(accent)
	tagSt     = lipgloss.NewStyle().Foreground(dim)
	panelSt   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(dim).Padding(0, 1)
	selSt     = lipgloss.NewStyle().Bold(true).Foreground(accent)
	descSt    = lipgloss.NewStyle().Foreground(dim)
	statusSt  = lipgloss.NewStyle().Foreground(warn)
	okSt      = lipgloss.NewStyle().Foreground(good)
	sudoBadge = lipgloss.NewStyle().Foreground(warn).Render(" [sudo]")
)

type view int

const (
	viewMenu view = iota
	viewLoading
	viewReport
	viewConfirm
)

type execDoneMsg struct{ err error }

type reportMsg struct {
	title string
	body  string
	err   error
}

type Model struct {
	info        sysinfo.Info
	version     string
	items       []steps.Step
	cursor      int
	view        view
	vp          viewport.Model
	spin        spinner.Model
	status      string
	reportTitle string
	confirm     *steps.Step
	width       int
	height      int
}

func New(info sysinfo.Info, version string) Model {
	sp := spinner.New(spinner.WithSpinner(spinner.Dot))
	sp.Style = lipgloss.NewStyle().Foreground(accent)
	return Model{info: info, version: version, items: steps.All(info), spin: sp}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.vp = viewport.New(msg.Width-4, max(msg.Height-8, 5))
		return m, nil

	case spinner.TickMsg:
		if m.view != viewLoading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd

	case execDoneMsg:
		m.refresh()
		m.view = viewMenu
		if msg.err != nil {
			m.status = "step failed: " + msg.err.Error()
		} else {
			m.status = "done ✓"
		}
		return m, nil

	case reportMsg:
		m.view = viewReport
		m.reportTitle = msg.title
		body := msg.body
		if msg.err != nil {
			body += "\n" + statusSt.Render("error: "+msg.err.Error())
		}
		m.vp.SetContent(body)
		m.vp.GotoTop()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "ctrl+c" {
		return m, tea.Quit
	}

	switch m.view {
	case viewMenu:
		switch key {
		case "q", "esc":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "r":
			m.refresh()
			m.status = "system info refreshed"
		case "enter":
			return m.launch(m.items[m.cursor])
		}

	case viewConfirm:
		switch key {
		case "y", "Y":
			st := *m.confirm
			m.confirm = nil
			return m.run(st)
		case "n", "N", "esc", "q":
			m.confirm = nil
			m.view = viewMenu
			m.status = "cancelled"
		}

	case viewReport:
		switch key {
		case "esc", "q":
			m.view = viewMenu
			m.status = ""
		default:
			var cmd tea.Cmd
			m.vp, cmd = m.vp.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m Model) launch(st steps.Step) (tea.Model, tea.Cmd) {
	if st.NeedsRoot && !m.info.Root {
		m.status = fmt.Sprintf("%q needs root — restart with: sudo dockup", st.Title)
		return m, nil
	}
	if st.Destructive {
		m.confirm = &st
		m.view = viewConfirm
		return m, nil
	}
	return m.run(st)
}

func (m Model) run(st steps.Step) (tea.Model, tea.Cmd) {
	if st.Command != nil {
		// Hand the real terminal to the process (apt prompts, progress bars).
		return m, tea.ExecProcess(st.Command(m.info), func(err error) tea.Msg {
			return execDoneMsg{err}
		})
	}
	m.view = viewLoading
	m.status = ""
	title := st.Title
	report := st.Report
	info := m.info
	return m, tea.Batch(m.spin.Tick, func() tea.Msg {
		body, err := report(info)
		return reportMsg{title: title, body: body, err: err}
	})
}

func (m *Model) refresh() {
	m.info = sysinfo.Detect()
	m.items = steps.All(m.info)
	if m.cursor >= len(m.items) {
		m.cursor = max(len(m.items)-1, 0)
	}
}

func (m Model) View() string {
	if m.width == 0 {
		return "loading..."
	}
	header := lipgloss.JoinHorizontal(lipgloss.Bottom,
		titleSt.Render("⚡ dockup "),
		tagSt.Render(m.version+" — install & maintain container runtimes"),
	)

	var body string
	switch m.view {
	case viewLoading:
		body = "\n " + m.spin.View() + " working...\n"
	case viewReport:
		body = panelSt.Render(m.vp.View()) + "\n" + tagSt.Render("  esc back • ↑/↓ scroll")
	case viewConfirm:
		body = panelSt.Render(fmt.Sprintf("%s\n\n%s\n\nProceed? (y/N)",
			titleSt.Render(m.confirm.Title), m.confirm.Desc))
	default:
		body = lipgloss.JoinVertical(lipgloss.Left, m.infoPanel(), m.menu(),
			tagSt.Render("  ↑/↓ move • enter run • r refresh • q quit"))
	}

	out := lipgloss.JoinVertical(lipgloss.Left, header, "", body)
	if m.status != "" && m.view == viewMenu {
		out += "\n" + statusSt.Render("  "+m.status)
	}
	return out + "\n"
}

func (m Model) infoPanel() string {
	yes := okSt.Render("yes")
	no := tagSt.Render("no")
	b := func(v bool) string {
		if v {
			return yes
		}
		return no
	}
	val := func(v string) string {
		if v == "" {
			return tagSt.Render("not installed")
		}
		return okSt.Render(v)
	}
	host := m.info.DistroName
	if host == "" {
		host = m.info.OS
	}
	lines := []string{
		fmt.Sprintf("host    %s (%s)", host, m.info.Arch),
		fmt.Sprintf("docker  %s   compose %s   podman %s", val(m.info.Docker), val(m.info.Compose), val(m.info.Podman)),
		fmt.Sprintf("root %s   nvidia gpu %s   nvidia-ctk %s", b(m.info.Root), b(m.info.HasGPU), b(m.info.NvidiaCTK)),
	}
	return panelSt.Render(strings.Join(lines, "\n"))
}

func (m Model) menu() string {
	var b strings.Builder
	for i, st := range m.items {
		cursor, title := "  ", st.Title
		if st.NeedsRoot && !m.info.Root {
			title += sudoBadge
		}
		if i == m.cursor {
			cursor = selSt.Render("❯ ")
			b.WriteString(cursor + selSt.Render(title) + "\n")
			b.WriteString("      " + descSt.Render(st.Desc) + "\n")
			continue
		}
		b.WriteString(cursor + title + "\n")
	}
	return b.String()
}
