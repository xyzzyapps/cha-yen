// Command cha-yen hosts a Bubble Tea program inside a terminal-styled
// Fyne window. On desktop it runs natively; the same code is packaged as
// an Android APK (see scripts/build-android.ps1 and README.md).
package main

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"

	tea "charm.land/bubbletea/v2"

	"github.com/xyzzyapps/cha-yen/internal/termcore"
	"github.com/xyzzyapps/cha-yen/internal/termui"
)

func main() {
	demo := flag.String("demo", "showcase", "demo model: showcase | counter | list | clock")
	cols := flag.Int("cols", 80, "terminal columns (fixed grid)")
	rows := flag.Int("rows", 20, "terminal rows (fixed grid)")
	logFile := flag.String("logfile", "", "optional path for debug logs")
	dump := flag.Int("dump", 0, "headless: render the grid to stdout after N ms and exit")
	flag.Parse()

	logger := setupLogging(*logFile)
	slog.SetDefault(logger)

	var model tea.Model
	switch *demo {
	case "counter":
		model = counterModel{}
	case "list":
		model = newListModel()
	case "clock":
		model = clockModel{now: time.Now()}
	case "showcase":
		model = newShowcase()
	default:
		logger.Error("unknown demo", "name", *demo)
		os.Exit(2)
	}

	engine := termcore.New(model, *cols, *rows, logger.With("comp", "termcore"))
	if err := engine.Start(); err != nil {
		logger.Error("engine start failed", "err", err)
		os.Exit(1)
	}

	if *dump > 0 {
		time.Sleep(time.Duration(*dump) * time.Millisecond)
		s := engine.Snapshot()
		fmt.Printf("    +-%s+\n", strings.Repeat("-", s.Cols))
		for y := 0; y < s.Rows; y++ {
			var b strings.Builder
			for x := 0; x < s.Cols; x++ {
				t := s.At(x, y).Text
				if t == "" {
					t = " "
				}
				b.WriteString(t)
			}
			fmt.Printf(" %2d|%s|\n", y, b.String())
		}
		fmt.Printf("    +-%s+\n", strings.Repeat("-", s.Cols))
		engine.Stop()
		return
	}

	a := app.NewWithID("dev.bub.android")
	a.Settings().SetTheme(theme.DarkTheme())
	w := a.NewWindow("chayen")
	terminal := termui.NewTerminal(engine, logger.With("comp", "termui"))
	w.SetContent(terminal)
	w.Resize(fyne.NewSize(float32(*cols)*8+20, float32(*rows)*16+80))
	w.SetOnClosed(func() {
		terminal.Stop()
		engine.Stop()
	})

	// Close the window when the Bubble Tea program quits on its own.
	go func() {
		<-engine.Done()
		logger.Info("program exited; closing window")
		fyne.Do(func() { a.Quit() })
	}()

	w.Show()
	w.Canvas().Focus(terminal.Input())
	w.RequestFocus()
	logger.Info("ui running", "demo", *demo)
	a.Run()
	engine.Stop()
}

// setupLogging writes structured logs to stderr and optionally a file.
func setupLogging(path string) *slog.Logger {
	handlers := []io.Writer{os.Stderr}
	if path != "" {
		if f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
			handlers = append(handlers, f)
		}
	}
	lvl := slog.LevelInfo
	if os.Getenv("BUB_DEBUG") != "" {
		lvl = slog.LevelDebug
	}
	return slog.New(slog.NewTextHandler(io.MultiWriter(handlers...), &slog.HandlerOptions{Level: lvl}))
}
