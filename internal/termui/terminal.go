// Package termui renders a termcore.Engine screen inside a Fyne
// widget built on top of widget.TextGrid, and captures keyboard and
// touch input to feed back into the engine.
package termui

import (
	"image/color"
	"log/slog"
	"sync"
	"time"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/mobile"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/xyzzyapps/cha-yen/internal/termcore"
)

// Terminal is a Fyne widget that displays and controls a Bubble Tea
// program running on a virtual terminal engine.
type Terminal struct {
	widget.BaseWidget

	engine    termcore.Engine
	grid      *widget.TextGrid
	input     *keyCatcher
	toolbar   *fyne.Container
	baseTheme fyne.Theme

	log *slog.Logger

	mu          sync.Mutex
	last        *termcore.Screen
	styleKey    map[styleKey]*widget.CustomTextGridStyle
	cellW       float32
	cellH       float32
	targetCols  int     // fixed logical columns to fit the window width
	text        float32 // current scaled font size
	appliedText float32 // font size last pushed to the theme

	pumpOnce sync.Once
	stopPump chan struct{}
}

// fitTheme wraps a base theme and overrides the text size so a fixed number
// of terminal columns exactly fills the window width (the font scales to the
// device instead of the grid being derived from a fixed font size).
type fitTheme struct {
	base fyne.Theme
	text float32
}

func (t fitTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	return t.base.Color(n, v)
}
func (t fitTheme) Font(s fyne.TextStyle) fyne.Resource     { return t.base.Font(s) }
func (t fitTheme) Icon(n fyne.ThemeIconName) fyne.Resource { return t.base.Icon(n) }
func (t fitTheme) Size(n fyne.ThemeSizeName) float32 {
	if n == theme.SizeNameText {
		return t.text
	}
	return t.base.Size(n)
}

// NewTerminal creates a terminal view bound to the given engine.
func NewTerminal(e termcore.Engine, log *slog.Logger) *Terminal {
	if log == nil {
		log = slog.Default()
	}
	t := &Terminal{
		engine:   e,
		grid:     widget.NewTextGrid(),
		log:      log,
		styleKey: make(map[styleKey]*widget.CustomTextGridStyle),
		stopPump: make(chan struct{}),
	}
	t.grid.Scroll = fyne.ScrollNone
	t.input = &keyCatcher{term: t}
	t.input.ExtendBaseWidget(t.input)
	t.toolbar = t.buildToolbar()
	// Lock the logical column count to the engine's initial width and scale
	// the font so those columns fill whatever device we run on.
	t.targetCols, _ = e.Size()
	if t.targetCols < 40 {
		t.targetCols = 80
	}
	if settings := fyne.CurrentApp().Settings(); settings != nil {
		t.baseTheme = settings.Theme()
		t.text = t.baseTheme.Size(theme.SizeNameText)
		t.appliedText = t.text
		settings.SetTheme(fitTheme{base: t.baseTheme, text: t.text})
	}
	t.ExtendBaseWidget(t)
	return t
}

// Input returns the focusable object that receives keyboard events;
// focus it (win.Canvas().Focus(term.Input())) to start capturing keys
// and raise the soft keyboard on mobile.
func (t *Terminal) Input() fyne.Focusable { return t.input }

// Stop ends the screen-update pump goroutine.
func (t *Terminal) Stop() { close(t.stopPump) }

// ---------------------------------------------------------------- renderer

type terminalRenderer struct {
	t *Terminal
}

func (t *Terminal) CreateRenderer() fyne.WidgetRenderer {
	r := &terminalRenderer{t: t}
	t.startPump()
	return r
}

func (r *terminalRenderer) Objects() []fyne.CanvasObject {
	// input is listed last so it sits on top of the grid and receives
	// taps; it paints nothing itself.
	return []fyne.CanvasObject{r.t.grid, r.t.toolbar, r.t.input}
}

