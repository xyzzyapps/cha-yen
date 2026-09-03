// Showcase demo: exercises every major Bubble Tea + Bubbles widget through
// the virtual-terminal engine. Navigation: tab/shift-tab or left/right or
// 1-0 to switch panes; q quits. Each pane owns real component sub-models.
package main

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// pane is the interface every showcase pane implements. Panes are stored as
// interface values so the model can swap the concrete sub-model generically.
type pane interface {
	name() string
	Init() tea.Cmd
	Update(msg tea.Msg) (pane, tea.Cmd)
	View(width, height int) string
	Focus() tea.Cmd
	Blur()
	mouseMode() tea.MouseMode
}

type showcaseModel struct {
	panes    []pane
	tab      int
	width    int
	height   int
	quitting bool
	help     help.Model
}

type showKeys struct {
	Next key.Binding
	Prev key.Binding
	Quit key.Binding
}

func (k showKeys) ShortHelp() []key.Binding { return []key.Binding{k.Next, k.Prev, k.Quit} }
func (k showKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Next, k.Prev, k.Quit}}
}

var showKeyMap = showKeys{
	Next: key.NewBinding(key.WithKeys("right", "tab", "l"), key.WithHelp("→/tab", "next")),
	Prev: key.NewBinding(key.WithKeys("left", "shift+tab", "h"), key.WithHelp("←/S-tab", "prev")),
	Quit: key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

func newShowcase() showcaseModel {
	h := help.New()
	h.ShowAll = false
	return showcaseModel{
		panes: []pane{
			newInputPane(), newListPane(), newTablePane(), newViewportPane(),
			newWidgetsPane(), newControlsPane(), newStylesPane(), newTreePane(),
			newMousePane(), newFilePane(),
		},
		help: h,
	}
}

func (m showcaseModel) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, p := range m.panes {
		if c := p.Init(); c != nil {
			cmds = append(cmds, c)
		}
	}
	cmds = append(cmds, m.panes[m.tab].Focus())
	return tea.Batch(cmds...)
}

func (m showcaseModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, showKeyMap.Quit):
			m.quitting = true
			return m, tea.Quit
		case key.Matches(msg, showKeyMap.Next):
			return m.gotoTab((m.tab + 1) % len(m.panes))
		case key.Matches(msg, showKeyMap.Prev):
			return m.gotoTab((m.tab - 1 + len(m.panes)) % len(m.panes))
		}
		if s := msg.String(); len(s) == 1 && s[0] >= '1' && s[0] <= '9' {
			if n := int(s[0] - '1'); n < len(m.panes) {
				return m.gotoTab(n)
			}
		}
	}
	p, cmd := m.panes[m.tab].Update(msg)
	m.panes[m.tab] = p
	return m, cmd
}

func (m showcaseModel) gotoTab(n int) (tea.Model, tea.Cmd) {
	m.panes[m.tab].Blur()
	m.tab = n
	return m, m.panes[n].Focus()
}

var (
	tabActive   = lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("212")).Foreground(lipgloss.Color("230")).Padding(0, 1)
	tabInactive = lipgloss.NewStyle().Foreground(lipgloss.Color("247")).Padding(0, 1)
	bodyStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("241")).Padding(1, 2)
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
)

// tabStrip renders a horizontal tab bar, wrapping onto multiple lines so it
// always fits the available width (10 panes do not fit one line on a phone).
func (m showcaseModel) tabStrip() string {
	var lines []string
	var cur []string
	curW := 0
	for i, p := range m.panes {
		s := tabInactive
		if i == m.tab {
			s = tabActive
		}
		label := fmt.Sprintf("%d·%s", i+1, p.name())
		tw := lipgloss.Width(label) + 2 // padding(0,1)
		if curW+tw > m.width && len(cur) > 0 {
			lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Top, cur...))
			cur, curW = nil, 0
		}
		cur = append(cur, s.Render(label))
		curW += tw
	}
	if len(cur) > 0 {
		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Top, cur...))
	}
	return strings.Join(lines, "\n")
}

func (m showcaseModel) View() tea.View {
	if m.quitting {
		return tea.NewView("showcase: bye\n")
	}
	w, h := m.width, m.height
	if w == 0 {
		w, h = 80, 28
	}
	inner := w - 8
	if inner < 20 {
		inner = 20
	}
	paneH := h - 9
	if paneH < 3 {
		paneH = 3
	}
	content := titleStyle.Render("Cha-Yen widget showcase") + "\n" +
		m.tabStrip() + "\n" +
		bodyStyle.Width(w-2).Height(paneH).Render(m.panes[m.tab].View(inner, paneH)) + "\n" +
		m.help.View(showKeyMap)
	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = m.panes[m.tab].mouseMode()
	return v
}
