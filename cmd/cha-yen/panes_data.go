package main

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ---- list -----------------------------------------------------------------

type demoItem struct {
	name string
	desc string
}

func (i demoItem) Title() string       { return i.name }
func (i demoItem) Description() string { return i.desc }
func (i demoItem) FilterValue() string { return i.name }

type listPane struct {
	l   list.Model
	msg string
}

func newListPane() pane {
	items := []list.Item{
		demoItem{"Bubble Tea", "The application framework"},
		demoItem{"Lip Gloss", "Style and align terminal output"},
		demoItem{"Bubbles", "Interactive components"},
		demoItem{"Glamour", "Markdown rendering"},
		demoItem{"Huh", "Interactive forms"},
	}
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "charm stack"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	return &listPane{l: l}
}

func (p *listPane) name() string   { return "List" }
func (p *listPane) Init() tea.Cmd  { return nil }
func (p *listPane) Focus() tea.Cmd { return nil }
func (p *listPane) Blur()          {}

func (p *listPane) Update(msg tea.Msg) (pane, tea.Cmd) {
	var cmd tea.Cmd
	p.l, cmd = p.l.Update(msg)
	if k, ok := msg.(tea.KeyPressMsg); ok && k.String() == "enter" {
		if it := p.l.SelectedItem(); it != nil {
			p.msg = "selected: " + it.(demoItem).Title()
		}
	}
	return p, cmd
}

func (p *listPane) View(w, h int) string {
	p.l.SetSize(w, h-2)
	v := p.l.View()
	if p.msg != "" {
		v += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("84")).Render(p.msg)
	}
	return v
}

// ---- table ----------------------------------------------------------------

type tablePane struct {
	t   table.Model
	msg string
}

func newTablePane() pane {
	cols := []table.Column{
		{Title: "Package", Width: 14},
		{Title: "Stars", Width: 8},
		{Title: "Kind", Width: 12},
	}
	rows := []table.Row{
		{"bubbletea", "42k", "framework"},
		{"lipgloss", "9k", "styling"},
		{"bubbles", "4k", "components"},
		{"huh", "3k", "forms"},
		{"glamour", "9k", "markdown"},
	}
	t := table.New(table.WithColumns(cols), table.WithRows(rows))
	return &tablePane{t: t}
}

func (p *tablePane) name() string   { return "Table" }
func (p *tablePane) Init() tea.Cmd  { return nil }
func (p *tablePane) Focus() tea.Cmd { return nil }
func (p *tablePane) Blur()          {}

func (p *tablePane) Update(msg tea.Msg) (pane, tea.Cmd) {
	var cmd tea.Cmd
	p.t, cmd = p.t.Update(msg)
	if k, ok := msg.(tea.KeyPressMsg); ok && k.String() == "enter" {
		r := p.t.SelectedRow()
		if len(r) > 0 {
			p.msg = "row: " + strings.Join(r, " / ")
		}
	}
	return p, cmd
}

func (p *tablePane) View(w, h int) string {
	p.t.SetWidth(w)
	p.t.SetHeight(h - 2)
	v := p.t.View()
	if p.msg != "" {
		v += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("84")).Render(p.msg)
	}
	return v
}

// ---- viewport -------------------------------------------------------------

type viewportPane struct {
	vp viewport.Model
}

func newViewportPane() pane {
	vp := viewport.New()
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Scroll me: ↑/↓ or j/k, PgUp/PgDn, g/G") + "\n\n")
	for i := 1; i <= 60; i++ {
		b.WriteString(fmt.Sprintf("  line %02d — the quick brown fox jumps over the lazy dog\n", i))
	}
	vp.SetContent(b.String())
	return &viewportPane{vp: vp}
}

func (p *viewportPane) name() string   { return "Viewport" }
func (p *viewportPane) Init() tea.Cmd  { return nil }
func (p *viewportPane) Focus() tea.Cmd { return nil }
func (p *viewportPane) Blur()          {}

func (p *viewportPane) Update(msg tea.Msg) (pane, tea.Cmd) {
	var cmd tea.Cmd
	p.vp, cmd = p.vp.Update(msg)
	return p, cmd
}

func (p *viewportPane) View(w, h int) string {
	p.vp.SetWidth(w)
	p.vp.SetHeight(h)
	return p.vp.View() + "\n" + fmt.Sprintf("at %.0f%%", p.vp.ScrollPercent()*100)
}

func (p *listPane) mouseMode() tea.MouseMode     { return tea.MouseModeNone }
func (p *tablePane) mouseMode() tea.MouseMode    { return tea.MouseModeNone }
func (p *viewportPane) mouseMode() tea.MouseMode { return tea.MouseModeNone }
