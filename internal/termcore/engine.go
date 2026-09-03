// Package termcore is the platform-neutral engine that runs a Bubble Tea
// program against a virtual terminal (charmbracelet/x/vt) instead of a real
// TTY. The emulator acts as both the program's input and output; the UI layer
// polls snapshots of the emulator's screen and injects key/text events.
//
// This package must stay free of any GUI dependency so it can be unit-tested
// headlessly on desktop and reused unchanged inside the Android APK.
package termcore

import (
	"fmt"
	"image/color"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	vt "github.com/charmbracelet/x/vt"
)

// Re-exported input types so the UI layer does not have to depend on
// ultraviolet directly for constructing key events.
type (
	// KeyPressEvent is a terminal key press event (code + modifiers).
	KeyPressEvent = uv.KeyPressEvent
	// Mod is a key modifier bitmask (ModCtrl, ModAlt, ...).
	Mod = uv.KeyMod
)

// Modifier constants (aliases of the ultraviolet values).
const (
	ModShift = uv.ModShift
	ModAlt   = uv.ModAlt
	ModCtrl  = uv.ModCtrl
	ModMeta  = uv.ModMeta
)

// Mouse event types re-exported for tap/drag injection from the UI layer.
type (
	// MouseEvent is any terminal mouse event.
	MouseEvent = uv.MouseEvent
	// MouseClickEvent is a button press.
	MouseClickEvent = uv.MouseClickEvent
	// MouseReleaseEvent is a button release.
	MouseReleaseEvent = uv.MouseReleaseEvent
	// MouseMotionEvent is motion with a button state.
	MouseMotionEvent = uv.MouseMotionEvent
	// MouseWheelEvent is a wheel turn.
	MouseWheelEvent = uv.MouseWheelEvent
	// Mouse carries coordinates + button state for mouse events.
	Mouse = uv.Mouse
)

// Mouse buttons (X11 codes).
const (
	MouseLeft       = uv.MouseLeft
	MouseRight      = uv.MouseRight
	MouseMiddle     = uv.MouseMiddle
	MouseWheelUp    = uv.MouseWheelUp
	MouseWheelDown  = uv.MouseWheelDown
	MouseWheelLeft  = uv.MouseWheelLeft
	MouseWheelRight = uv.MouseWheelRight
)

// Special key codes re-exported from ultraviolet for the UI layer.
const (
	KeyEnter     = uv.KeyEnter
	KeyTab       = uv.KeyTab
	KeyBackspace = uv.KeyBackspace
	KeyEscape    = uv.KeyEscape
	KeySpace     = uv.KeySpace
	KeyUp        = uv.KeyUp
	KeyDown      = uv.KeyDown
	KeyLeft      = uv.KeyLeft
	KeyRight     = uv.KeyRight
	KeyHome      = uv.KeyHome
	KeyEnd       = uv.KeyEnd
	KeyPgUp      = uv.KeyPgUp
	KeyPgDown    = uv.KeyPgDown
	KeyDelete    = uv.KeyDelete
	KeyInsert    = uv.KeyInsert
)

// Cell is a render-ready terminal cell snapshot, decoupled from ultraviolet's
// internal representation. Colors are already resolved (defaults applied,
// reverse video swapped).
type Cell struct {
	// Text is the cell's grapheme; empty means blank.
	Text string
	// Fg/Bg are the resolved colors (never nil).
	Fg color.Color
	Bg color.Color
	// Style flags.
	Bold          bool
	Italic        bool
	Underline     bool
	Strikethrough bool
	// Wide marks the first half of a double-width cell; the cell that
	// follows it is a placeholder with WideTail set.
	Wide     bool
	WideTail bool
}

// Screen is an immutable snapshot of the virtual terminal grid.
type Screen struct {
	Cols, Rows  int
	Cells       []Cell // row-major, len == Cols*Rows
	CursorX     int
	CursorY     int
	CursorShown bool
	AltScreen   bool
}

// At returns the cell at (x, y), or a zero cell when out of bounds.
func (s *Screen) At(x, y int) Cell {
	if x < 0 || y < 0 || x >= s.Cols || y >= s.Rows {
		return Cell{}
	}
	return s.Cells[y*s.Cols+x]
}

