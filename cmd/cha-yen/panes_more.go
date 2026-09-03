package main

import (
	"fmt"

	"charm.land/bubbles/v2/filepicker"
	"charm.land/bubbles/v2/tree"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ---- styles (lipgloss) ----------------------------------------------------

type stylesPane struct{ i int }

func newStylesPane() pane { return &stylesPane{} }

func (p *stylesPane) name() string             { return "Styles" }
func (p *stylesPane) Init() tea.Cmd            { return nil }
func (p *stylesPane) Focus() tea.Cmd           { return nil }
func (p *stylesPane) Blur()                    {}
func (p *stylesPane) mouseMode() tea.MouseMode { return tea.MouseModeNone }

func (p *stylesPane) Update(msg tea.Msg) (pane, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		if k.String() == "right" || k.String() == "l" {
			p.i++
		}
		if k.String() == "left" || k.String() == "h" {
			p.i--
		}
	}
	return p, nil
}

func (p *stylesPane) View(w, h int) string {
	borderStyles := []lipgloss.Border{
		lipgloss.RoundedBorder(), lipgloss.NormalBorder(),
		lipgloss.DoubleBorder(), lipgloss.ThickBorder(),
		lipgloss.BlockBorder(), lipgloss.ASCIIBorder(),
	}
	b := borderStyles[((p.i%len(borderStyles))+len(borderStyles))%len(borderStyles)]
	box := lipgloss.NewStyle().
		Border(b).BorderForeground(lipgloss.Color("212")).
		Padding(1, 3).Render("rounded/normal/double/thick...")

	palette := ""
	for i := 0; i < 16; i++ {
		palette += lipgloss.NewStyle().
			Background(lipgloss.Color(fmt.Sprintf("%d", i))).
			Foreground(lipgloss.Color("0")).Render("  ")
	}
	gradient := lipgloss.NewStyle().
		Background(lipgloss.Color("#7D56F4")).
		Foreground(lipgloss.Color("#FFD700")).Bold(true).
		Padding(0, 1).Render(" truecolor ")

	align := lipgloss.JoinHorizontal(lipgloss.Center,
		lipgloss.NewStyle().Width(6).Align(lipgloss.Right).Render("right:"),
		lipgloss.NewStyle().Width(10).Align(lipgloss.Left).Render("value"),
	)

	return "←/→ cycles border styles\n\n" + box + "\n\n" +
		"ANSI 16 palette:\n" + palette + "\n\n" +
		"256 ramp: " + ramp() + "\n\n" +
		gradient + "\n\n" + align
}

func ramp() string {
	s := ""
	for i := 0; i < 24; i++ {
		s += lipgloss.NewStyle().Background(lipgloss.Color(fmt.Sprintf("%d", 16+i*8))).Render(" ")
	}
	return s
}

// ---- tree -----------------------------------------------------------------

type treePane struct {
	t tree.Model
}

func newTreePane() pane {
	root := tree.NewNode()
	root.SetValue("chayen/")
	root.Child(
		tree.Root("cmd/").Child(tree.Root("chayen/").Child(
			tree.Root("main.go"), tree.Root("showcase.go"), tree.Root("models.go"),
		)),
		tree.Root("internal/").Child(
			tree.Root("termcore/").Child(tree.Root("engine.go"), tree.Root("engine_test.go")),
			tree.Root("termui/").Child(tree.Root("terminal.go"), tree.Root("keys.go")),
		),
		tree.Root("bubbletea/").Child(tree.Root("tea.go"), tree.Root("options.go")),
	)
	root.Open()
	return &treePane{t: tree.New(root, 0, 0)}
}

func (p *treePane) name() string             { return "Tree" }
func (p *treePane) Init() tea.Cmd            { return nil }
func (p *treePane) Focus() tea.Cmd           { return nil }
func (p *treePane) Blur()                    {}
func (p *treePane) mouseMode() tea.MouseMode { return tea.MouseModeNone }

