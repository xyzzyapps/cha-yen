package termcore

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// counterModel is a minimal Bubble Tea program used to exercise the engine
// end-to-end: it renders a count, reacts to up/down keys, and quits on 'q'.
type counterModel struct{ n int }

func (m counterModel) Init() tea.Cmd { return nil }

func (m counterModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
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
	return tea.NewView("counter\nvalue: " + itoa(m.n) + "\n[q]uit [up/down]")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}

// screenText flattens a snapshot into newline-joined rows for assertions.
func screenText(s *Screen) string {
	var b strings.Builder
	for y := 0; y < s.Rows; y++ {
		for x := 0; x < s.Cols; x++ {
			c := s.At(x, y)
			if c.Text == "" {
				b.WriteByte(' ')
			} else {
				b.WriteString(c.Text)
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// waitForScreen polls snapshots until cond is satisfied or timeout.
func waitForScreen(t *testing.T, e Engine, cond func(*Screen) bool) *Screen {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		s := e.Snapshot()
		if cond(s) {
			return s
		}
		select {
		case <-e.Changes():
		case <-time.After(20 * time.Millisecond):
		}
	}
	t.Fatalf("timed out waiting for screen condition; last screen:\n%s", screenText(e.Snapshot()))
	return nil
}

func TestEngineRendersProgram(t *testing.T) {
	e := New(counterModel{}, 40, 10, nil)
	if err := e.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer e.Stop()

	s := waitForScreen(t, e, func(s *Screen) bool {
		return strings.Contains(screenText(s), "value: 0")
	})
	if s.Cols != 40 || s.Rows != 10 {
		t.Fatalf("geometry = %dx%d, want 40x10", s.Cols, s.Rows)
	}
}

func TestEngineKeyInput(t *testing.T) {
	e := New(counterModel{2}, 40, 10, nil)
	if err := e.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer e.Stop()

	waitForScreen(t, e, func(s *Screen) bool {
		return strings.Contains(screenText(s), "value: 2")
	})

	e.SendKey(KeyPressEvent{Code: KeyDown}) // down -> increments
	s := waitForScreen(t, e, func(s *Screen) bool {
		return strings.Contains(screenText(s), "value: 3")
	})
	_ = s
}

func TestEngineTextInputAndQuit(t *testing.T) {
	e := New(counterModel{}, 40, 10, nil)
	if err := e.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	waitForScreen(t, e, func(s *Screen) bool {
		return strings.Contains(screenText(s), "counter")
	})

	e.SendText("q")
	select {
	case <-e.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("program did not exit after 'q'")
	}
	if err := e.Err(); err != nil {
		t.Fatalf("program error: %v", err)
	}
}

func TestEngineResize(t *testing.T) {
	e := New(counterModel{}, 40, 10, nil)
	if err := e.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer e.Stop()
	waitForScreen(t, e, func(s *Screen) bool { return s.Cols == 40 })

	e.Resize(60, 20)
	waitForScreen(t, e, func(s *Screen) bool {
		return s.Cols == 60 && s.Rows == 20 && strings.Contains(screenText(s), "counter")
	})
}
