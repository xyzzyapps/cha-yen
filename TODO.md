# Cha-Yen — Bubble Tea on Android (terminal-widget port)

Goal: run charmbracelet/bubbletea v2 TUIs inside an Android APK, rendered as a
terminal, using an in-process virtual terminal + Fyne TextGrid widget.
Module path: `github.com/xyzzyapps/cha-yen`.

## Verified facts (from audit, 2026-09-03)

- bubbletea v2 (`charm.land/bubbletea/v2`, pinned to tag v2.0.9, vendored in
  `./bubbletea` via a `replace` directive) runs fully on non-TTY
  `io.Reader`/`io.Writer` via `WithInput`/`WithOutput` (tea_test.go does exactly
  this). TTY paths are guarded by `term.File` + `IsTerminal` checks — pipes skip
  raw mode.
- `WithWindowSize`, `WithEnvironment`, `WithColorProfile`, `WithoutSignalHandler`
  exist; resize is driven with `p.Send(tea.WindowSizeMsg{...})`.
- `charm.land/bubbles/v2` (v2.2.1) is the v2-compatible component library
  (textinput, textarea, list, table, viewport, spinner, progress, timer,
  stopwatch, paginator, help, key, cursor, tree, filepicker).
- `github.com/charmbracelet/x/vt` is a full virtual terminal emulator using the
  SAME `uv.Cell` type as bubbletea's renderer: `Write()` consumes ANSI,
  `Read()` yields injected key/mouse escape sequences (`SendKey`, `SendText`,
  `SendMouse`), plus `Resize`, scrollback, `Touched()` damage tracking, and
  `Callbacks.EnableMode/DisableMode` (used to detect when the app enables mouse
  tracking). => It is BOTH the output writer and the input reader of the program.
- Android is GOOS=linux => all unix build tags in bubbletea/ultraviolet apply.
- Fyne v2.8.1 `widget.TextGrid` has per-cell styles
  (`CustomTextGridStyle`: text color, background color, bold, underline) —
  the terminal surface. No pre-existing Fyne terminal widget exists, so the
  adapter is built here.

## Architecture (vertical slices)

    +------------------------------------------------------------+
    | Android APK / desktop window (Fyne)                        |
    |  termui.Terminal                                           |
    |   |- widget.TextGrid  <--- cell diff <--- vt.Emulator      |
    |   |- keyboard toolbar (Esc/Ctrl/arrows/Tab) --- SendKey -> |
    |   |- soft keyboard (focusable catcher) ------ SendText ->  |
    |   |- tap/drag ---------- SendMouse (SGR) --------------->  |
    |                              vt.Emulator (x/vt)            |
    |                    output: ^ Write()   input: | Read()     |
    |                     tea.NewProgram(WithInput(uv), WithOutput(uv))
    |                              | goroutine                   |
    |                        bubbletea Model/Update/View         |
    +------------------------------------------------------------+

Modules:
- `internal/termcore`  — engine glue: emulator + program lifecycle, resize,
  change-notify pump, ANSI/style mapping to a GUI-free snapshot. Pure Go,
  unit-testable headless.
- `internal/termui`    — Fyne widget: TextGrid renderer, key/touch adapters,
  extra-keys toolbar, dark theme + font fit.
- `cmd/cha-yen`        — main app + a full widget showcase.

## Phases

### Phase 0 — Toolchain
- [x] 0.1 Install JDK 17 (Temurin via winget); builds set JAVA_HOME to it
- [x] 0.2 fyne CLI (`fyne.io/tools/cmd/fyne`)
- [x] 0.3 gomobile + `gomobile init` (ANDROID_HOME, NDK 30.0.14904198)

### Phase 1 — Core engine, desktop-first
- [x] 1.1 module + deps (bubbletea v2 replace -> ./bubbletea, bubbles v2, x/vt,
      lipgloss v2, fyne v2)
- [x] 1.2 termcore: tea.Program wired to a vt.Emulator (input+output);
      WithWindowSize + WithEnvironment(TERM=xterm-256color) + TrueColor profile;
      WithoutSignalHandler for the APK
- [x] 1.3 Unit tests: render a model, inject keys/text, assert screen + quit
- [x] 1.4 termui: uv.Cell -> TextGrid style mapping (fg/bg incl. indexed +
      truecolor, bold/underline/reverse/conceal); row-diffed ~30fps pump
- [x] 1.5 Desktop window; resize recomputes cols/rows from glyph metrics

### Phase 2 — Touch input
- [x] 2.1 Focusable key catcher: TypedRune -> SendText, TypedKey -> SendKey
- [x] 2.2 Extra-keys toolbar (Esc, Ctrl lock, Tab, PgUp/PgDn, arrows) at TOP so
      the soft keyboard never covers it
- [x] 2.3 Tap/drag -> SGR mouse events via SendMouse, gated on the program
      enabling mouse tracking (detected via vt mode callbacks)

### Phase 3 — Android build
- [x] 3.1 `scripts/build-android.ps1` -> debug APK via `fyne package`
      (arm64 for devices, amd64 for the emulator)
- [x] 3.2 Install + launch via adb; screenshot/logcat verification
- [x] 3.3 Platform fixes: dark theme (white-surround fix), UTF-8 rune decode
      (gibberish fix), input-injection deadlock fix, geometry from pixel size

### Phase 4 — Docs, polish, branding
- [x] 4.1 Full widget showcase (textinput, textarea, list, table, viewport,
      spinner, progress, timer, stopwatch, cursor, paginator, tree, filepicker,
      hand-rolled checkbox/radio/select, lipgloss styles, mouse regions)
- [x] 4.2 Headless scroll/container/UTF-8 test suite
- [x] 4.3 Rename bubandroid -> Cha-Yen; module path github.com/xyzzyapps/cha-yen
- [x] 4.4 LICENSE (MIT, same as bubbletea), README.md, SPEC.md
- [ ] 4.5 git init + first commit (this session)

## Known limitations / follow-ups
- Physical/Bluetooth keyboards: Fyne's Android driver forwards only IME
  commit-text, not hardware key events. Optional Phase 5: a Java onKeyDown shim
  in GoNativeActivity to inject real keys.
- Portrait yields ~40 columns; landscape yields ~80. Consider forcing landscape.
- Wide/CJK: first rune of a grapheme per cell; tail cells blank (not full width).
- Clipboard (atotto/clipboard) is a desktop shell-out no-op on Android; bracketed
  paste from the soft keyboard works.
- x/vt is pre-1.0 (pinned pseudo-version); fallback would be hinshun/vt10x.
