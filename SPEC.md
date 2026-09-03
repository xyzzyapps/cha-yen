# Cha-Yen — Technical Specification

SRS-style specification for **Cha-Yen**, an Android APK (and desktop app) that
runs [Bubble Tea](https://github.com/charmbracelet/bubbletea) v2 terminal UIs
inside a virtual terminal rendered by a Fyne `TextGrid`.

Module path: `github.com/xyzzyapps/cha-yen`.

---

## 1. Purpose

A Bubble Tea program normally needs a real TTY: a `*os.File` stdin/stdout, a
terminal that interprets ANSI/SGR escape sequences, and a keyboard. An Android
APK has none of that — no TTY, no terminal, no ANSI host.

Cha-Yen keeps the Bubble Tea runtime **unchanged** and replaces its
environment: the program's input/output are wired to an **in-process virtual
terminal emulator**, and that emulator's screen is painted to the device by a
GPU-accelerated UI toolkit.

Non-goals: running arbitrary shell command lines, being a general terminal
emulator, or supporting a physical Bluetooth keyboard (see §8).

---

## 2. Architecture

```
        +-------------------------------------------+
        |  your Bubble Tea program (Model/Update/View)|
        +-------------------------------------------+
                 ^ io.Reader      | io.Writer
                 | (key/mouse)    v (ANSI/SGR bytes)
        +-------------------------------------------+
        |  tea.Program  (charm.land/bubbletea/v2)    |   runs in a goroutine
        |  WithInput(emu) WithOutput(emu)            |
        +-------------------------------------------+
                 ^ Read()         | Write()
                 |               v
        +-------------------------------------------+
        |  vt.SafeEmulator (github.com/charmbracelet|   a real VT100/xterm
        |   /x/vt): cell grid, scroll regions,       |   virtual terminal
        |   SGR mouse, mode callbacks                |
        +-------------------------------------------+
                 ^ CellAt/snapshot | SendKey/SendText/SendMouse
                 |                v
        +-------------------------------------------+
        |  termui.Terminal (Fyne)                    |
        |   - widget.TextGrid (per-cell fg/bg/bold)  |
        |   - keyCatcher (soft keyboard + focus)     |
        |   - extra-keys toolbar (Esc/Ctrl/arrows)   |
        |   - tap/drag -> SGR mouse                  |
        +-------------------------------------------+
                 |
                 v
        Android APK (arm64) / desktop window
```

Three layers, each independently testable:

| Layer | Package | Responsibility |
|-------|---------|----------------|
| Engine | `internal/termcore` | Run a `tea.Program` against a virtual terminal; expose snapshots + input injection + resize. **GUI-free.** |
| UI | `internal/termui` | Render the engine's cell grid with Fyne; capture keyboard/touch; translate to engine input. |
| App | `cmd/cha-yen` | Wire engine + UI into a window; demo models + widget showcase. |

---

## 3. Data model

### 3.1 Virtual terminal

`github.com/charmbracelet/x/vt` is used as the terminal. It is the *same* cell
type (`ultraviolet.Cell`) that Bubble Tea's v2 renderer produces, so the
renderer's ANSI stream parses cleanly back into cells. The emulator is wired as
**both** the program's `io.Reader` (it yields injected key/mouse escape
sequences) and `io.Writer` (it consumes the renderer's ANSI output). This is the
central trick that makes Bubble Tea believe it is on a real terminal.

### 3.2 `termcore.Cell` / `termcore.Screen`

A GUI-free snapshot of the screen, decoupled from ultraviolet internals:

```go
type Cell struct {
    Text          string      // grapheme cluster ("" = blank)
    Fg, Bg        color.Color // resolved, never nil
    Bold, Italic, Underline, Strikethrough bool
    Wide, WideTail bool        // double-width cell + its placeholder
}
type Screen struct {
    Cols, Rows int
    Cells      []Cell         // row-major
    CursorX, CursorY int
    AltScreen  bool
}
```

`Snapshot()` walks the emulator grid, resolves default fg/bg, applies reverse
video and conceal, and marks wide-cell tails. Colors are resolved against the
emulator's default palette so the UI never deals with `nil`.

### 3.3 `termcore.Engine` interface

```go
type Engine interface {
    Start() error
    Stop()
    Done() <-chan struct{}
    Err() error
    Resize(cols, rows int)
    Size() (cols, rows int)
    SendKey(k KeyPressEvent)
    SendText(text string)
    Paste(text string)
    SendMouse(m MouseEvent)
    MouseEnabled() bool
    Snapshot() *Screen
    Changes() <-chan struct{}   // coalesced "output was written" signal
}
```

Re-exports `KeyPressEvent`, `Mouse*Event`, key/mod/mouse constants so the UI
does not import ultraviolet directly.

---

## 4. Engine lifecycle

1. `New(model, cols, rows, log)` builds a `vt.SafeEmulator`, sets a dark default
   background/foreground, and registers `EnableMode`/`DisableMode` callbacks
   (tracked in an atomic bitmask → `MouseEnabled()`).
2. `Start()` creates a `tea.Program` with:
   - `WithInput(emu)` / `WithOutput(signalWriter{emu})`
   - `WithWindowSize(cols, rows)`
   - `WithEnvironment(["TERM=xterm-256color"])`
   - `WithColorProfile(TrueColor)`
   - `WithoutSignalHandler()` (no SIGWINCH/SIGTSTP inside an APK)

   It runs `prog.Run()` in a goroutine; the `signalWriter` forwards bytes to the
   emulator and signals `Changes()` (non-blocking, coalesced).
3. `Resize` resizes the emulator and sends `tea.WindowSizeMsg`.
4. `Stop`/program exit closes `Done()` and unblocks the input reader.

**Concurrency note:** input injection (`SendKey`/`SendText`/`SendMouse`) uses the
raw embedded `Emulator` under a dedicated mutex, *not* the `SafeEmulator`
wrappers. The wrappers hold the screen mutex across a **blocking** `io.Pipe`
write, which deadlocks against the render path; the input pipe is independent of
the screen buffer, so it is serialized separately.

---

## 5. UI layer (`termui`)

- **Render pump**: a goroutine waits on `Changes()` with a ~30fps ticker floor;
  each tick snapshots the screen, diffs rows vs the last snapshot, and pushes
  only changed rows into `widget.TextGrid` via `fyne.Do`.
- **Style mapping**: `uv.Cell` → `widget.CustomTextGridStyle` (fg/bg incl.
  indexed + truecolor, bold/italic/underline, reverse/conceal resolved in the
  engine). Styles are cached by RGBA key to avoid per-frame allocation.
- **UTF-8 correctness**: cell text is a grapheme cluster; the first **rune** is
  decoded with `utf8.DecodeRuneInString` (not the first *byte*), so box-drawing
  borders, arrows, and other multibyte glyphs render correctly.
- **Geometry**: cols/rows are derived from the actual window pixel size ÷ the
  rendered monospace cell metrics, so the terminal fills the screen edge-to-edge.
- **Input**: a focusable `keyCatcher` (invisible widget) owns focus, raises the
  Android soft keyboard (`mobile.Keyboardable`), and maps `TypedRune`/`TypedKey`
  to engine input. A `Ctrl`-arm toggle in the toolbar lets the soft keyboard
  produce `ctrl+<letter>`.
- **Toolbar**: extra keys (Esc, Tab, Ctrl, PgUp, PgDn, ↑↓←→) live at the **top**
  so the soft keyboard (which overlays the bottom) never hides them.
- **Mouse**: taps/drags become SGR `MouseClick`/`Motion`/`Release` events, but
  only when `MouseEnabled()` (i.e. the running program turned on mouse
  tracking).

---

## 6. Build & run

- Desktop: `go run ./cmd/cha-yen -demo showcase` (flags: `-demo`, `-cols`,
  `-rows`, `-dump N` headless grid dump, `-logfile`; `BUB_DEBUG=1` verbose).
- Android: `scripts/build-android.ps1 [-Arch arm64|amd64]` →
  `cmd/cha-yen/Cha-Yen.apk`. Requires Fyne CLI, Android SDK+NDK, JDK 17.
  - `arm64` for physical devices.
  - `amd64` for the x86_64 Android emulator (arm64 Go binaries crash in Go's
    runtime MIDR CPU detection under ARM-to-x86 translation).

---

## 7. Testing

Headless (no GUI required), driven through the real engine:

- `internal/termcore`: render, key/text input, quit, resize-redraw.
- `cmd/cha-yen`: every showcase pane renders; tab navigation; a
  UTF-8/box-drawing round-trip assertion; viewport/list/textarea **scroll**
  (keyboard + mouse wheel + page-down); terminal scroll-region behaviour when
  content exceeds the screen; nested lipgloss containers.

---

## 8. Known limitations

1. **Physical/Bluetooth keyboards**: Fyne's Android driver forwards only IME
   commit-text, not hardware key events. Soft keyboard + toolbar only. (Optional
   future: a Java `onKeyDown` shim in the activity template.)
2. **Portrait ≈ 40 columns** (phones are tall/narrow); landscape ≈ 80.
3. **Wide/CJK**: one rune per cell (first of the grapheme); tail cells blank.
4. **Clipboard**: `atotto/clipboard` shells out to desktop tools → no-op on
   Android; bracketed paste from the soft keyboard still works.
5. **x/vt** is pre-1.0 (pinned pseudo-version).

---

## 9. Dependencies

| Module | Role |
|--------|------|
| `charm.land/bubbletea/v2` (vendored `./bubbletea`, tag v2.0.9) | TUI framework |
| `charm.land/bubbles/v2` | UI components (showcase) |
| `charm.land/lipgloss/v2` | styling |
| `github.com/charmbracelet/x/vt` | virtual terminal emulator |
| `github.com/charmbracelet/ultraviolet` | cell/ANSI types (shared with bubbletea) |
| `fyne.io/fyne/v2` | cross-platform GPU UI (TextGrid, input, Android packaging) |

## 10. License

MIT (see `LICENSE`). The vendored `bubbletea/` tree retains Charmbracelet's
original copyright (`bubbletea/LICENSE`).
