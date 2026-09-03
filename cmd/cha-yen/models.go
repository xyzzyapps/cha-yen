// Demo Bubble Tea models used to validate the Android terminal port.
// Selected with the -demo flag: counter | list | clock.
package main

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
)

// ---------------------------------------------------------------- counter

type counterModel struct {
	n int
}

func (m counterModel) Init() tea.Cmd { return nil }

func (m counterModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up":
			m.n--
		case "down":
			m.n++
		}
	}
	return m, nil
}

func (m counterModel) View() tea.View {
	return tea.NewView(fmt.Sprintf(
		"  counter demo\n\n  value: %d\n\n  [up/down] change  [q] quit\n", m.n))
}

// -------------------------------------------------------------------- list

type listModel struct {
	items    []string
	cursor   int
	selected map[int]bool
	width    int
}

func newListModel() listModel {
	return listModel{
		items: []string{
			"bubbletea on android",
			"virtual terminal widget",
			"gomobile + fyne packaging",
			"touch keyboard plumbing",
			"sip some bubble tea",
		},
		selected: map[int]bool{},
	}
}

func (m listModel) Init() tea.Cmd { return nil }

func (m listModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter", "space":
			m.selected[m.cursor] = !m.selected[m.cursor]
		}
	}
	return m, nil
}

func (m listModel) View() tea.View {
	s := "  todo (enter toggles, q quits)\n\n"
	for i, item := range m.items {
		box := " "
		if m.selected[i] {
			box = "x"
		}
		cursor := "  "
		if m.cursor == i {
			cursor = "> "
		}
		s += fmt.Sprintf("  %s[%s] %s\n", cursor, box, item)
	}
	return tea.NewView(s)
}

// ------------------------------------------------------------------- clock

type clockModel struct {
	now time.Time
}

func (m clockModel) Init() tea.Cmd { return tick() }

func (m clockModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case time.Time:
		m.now = msg
	}
	return m, tick()
}

func (m clockModel) View() tea.View {
	return tea.NewView("  clock demo (spinner-style redraw)\n\n  " +
		m.now.Format("2006-01-02 15:04:05") + "\n\n  [q] quit\n")
}

func tick() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg { return t })
}
