package main

import (
	"strings"
	"testing"
	"time"

	"github.com/xyzzyapps/cha-yen/internal/termcore"
)

// TestShowcaseThroughEngine runs the real showcase model through the
// virtual-terminal engine (the same code path the Android APK uses) and
// asserts that multi-byte graphemes — box-drawing borders and arrows —
// survive the ANSI->emulator->snapshot round trip intact. This guards the
// unicode-rendering regression (byte-vs-rune truncation) end to end.
func TestShowcaseThroughEngine(t *testing.T) {
	e := termcore.New(newShowcase(), 90, 32, nil)
	if err := e.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer e.Stop()

	deadline := time.Now().Add(5 * time.Second)
	var joined string
	for time.Now().Before(deadline) {
		s := e.Snapshot()
		var b strings.Builder
		for _, c := range s.Cells {
			b.WriteString(c.Text)
		}
		joined = b.String()
		if strings.Contains(joined, "Cha-Yen widget showcase") {
			break
		}
		select {
		case <-e.Changes():
		case <-time.After(20 * time.Millisecond):
		}
	}

	// The title proves the program rendered.
	if !strings.Contains(joined, "Cha-Yen widget showcase") {
		t.Fatalf("title not found on screen; got: %q", firstRunes(joined, 200))
	}
	// Body uses a lipgloss rounded border: these are 3-byte UTF-8 runes.
	// A byte-truncation bug would turn them into garbage, never these runes.
	boxes := []rune{'╭', '─', '│', '╮', '╰', '╯'}
	found := 0
	for _, want := range boxes {
		if strings.ContainsRune(joined, want) {
			found++
		}
	}
	if found == 0 {
		t.Fatalf("no box-drawing border runes survived the round trip (unicode regression?)")
	}
	// The tab strip uses the middle dot separator.
	if !strings.ContainsRune(joined, '·') {
		t.Errorf("middle-dot tab separator U+00B7 missing from screen")
	}
	t.Logf("ok: %d/6 border glyph classes present, title + tab dots rendered", found)
}

func firstRunes(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		r = r[:n]
	}
	return string(r)
}
