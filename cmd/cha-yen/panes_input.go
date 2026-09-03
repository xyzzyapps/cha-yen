package main

import (
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// inputPane demonstrates single-line and multi-line text entry.
type inputPane struct {
	ti textinput.Model
	ta textarea.Model
}

func newInputPane() pane {
	ti := textinput.New()
	ti.Placeholder = "Type your name..."
	ti.Prompt = "name ❯ "
	ti.Validate = func(s string) error {
		if len(s) > 24 {
			return errTooLong
		}
		return nil
	}
	ta := textarea.New()
	ta.Placeholder = "Multi-line notes go here (Ctrl+Q to quit the field on desktop)..."
	ta.Prompt = "│ "
	ta.SetHeight(6)
	return &inputPane{ti: ti, ta: ta}
}

type longStringErr string

func (e longStringErr) Error() string { return string(e) }

const errTooLong = longStringErr("max 24 chars")

var fieldLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Bold(true)

func (p *inputPane) name() string  { return "Input" }
func (p *inputPane) Init() tea.Cmd { return nil }
func (p *inputPane) Focus() tea.Cmd {
	p.ta.Focus()
	return p.ti.Focus()
}
func (p *inputPane) Blur() { p.ti.Blur(); p.ta.Blur() }

func (p *inputPane) Update(msg tea.Msg) (pane, tea.Cmd) {
	var (
		cmds []tea.Cmd
		c    tea.Cmd
	)
	p.ti, c = p.ti.Update(msg)
	cmds = append(cmds, c)
	p.ta, c = p.ta.Update(msg)
	cmds = append(cmds, c)
	return p, tea.Batch(cmds...)
}

func (p *inputPane) View(w, h int) string {
	p.ti.SetWidth(w)
	p.ta.SetWidth(w)
	out := fieldLabel.Render("textinput") + "\n" + p.ti.View() + "\n\n" +
		fieldLabel.Render("textarea") + "\n" + p.ta.View()
	if p.ti.Validate != nil {
		if err := p.ti.Validate(p.ti.Value()); err != nil {
			out += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("160")).Render("⚠ "+err.Error())
		}
	}
	return out
}

func (p *inputPane) mouseMode() tea.MouseMode { return tea.MouseModeNone }
