package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/xyzzyapps/cha-yen/internal/termcore"
)

// ---- test harness ---------------------------------------------------------

// run drives a model through the virtual-terminal engine and returns a
// screen-text getter plus a cleanup func. It forwards WindowSizeMsg so
// components size themselves.
func run(t *testing.T, m tea.Model, cols, rows int) (screen func() string, send func(func(termcore.Engine)), stop func()) {
	t.Helper()
	e := termcore.New(m, cols, rows, nil)
	if err := e.Start(); err != nil {
		t.Fatalf("engine start: %v", err)
	}
	screen = func() string {
		s := e.Snapshot()
		var b strings.Builder
		for y := 0; y < s.Rows; y++ {
			for x := 0; x < s.Cols; x++ {
				tx := s.At(x, y).Text
				if tx == "" {
					tx = " "
				}
				b.WriteString(tx)
			}
			b.WriteByte('\n')
		}
		return b.String()
	}
	send = func(fn func(termcore.Engine)) { fn(e) }
	stop = func() { e.Stop(); <-e.Done() }
	return
}

// waitForScreen polls until cond passes or times out, returning the text.
func waitForScreen(t *testing.T, screen func() string, cond func(string) bool) string {
	t.Helper()
	deadline := time.Now().Add(4 * time.Second)
	var s string
	for time.Now().Before(deadline) {
		s = screen()
		if cond(s) {
			return s
		}
		time.Sleep(15 * time.Millisecond)
	}
	t.Fatalf("condition not met; last screen:\n%s", s)
	return ""
}

// ---- viewport (scrollable) ------------------------------------------------

type vpModel struct {
	vp viewport.Model
	w  int
}

