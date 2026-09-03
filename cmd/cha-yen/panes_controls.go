package main

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// controlsPane hand-rolls checkbox, radio, and select widgets. We avoid the
// `huh` form library because it pulls in creack/pty (a runtime PTY), which is
// undesirable inside an Android APK. These controls are simple and cover the
// same interaction patterns.
type controlsPane struct {
	checks   []checkItem
	checkIdx int
	radios   []string
	radioIdx int
	sels     []string
	selIdx   int
	open     bool
}

type checkItem struct {
	label    string
	checked  bool
	disabled bool
}

func newControlsPane() pane {
	return &controlsPane{
		checks: []checkItem{
			{label: "Enable sync"},
			{label: "Show hidden files"},
			{label: "Legacy mode", disabled: true},
		},
		radios:   []string{"Auto", "Light", "Dark"},
		sels:     []string{"Go", "Rust", "Zig", "Nim", "Odin"},
		radioIdx: 2,
	}
}

func (p *controlsPane) name() string   { return "Controls" }
func (p *controlsPane) Init() tea.Cmd  { return nil }
func (p *controlsPane) Focus() tea.Cmd { return nil }
func (p *controlsPane) Blur()          { p.open = false }

var (
	ctlLabel  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	ctlCursor = lipgloss.NewStyle().Foreground(lipgloss.Color("84"))
	ctlDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	ctlOn     = lipgloss.NewStyle().Foreground(lipgloss.Color("84"))
)

func (p *controlsPane) Update(msg tea.Msg) (pane, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return p, nil
	}
	switch k.String() {
	case "up", "k":
		if p.checkIdx > 0 {
			p.checkIdx--
		}
	case "down", "j":
		if p.checkIdx < len(p.checks)-1 {
			p.checkIdx++
		}
	case "space", "enter":
		if c := &p.checks[p.checkIdx]; !c.disabled {
			c.checked = !c.checked
		}
	case "left", "h":
		p.radioIdx = (p.radioIdx - 1 + len(p.radios)) % len(p.radios)
		if p.open {
			p.selIdx = (p.selIdx - 1 + len(p.sels)) % len(p.sels)
		}
	case "right", "l":
		p.radioIdx = (p.radioIdx + 1) % len(p.radios)
		if p.open {
			p.selIdx = (p.selIdx + 1) % len(p.sels)
		}
	case "o":
		p.open = !p.open
	}
	return p, nil
}

func (p *controlsPane) View(w, h int) string {
	out := ctlLabel.Render("checkboxes") + "  (space toggles)\n"
	for i, c := range p.checks {
		box := "[ ]"
		if c.checked {
			box = "[x]"
		}
		s := ctlDim
		if i == p.checkIdx {
			s = ctlCursor
		}
		if c.disabled {
			out += s.Render("  "+box+" "+c.label+" (disabled)") + "\n"
		} else {
			out += s.Render("  "+box+" "+c.label) + "\n"
		}
	}

	out += "\n" + ctlLabel.Render("radio") + "  (←/→ picks)\n  "
	for i, r := range p.radios {
		if i == p.radioIdx {
			out += ctlOn.Render("(o) " + r + "  ")
		} else {
			out += ctlDim.Render("( ) " + r + "  ")
		}
	}

	out += "\n\n" + ctlLabel.Render("select") + "  (o opens, ←/→ chooses)\n  "
	out += ctlCursor.Render("› " + p.sels[p.selIdx])
	if p.open {
		out += "\n"
		for i, s := range p.sels {
			if i == p.selIdx {
				out += ctlOn.Render("    • " + s + "\n")
			} else {
				out += ctlDim.Render("      " + s + "\n")
			}
		}
	}
	return out
}

func (p *controlsPane) mouseMode() tea.MouseMode { return tea.MouseModeNone }