// Layout places the extra-keys toolbar at the TOP so the Android soft
// keyboard (which overlays the bottom of the window) never covers it.
func (r *terminalRenderer) Layout(size fyne.Size) {
	tb := r.t.toolbar.MinSize().Height
	r.t.toolbar.Resize(fyne.NewSize(size.Width, tb))
	r.t.toolbar.Move(fyne.NewPos(0, 0))
	gs := fyne.NewSize(size.Width, size.Height-tb)
	r.t.grid.Resize(gs)
	r.t.grid.Move(fyne.NewPos(0, tb))
	r.t.input.Resize(gs)
	r.t.input.Move(fyne.NewPos(0, tb))
	r.t.recomputeGeometry()
}

func (r *terminalRenderer) MinSize() fyne.Size {
	return fyne.NewSize(20*r.t.cellWidth(), 10*r.t.cellHeight()+40)
}

func (r *terminalRenderer) Refresh() { r.t.grid.Refresh() }

func (r *terminalRenderer) Destroy() {}

// ---------------------------------------------------------------- geometry

func (t *Terminal) cellWidth() float32 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.cellW
}

func (t *Terminal) cellHeight() float32 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.cellH
}

// recomputeGeometry derives the cell grid from the actual window pixel size so
// the terminal always fills the screen edge-to-edge. cols/rows come from the
// rendered cell metrics; the content adapts to whatever grid results.
func (t *Terminal) recomputeGeometry() {
	t.mu.Lock()
	m := fyne.MeasureText("M", t.text, fyne.TextStyle{Monospace: true})
	t.cellW, t.cellH = m.Width, m.Height
	w := t.Size().Width
	h := t.Size().Height - t.toolbar.MinSize().Height
	cols := int(w / t.cellW)
	rows := int(h / t.cellH)
	t.mu.Unlock()

	if cols < 20 {
		cols = 20
	}
	if rows < 5 {
		rows = 5
	}
	ecols, erows := t.engine.Size()
	if cols != ecols || rows != erows {
		t.log.Info("termui: geometry changed", "cols", cols, "rows", rows, "pxw", w, "cellw", t.cellW)
		t.engine.Resize(cols, rows)
	}
}

// ---------------------------------------------------------------- pump

// startPump runs the snapshot -> TextGrid update loop. It waits for engine
// change signals (coalesced) with a 30fps floor so spinner animations stay
// smooth without busy-waiting.
func (t *Terminal) startPump() {
	t.pumpOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(33 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-t.engine.Changes():
				case <-ticker.C:
				case <-t.stopPump:
					return
				}
				t.render()
			}
		}()
	})
}

// render takes a snapshot and pushes changed rows into the TextGrid.
// It runs on the pump goroutine and marshals widget updates via fyne.Do.
func (t *Terminal) render() {
	snap := t.engine.Snapshot()

	t.mu.Lock()
	last := t.last
	changed := make([]int, 0, snap.Rows)
	if last == nil || last.Cols != snap.Cols || last.Rows != snap.Rows {
		for y := 0; y < snap.Rows; y++ {
			changed = append(changed, y)
		}
	} else {
		for y := 0; y < snap.Rows; y++ {
			if rowDiffers(last, snap, y) {
				changed = append(changed, y)
			}
		}
	}
	if len(changed) == 0 {
		t.mu.Unlock()
		return
	}
	t.last = snap
	rows := make([]widget.TextGridRow, len(changed))
	for i, y := range changed {
		rows[i] = t.rowToGrid(snap, y)
	}
	t.mu.Unlock()

	fyne.Do(func() {
		t.applyRows(changed, rows, snap)
		t.grid.Refresh()
	})
}

// applyRows writes the changed rows into the grid (main thread only).
func (t *Terminal) applyRows(changed []int, rows []widget.TextGridRow, snap *termcore.Screen) {
	for len(t.grid.Rows) < snap.Rows {
		t.grid.Rows = append(t.grid.Rows, widget.TextGridRow{})
	}
	for i, y := range changed {
		if y < len(t.grid.Rows) {
			t.grid.Rows[y] = rows[i]
		}
	}
	if len(t.grid.Rows) > snap.Rows {
		t.grid.Rows = t.grid.Rows[:snap.Rows]
	}
}

