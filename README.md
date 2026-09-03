# Cha-Yen

> Bubble Tea, on Android. 🧋

Cha-Yen (ชาเย็น — Thai iced milk tea) runs [Bubble Tea](https://github.com/charmbracelet/bubbletea)
terminal UIs inside an Android APK. You keep writing normal Bubble Tea apps;
Cha-Yen renders them as a real terminal on your phone.

It does **not** use a PTY, a terminal emulator app, or an SSH bridge. Instead it
runs the Bubble Tea program *in-process* against a virtual terminal and paints
the resulting screen with a GPU-accelerated Fyne widget.

## How it works

```
   Bubble Tea Model (your app)
            |  charm.land/bubbletea/v2
            v
   +---------------------+   ANSI out    +------------------+
   |  tea.Program        |--------------->|  vt.Emulator     |
   |  WithInput/Output   |<---------------|  (charmbracelet/ |
   +---------------------+   key/mouse    |   x/vt virtual   |
                                         |   terminal)      |
                                         +------------------+
                                            |  CellAt()     |  key/mouse
                                            v  snapshot      ^ injection
                                   +--------------------------+
                                   |  Fyne TextGrid (terminal |
                                   |  view) + soft keyboard   |
                                   |  + extra-keys toolbar   |
                                   +--------------------------+
                                   Android APK (arm64) / desktop
```

- **`internal/termcore`** — GUI-free engine. Runs a `tea.Program` with
  `WithInput`/`WithOutput` pointed at a `charmbracelet/x/vt` virtual terminal
  (the emulator is *both* the program's stdin and stdout). Exposes screen
  snapshots, key/text/mouse injection, and resize. Fully unit-testable
  headless.
- **`internal/termui`** — Fyne layer. Maps the emulator's cell grid onto a
  `widget.TextGrid`, captures the Android soft keyboard + a toolbar of extra
  keys (Esc/Tab/Ctrl/arrows/PgUp/PgDn), and translates taps into SGR mouse
  events when the app enables mouse tracking.
- **`cmd/cha-yen`** — the app + a widget showcase exercising the major Bubble
  Tea / Bubbles components.

## Requirements

- Go 1.25+
- [Fyne CLI](https://docs.fyne.io/started/) (`go install fyne.io/tools/cmd/fyne@latest`)
- Android SDK + NDK + JDK 17 (for the APK)

## Run on desktop

```sh
go run ./cmd/cha-yen -demo showcase
```

Flags:

| Flag      | Default    | Purpose                                              |
|-----------|------------|------------------------------------------------------|
| `-demo`   | `showcase` | `showcase` \| `counter` \| `list` \| `clock`         |
| `-cols`   | `80`       | initial terminal columns                             |
| `-rows`   | `20`       | initial terminal rows                                |
| `-dump N` | `0`        | headless: render the grid to stdout after N ms, exit |
| `-logfile`| ``         | write structured logs to a file                      |

Set `BUB_DEBUG=1` for debug-level logs.

## Build the Android APK

```sh
pwsh scripts/build-android.ps1            # arm64 (physical devices)
pwsh scripts/build-android.ps1 -Arch amd64  # x86_64 (Android emulator)
```

Then install:

```sh
adb install -r cmd/cha-yen/Cha-Yen.apk
```

> **Note:** the x86_64 Android emulator runs arm64 Go binaries through ARM
> translation, which can crash in Go's runtime CPU detection
> (`internal/cpu.getMIDR`). Use `-Arch amd64` for the emulator and `arm64` for
> real phones.

## Using it with your own Bubble Tea app

1. Write a standard Bubble Tea v2 model (`Init`/`Update`/`View`).
2. In `cmd/cha-yen/main.go`, register it:
   ```go
   engine := termcore.New(myModel{}, 80, 20, logger)
   ```
3. Build the APK. Everything else (rendering, keyboard, mouse, resize) is
   handled by the engine.

## Tests

```sh
go test ./...
```

`internal/termcore` and the showcase/scroll suites run headless — they drive
real components through the engine and assert the rendered grid (including
box-drawing/UTF-8 correctness and scroll behaviour).

## Known limitations

- **Physical/Bluetooth keyboards** don't deliver keys on Android: Fyne's driver
  only forwards IME commit-text, not hardware key events. Use the soft keyboard
  or the on-screen toolbar.
- **Portrait** gives ~40 columns (phones are tall/narrow); rotate to **landscape**
  for a full 80-column terminal.
- Wide/CJK graphemes render one rune per cell (first rune of the cluster).
- Clipboard (Ctrl+V) via `atotto/clipboard` shells out to desktop tools and is
  a no-op on Android; bracketed paste still works from the soft keyboard.

## License

MIT — see [LICENSE](LICENSE). The vendored `bubbletea/` tree remains under its
original copyright (Charmbracelet, Inc.); see `bubbletea/LICENSE`.