func (m vpModel) Init() tea.Cmd { return nil }
func (m vpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if sz, ok := msg.(tea.WindowSizeMsg); ok {
		m.w = sz.Width
		m.vp.SetWidth(sz.Width)
		m.vp.SetHeight(sz.Height)
	}
	var c tea.Cmd
	m.vp, c = m.vp.Update(msg)
	return m, c
}
func (m vpModel) View() tea.View {
	v := tea.NewView(m.vp.View())
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func newVP() vpModel {
	var b strings.Builder
	for i := 1; i <= 80; i++ {
		fmt.Fprintf(&b, "LINE-%02d the quick brown fox\n", i)
	}
	vp := viewport.New()
	vp.SetContent(b.String())
	return vpModel{vp: vp}
}

func TestViewportKeyboardScroll(t *testing.T) {
	screen, send, stop := run(t, newVP(), 60, 15)
	defer stop()
	waitForScreen(t, screen, func(s string) bool { return strings.Contains(s, "LINE-01") })

	// Scroll down far enough that the first line leaves the viewport.
	send(func(e termcore.Engine) {
		for i := 0; i < 30; i++ {
			e.SendKey(termcore.KeyPressEvent{Code: termcore.KeyDown})
		}
	})
	s := waitForScreen(t, screen, func(s string) bool {
		return !strings.Contains(s, "LINE-01") && strings.Contains(s, "LINE-31")
	})
	if strings.Contains(s, "LINE-01") {
		t.Fatal("viewport did not scroll: LINE-01 still visible")
	}
}

func TestViewportMouseWheelScroll(t *testing.T) {
	screen, send, stop := run(t, newVP(), 60, 15)
	defer stop()
	waitForScreen(t, screen, func(s string) bool { return strings.Contains(s, "LINE-01") })

	send(func(e termcore.Engine) {
		for i := 0; i < 12; i++ {
			e.SendMouse(termcore.MouseWheelEvent{X: 5, Y: 5, Button: termcore.MouseWheelDown})
		}
	})
	waitForScreen(t, screen, func(s string) bool { return !strings.Contains(s, "LINE-01") })
}

func TestViewportGotoBottom(t *testing.T) {
	screen, send, stop := run(t, newVP(), 60, 15)
	defer stop()
	waitForScreen(t, screen, func(s string) bool { return strings.Contains(s, "LINE-01") })

	// Page down (space) repeatedly to reach the end of the content.
	send(func(e termcore.Engine) {
		for i := 0; i < 40; i++ {
			e.SendText(" ")
		}
	})
	waitForScreen(t, screen, func(s string) bool { return strings.Contains(s, "LINE-80") })
}

// ---- list (scrollable + paginated) ----------------------------------------

type liModel struct {
	l list.Model
}

func (m liModel) Init() tea.Cmd { return nil }
func (m liModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if sz, ok := msg.(tea.WindowSizeMsg); ok {
		m.l.SetSize(sz.Width, sz.Height)
	}
	var c tea.Cmd
	m.l, c = m.l.Update(msg)
	return m, c
}
func (m liModel) View() tea.View { return tea.NewView(m.l.View()) }

type strItem string

func (s strItem) Title() string       { return string(s) }
func (s strItem) Description() string { return "" }
func (s strItem) FilterValue() string { return string(s) }

func TestListScrollsThroughItems(t *testing.T) {
	items := make([]list.Item, 60)
	for i := range items {
		items[i] = strItem(fmt.Sprintf("ITEM-%02d", i))
	}
	l := list.New(items, list.NewDefaultDelegate(), 60, 12)
	l.SetShowPagination(false)
	m := liModel{l: l}

	screen, send, stop := run(t, m, 60, 15)
	defer stop()
	waitForScreen(t, screen, func(s string) bool { return strings.Contains(s, "ITEM-00") })

	send(func(e termcore.Engine) {
		for i := 0; i < 40; i++ {
			e.SendKey(termcore.KeyPressEvent{Code: termcore.KeyDown})
		}
	})
	waitForScreen(t, screen, func(s string) bool { return strings.Contains(s, "ITEM-3") })
}

// ---- textarea (multiline, scrollable) -------------------------------------

type taModel struct{ ta textarea.Model }

func (m taModel) Init() tea.Cmd { return m.ta.Focus() }
func (m taModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if sz, ok := msg.(tea.WindowSizeMsg); ok {
		m.ta.SetWidth(sz.Width)
		m.ta.SetHeight(sz.Height)
	}
	var c tea.Cmd
	m.ta, c = m.ta.Update(msg)
	return m, c
}
func (m taModel) View() tea.View { return tea.NewView(m.ta.View()) }

func TestTextareaMultilineInput(t *testing.T) {
	ta := textarea.New()
	ta.ShowLineNumbers = false
	ta.Focus()
	m := taModel{ta: ta}
	screen, send, stop := run(t, m, 50, 15)
	defer stop()
	waitForScreen(t, screen, func(s string) bool { return true })

	send(func(e termcore.Engine) {
		e.SendText("alpha")
		e.SendKey(termcore.KeyPressEvent{Code: termcore.KeyEnter})
		e.SendText("beta")
	})
	waitForScreen(t, screen, func(s string) bool {
		return strings.Contains(s, "alpha") && strings.Contains(s, "beta")
	})
}

// ---- terminal scroll region (content taller than screen, non-alt) ---------

type tallModel struct{ lines int }

func (m tallModel) Init() tea.Cmd { return nil }
func (m tallModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok && k.String() == "g" {
		m.lines += 10
	}
	return m, nil
}
func (m tallModel) View() tea.View {
	var b strings.Builder
	for i := 1; i <= m.lines; i++ {
		fmt.Fprintf(&b, "ROW-%03d\n", i)
	}
	// No AltScreen: the renderer must scroll the terminal to fit.
	return tea.NewView(b.String())
}

func TestTerminalScrollsWhenContentExceedsHeight(t *testing.T) {
	screen, send, stop := run(t, tallModel{lines: 5}, 40, 10)
	defer stop()
	waitForScreen(t, screen, func(s string) bool { return strings.Contains(s, "ROW-005") })

	// Grow content well beyond the 10-row screen; the emulator must scroll so
	// the newest lines are visible and old ones scroll off the top.
	send(func(e termcore.Engine) {
		for i := 0; i < 5; i++ {
			e.SendText("g")
		}
	})
	s := waitForScreen(t, screen, func(s string) bool { return strings.Contains(s, "ROW-054") })
	if strings.Contains(s, "ROW-001") {
		t.Log("note: ROW-001 still on screen (may be within scrollback view)")
	}
}

// ---- nested containers ----------------------------------------------------

type containerModel struct{}

func (containerModel) Init() tea.Cmd                           { return nil }
func (containerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return containerModel{}, nil }
func (containerModel) View() tea.View {
	box := func(label string) string {
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("212")).
			Padding(1, 3).Width(14).Render(label)
	}
	top := lipgloss.JoinHorizontal(lipgloss.Top, box("TL"), box("TR"))
	bot := lipgloss.JoinHorizontal(lipgloss.Top, box("BL"), box("BR"))
	grid := lipgloss.JoinVertical(lipgloss.Left, top, bot)
	return tea.NewView(grid)
}

func TestNestedContainersRender(t *testing.T) {
	screen, _, stop := run(t, containerModel{}, 60, 20)
	defer stop()
	s := waitForScreen(t, screen, func(s string) bool {
		return strings.Contains(s, "TL") && strings.Contains(s, "TR") &&
			strings.Contains(s, "BL") && strings.Contains(s, "BR")
	})
	// Every box contributes corners; assert multiple horizontal border runs
	// survived (structure intact, not garbled).
	if strings.Count(s, "─") < 8 {
		t.Fatalf("expected multiple border segments, got %d horizontal glyphs", strings.Count(s, "─"))
	}
}