func rowDiffers(a, b *termcore.Screen, y int) bool {
	off := y * a.Cols
	for x := 0; x < a.Cols; x++ {
		if a.Cells[off+x] != b.Cells[off+x] {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------- styles

type styleKey struct {
	fg, bg                  color.RGBA
	bold, italic, underline bool
}

// style returns a cached style for the given key (caller holds t.mu).
func (t *Terminal) style(k styleKey) *widget.CustomTextGridStyle {
	if s, ok := t.styleKey[k]; ok {
		return s
	}
	s := &widget.CustomTextGridStyle{
		TextStyle: fyne.TextStyle{Monospace: true, Bold: k.bold, Italic: k.italic, Underline: k.underline},
		FGColor:   k.fg,
		BGColor:   k.bg,
	}
	t.styleKey[k] = s
	return s
}

func toRGBA(c color.Color) color.RGBA {
	if c == nil {
		return color.RGBA{}
	}
	r, g, b, a := c.RGBA()
	return color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
}

// rowToGrid converts one snapshot row into a TextGrid row.
func (t *Terminal) rowToGrid(s *termcore.Screen, y int) widget.TextGridRow {
	cells := make([]widget.TextGridCell, s.Cols)
	for x := 0; x < s.Cols; x++ {
		c := s.At(x, y)
		r := rune(' ')
		if c.Text != "" && !c.WideTail {
			// c.Text is a grapheme cluster string; take its first RUNE
			// (decoding UTF-8), not its first byte, so multi-byte glyphs
			// like box-drawing borders and arrows render correctly.
			if rr, _ := utf8.DecodeRuneInString(c.Text); rr != utf8.RuneError {
				r = rr
			}
		}
		k := styleKey{
			fg:        toRGBA(c.Fg),
			bg:        toRGBA(c.Bg),
			bold:      c.Bold,
			italic:    c.Italic,
			underline: c.Underline,
		}
		cells[x] = widget.TextGridCell{Rune: r, Style: t.style(k)}
	}
	return widget.TextGridRow{Cells: cells}
}

// ---------------------------------------------------------------- toolbar

// buildToolbar creates the extra-keys row needed on touch devices where
// the soft keyboard cannot produce Esc/Tab/Ctrl/arrows.
func (t *Terminal) buildToolbar() *fyne.Container {
	mk := func(label string, fn func()) *widget.Button {
		b := widget.NewButton(label, fn)
		b.Importance = widget.MediumImportance
		return b
	}
	key := func(k termcore.KeyPressEvent) func() {
		return func() { t.engine.SendKey(k) }
	}
	ctrl := mk("Ctrl", func() {
		t.input.mu.Lock()
		t.input.ctrlArmed = !t.input.ctrlArmed
		armed := t.input.ctrlArmed
		t.input.mu.Unlock()
		t.log.Debug("termui: ctrl armed", "armed", armed)
	})
	return container.NewHBox(
		mk("Esc", key(termcore.KeyPressEvent{Code: termcore.KeyEscape})),
		mk("Tab", key(termcore.KeyPressEvent{Code: termcore.KeyTab})),
		ctrl,
		mk("PgUp", key(termcore.KeyPressEvent{Code: termcore.KeyPgUp})),
		mk("PgDn", key(termcore.KeyPressEvent{Code: termcore.KeyPgDown})),
		mk("↑", key(termcore.KeyPressEvent{Code: termcore.KeyUp})),
		mk("↓", key(termcore.KeyPressEvent{Code: termcore.KeyDown})),
		mk("←", key(termcore.KeyPressEvent{Code: termcore.KeyLeft})),
		mk("→", key(termcore.KeyPressEvent{Code: termcore.KeyRight})),
	)
}

// ---------------------------------------------------------------- input

// keyCatcher is an invisible focusable widget that owns keyboard focus,
// raises the soft keyboard on mobile, and forwards events to the engine.
type keyCatcher struct {
	widget.BaseWidget
	term *Terminal

	mu        sync.Mutex
	ctrlArmed bool
	dragMu    sync.Mutex
	lastDrag  termcore.Mouse
}

func (k *keyCatcher) Keyboard() mobile.KeyboardType { return mobile.SingleLineKeyboard }

func (k *keyCatcher) FocusGained() {}
func (k *keyCatcher) FocusLost()   {}

func (k *keyCatcher) TypedKey(ev *fyne.KeyEvent) {
	k.mu.Lock()
	ctrl := k.ctrlArmed
	k.ctrlArmed = false
	k.mu.Unlock()

	if msg, ok := mapKey(ev, ctrl); ok {
		k.term.engine.SendKey(msg)
	}
}

func (k *keyCatcher) TypedRune(r rune) {
	k.mu.Lock()
	ctrl := k.ctrlArmed
	k.ctrlArmed = false
	k.mu.Unlock()

	msg, text, ok := mapRune(r, ctrl)
	if !ok {
		return
	}
	if text != "" {
		k.term.engine.SendText(text)
	} else {
		k.term.engine.SendKey(msg)
	}
}

func (k *keyCatcher) cellPos(p *fyne.PointEvent) (x, y int) {
	k.term.mu.Lock()
	cw, ch := k.term.cellW, k.term.cellH
	k.term.mu.Unlock()
	if cw == 0 || ch == 0 {
		return 0, 0
	}
	return int(p.Position.X / cw), int(p.Position.Y / ch)
}

// Tapped focuses the terminal (raising the soft keyboard) and, when the
// running program has enabled terminal mouse tracking, injects an SGR
// click/release pair at the tapped cell.
func (k *keyCatcher) Tapped(e *fyne.PointEvent) {
	if c := fyne.CurrentApp().Driver().CanvasForObject(k.term); c != nil {
		c.Focus(k)
	}
	if !k.term.engine.MouseEnabled() {
		return
	}
	x, y := k.cellPos(e)
	k.term.engine.SendMouse(termcore.MouseClickEvent{X: x, Y: y, Button: termcore.MouseLeft})
	k.term.engine.SendMouse(termcore.MouseReleaseEvent{X: x, Y: y, Button: termcore.MouseLeft})
}

// Dragged injects cell-motion events while the program tracks mouse motion.
func (k *keyCatcher) Dragged(e *fyne.DragEvent) {
	if !k.term.engine.MouseEnabled() {
		return
	}
	x, y := k.cellPos(&fyne.PointEvent{Position: e.Position})
	k.dragMu.Lock()
	last := k.lastDrag
	k.dragMu.Unlock()
	if last.X == x && last.Y == y {
		return // only report cell changes
	}
	k.dragMu.Lock()
	k.lastDrag = termcore.Mouse{X: x, Y: y}
	k.dragMu.Unlock()
	k.term.engine.SendMouse(termcore.MouseMotionEvent{X: x, Y: y, Button: termcore.MouseLeft})
}

// DragEnd injects the button release.
func (k *keyCatcher) DragEnd() {
	if !k.term.engine.MouseEnabled() {
		return
	}
	k.dragMu.Lock()
	last := k.lastDrag
	k.dragMu.Unlock()
	k.term.engine.SendMouse(termcore.MouseReleaseEvent{X: last.X, Y: last.Y, Button: termcore.MouseLeft})
}

// CreateRenderer gives the catcher an empty visual; it only exists for input.
func (k *keyCatcher) CreateRenderer() fyne.WidgetRenderer { return &catcherRenderer{} }

type catcherRenderer struct{}

func (c *catcherRenderer) Objects() []fyne.CanvasObject { return nil }
func (c *catcherRenderer) Layout(fyne.Size)             {}
func (c *catcherRenderer) MinSize() fyne.Size           { return fyne.NewSize(0, 0) }
func (c *catcherRenderer) Refresh()                     {}
func (c *catcherRenderer) Destroy()                     {}

var _ fyne.Focusable = (*keyCatcher)(nil)
var _ mobile.Keyboardable = (*keyCatcher)(nil)