func (p *treePane) Update(msg tea.Msg) (pane, tea.Cmd) {
	var cmd tea.Cmd
	p.t, cmd = p.t.Update(msg)
	return p, cmd
}

func (p *treePane) View(w, h int) string {
	p.t.SetSize(w, h)
	return p.t.View()
}

// ---- mouse ----------------------------------------------------------------

type mousePane struct {
	lastX, lastY int
	clicks       int
	hits         map[string]int
}

func newMousePane() pane {
	return &mousePane{hits: map[string]int{}}
}

func (p *mousePane) name() string             { return "Mouse" }
func (p *mousePane) Init() tea.Cmd            { return nil }
func (p *mousePane) Focus() tea.Cmd           { return nil }
func (p *mousePane) Blur()                    {}
func (p *mousePane) mouseMode() tea.MouseMode { return tea.MouseModeCellMotion }

func (p *mousePane) Update(msg tea.Msg) (pane, tea.Cmd) {
	if mm, ok := msg.(tea.MouseMsg); ok {
		m := mm.Mouse()
		p.lastX, p.lastY = m.X, m.Y
		if _, isClick := msg.(tea.MouseClickMsg); isClick {
			p.clicks++
			region := p.hitTest(m.X, m.Y)
			p.hits[region]++
		}
	}
	return p, nil
}

// hitTest maps a cell coordinate to one of three colored buttons.
func (p *mousePane) hitTest(x, y int) string {
	if y < 4 || y > 6 {
		return "none"
	}
	switch {
	case x >= 2 && x <= 10:
		return "red"
	case x >= 13 && x <= 21:
		return "green"
	case x >= 24 && x <= 32:
		return "blue"
	}
	return "none"
}

func (p *mousePane) View(w, h int) string {
	btn := func(label string, bg string) string {
		return lipgloss.NewStyle().Background(lipgloss.Color(bg)).
			Foreground(lipgloss.Color("230")).Bold(true).
			Padding(0, 2).Render(label)
	}
	lines := []string{
		"Tap/click the colored buttons (SGR mouse events).",
		"",
		"  " + btn("RED", "#C0392B") + "   " + btn("GREEN", "#27AE60") + "   " + btn("BLUE", "#2980B9"),
		"",
		fmt.Sprintf("  last: (%d,%d)   clicks: %d", p.lastX, p.lastY, p.clicks),
		fmt.Sprintf("  red=%d green=%d blue=%d", p.hits["red"], p.hits["green"], p.hits["blue"]),
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// ---- file picker ----------------------------------------------------------

type filePane struct {
	fp     filepicker.Model
	sel    string
	status string
}

func newFilePane() pane {
	fp := filepicker.New()
	fp.CurrentDirectory = "."
	fp.FileAllowed = true
	fp.DirAllowed = true
	fp.ShowHidden = false
	fp.AutoHeight = false
	return &filePane{fp: fp}
}

func (p *filePane) name() string   { return "Files" }
func (p *filePane) Init() tea.Cmd  { return p.fp.Init() }
func (p *filePane) Focus() tea.Cmd { return nil }
func (p *filePane) Blur()          {}

func (p *filePane) Update(msg tea.Msg) (pane, tea.Cmd) {
	var cmd tea.Cmd
	p.fp, cmd = p.fp.Update(msg)
	if ok, path := p.fp.DidSelectFile(msg); ok {
		p.sel = path
	}
	if ok, path := p.fp.DidSelectDisabledFile(msg); ok {
		p.status = "disabled: " + path
	}
	return p, cmd
}

func (p *filePane) View(w, h int) string {
	p.fp.SetHeight(h - 4)
	v := p.fp.View()
	if p.sel != "" {
		v += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("84")).Render("picked: "+p.sel)
	}
	if p.status != "" {
		v += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render(p.status)
	}
	return v
}

func (p *filePane) mouseMode() tea.MouseMode { return tea.MouseModeNone }