// Engine runs one Bubble Tea program on a virtual terminal.
type Engine interface {
	// Start launches the program in a background goroutine.
	Start() error
	// Stop requests a graceful program exit and unblocks the input reader.
	Stop()
	// Done is closed when the program has exited.
	Done() <-chan struct{}
	// Err returns the program error (nil unless it crashed or was killed).
	Err() error
	// Resize changes the virtual terminal geometry and notifies the program.
	Resize(cols, rows int)
	// Size returns the current virtual terminal geometry.
	Size() (cols, rows int)
	// SendKey injects a key press event into the terminal input.
	SendKey(k KeyPressEvent)
	// SendText injects printable text (e.g. from a soft keyboard).
	SendText(text string)
	// Paste injects text honoring bracketed-paste mode.
	Paste(text string)
	// SendMouse injects a mouse event (encoded per the program's active
	// mouse-tracking modes).
	SendMouse(m MouseEvent)
	// MouseEnabled reports whether the running program has requested
	// terminal mouse tracking (SGR/X10 modes).
	MouseEnabled() bool
	// Snapshot returns the current screen contents.
	Snapshot() *Screen
	// Changes is signaled (coalesced) whenever the program wrote output.
	Changes() <-chan struct{}
}

type engine struct {
	mu      sync.Mutex
	started bool

	inputMu   sync.Mutex // serializes input injection (independent of screen lock)
	emu       *vt.SafeEmulator
	model     tea.Model
	notify    chan struct{}
	prog      *tea.Program
	done      chan struct{}
	err       error
	cols      int
	rows      int
	log       *slog.Logger
	mouseMode atomic.Int64 // bitmask of active mouse-tracking modes
}

// New creates an Engine for the given model at the initial geometry.
func New(model tea.Model, cols, rows int, log *slog.Logger) Engine {
	if log == nil {
		log = slog.Default()
	}
	emu := vt.NewSafeEmulator(cols, rows)
	// Give the virtual terminal a dark background so lipgloss-styled TUIs
	// look like a real terminal instead of black-on-white.
	emu.SetDefaultBackgroundColor(color.RGBA{R: 26, G: 27, B: 38, A: 255})
	emu.SetDefaultForegroundColor(color.RGBA{R: 192, G: 202, B: 245, A: 255})
	e := &engine{
		emu:    emu,
		model:  model,
		notify: make(chan struct{}, 1),
		done:   make(chan struct{}),
		cols:   cols,
		rows:   rows,
		log:    log,
	}
	emu.Emulator.SetCallbacks(vt.Callbacks{
		EnableMode:  func(m ansi.Mode) { trackMouseMode(&e.mouseMode, m, true) },
		DisableMode: func(m ansi.Mode) { trackMouseMode(&e.mouseMode, m, false) },
	})
	return e
}

// signalWriter forwards writes to the emulator and coalesces a change signal
// so the UI knows when to take a new snapshot.
type signalWriter struct {
	w  io.Writer
	ch chan struct{}
}

func (s *signalWriter) Write(p []byte) (int, error) {
	n, err := s.w.Write(p)
	select {
	case s.ch <- struct{}{}:
	default: // coalesce: a reader will observe the pending signal
	}
	return n, err
}

func (e *engine) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.started {
		return fmt.Errorf("termcore: engine already started")
	}
	e.started = true

	model := e.model
	out := &signalWriter{w: e.emu, ch: e.notify}

	e.prog = tea.NewProgram(model,
		tea.WithInput(e.emu), // emulator.Read yields injected key sequences
		tea.WithOutput(out),  // renderer ANSI stream feeds emulator.Write
		tea.WithWindowSize(e.cols, e.rows),
		tea.WithEnvironment([]string{"TERM=xterm-256color"}),
		tea.WithColorProfile(colorprofile.TrueColor),
		tea.WithoutSignalHandler(), // no SIGWINCH/SIGTSTP inside an APK
	)

	e.log.Info("termcore: engine starting", "cols", e.cols, "rows", e.rows)
	go func() {
		defer close(e.done)
		m, err := e.prog.Run()
		e.mu.Lock()
		e.err = err
		e.mu.Unlock()
		if err != nil {
			e.log.Error("termcore: program exited with error", "err", err)
		} else {
			_ = m
			e.log.Info("termcore: program exited cleanly")
		}
		// Unblock the input read loop so Run can return.
		_ = e.emu.Close()
	}()
	return nil
}

func (e *engine) Stop() {
	e.mu.Lock()
	prog := e.prog
	e.mu.Unlock()
	if prog != nil {
		prog.Quit()
	}
	_ = e.emu.Close()
}

func (e *engine) Done() <-chan struct{} { return e.done }

func (e *engine) Err() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.err
}

func (e *engine) Resize(cols, rows int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if cols < 1 || rows < 1 || (cols == e.cols && rows == e.rows) {
		return
	}
	e.cols, e.rows = cols, rows
	e.emu.Resize(cols, rows)
	if e.started {
		e.prog.Send(tea.WindowSizeMsg{Width: cols, Height: rows})
	}
	e.log.Debug("termcore: resized", "cols", cols, "rows", rows)
}

func (e *engine) Size() (int, int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.cols, e.rows
}

// Input injection uses the raw embedded Emulator (not the SafeEmulator
// wrappers) deliberately: SafeEmulator.SendKey holds the screen mutex while
// doing a BLOCKING io.Pipe write, which deadlocks against the render path
// (Write) that needs the same mutex. Writing to the input pipe touches none
// of the screen-buffer state, so we serialize input with its own mutex.
func (e *engine) SendKey(k KeyPressEvent) {
	e.inputMu.Lock()
	defer e.inputMu.Unlock()
	e.emu.Emulator.SendKey(k)
}
func (e *engine) SendText(text string) {
	e.inputMu.Lock()
	defer e.inputMu.Unlock()
	e.emu.Emulator.SendText(text)
}
func (e *engine) Paste(text string) {
	e.inputMu.Lock()
	defer e.inputMu.Unlock()
	e.emu.Emulator.Paste(text)
}
func (e *engine) SendMouse(m MouseEvent) {
	e.inputMu.Lock()
	defer e.inputMu.Unlock()
	e.emu.Emulator.SendMouse(m)
}
func (e *engine) MouseEnabled() bool { return e.mouseMode.Load() != 0 }

// trackMouseMode maintains a bitmask of active mouse-tracking modes so the
// UI can decide whether taps should be translated into SGR mouse sequences.
func trackMouseMode(bits *atomic.Int64, m ansi.Mode, on bool) {
	var bit int64
	switch m {
	case ansi.ModeMouseX10:
		bit = 1
	case ansi.ModeMouseNormal:
		bit = 2
	case ansi.ModeMouseButtonEvent:
		bit = 4
	case ansi.ModeMouseAnyEvent:
		bit = 8
	default:
		return
	}
	if on {
		bits.Add(bit)
	} else {
		bits.Add(-bit)
	}
}

func (e *engine) Changes() <-chan struct{} { return e.notify }

// Snapshot captures the emulator screen into a render-ready Screen.
func (e *engine) Snapshot() *Screen {
	cols, rows := e.emu.Width(), e.emu.Height()
	s := &Screen{
		Cols:        cols,
		Rows:        rows,
		Cells:       make([]Cell, cols*rows),
		AltScreen:   e.emu.IsAltScreen(),
		CursorShown: true, // refined below when the emulator exposes it
	}
	defFg, defBg := e.emu.ForegroundColor(), e.emu.BackgroundColor()
	cur := e.emu.CursorPosition()
	s.CursorX, s.CursorY = int(cur.X), int(cur.Y)

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			c := e.emu.CellAt(x, y)
			if c == nil {
				continue
			}
			cell := Cell{Text: c.Content}
			st := &c.Style
			fg, bg := st.Fg, st.Bg
			if st.Attrs&uv.AttrReverse != 0 {
				fg, bg = bg, fg
			}
			if st.Attrs&uv.AttrConceal != 0 {
				fg = nil
			}
			cell.Fg = orColor(fg, defFg)
			cell.Bg = orColor(bg, defBg)
			cell.Bold = st.Attrs&uv.AttrBold != 0
			cell.Italic = st.Attrs&uv.AttrItalic != 0
			cell.Underline = st.Underline != uv.UnderlineNone
			cell.Strikethrough = st.Attrs&uv.AttrStrikethrough != 0
			if c.Width == 2 {
				cell.Wide = true
			}
			s.Cells[y*cols+x] = cell
		}
	}
	// Mark placeholders following wide cells.
	for y := 0; y < rows; y++ {
		for x := 0; x < cols-1; x++ {
			if s.Cells[y*cols+x].Wide {
				s.Cells[y*cols+x+1].WideTail = true
			}
		}
	}
	return s
}

func orColor(c, def color.Color) color.Color {
	if c != nil {
		return c
	}
	if def != nil {
		return def
	}
	return color.White
}

var _ Engine = (*engine)(nil)
