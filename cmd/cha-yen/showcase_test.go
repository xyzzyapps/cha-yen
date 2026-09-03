package main

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestShowcaseEveryPaneRenders drives the showcase through all tabs and
// asserts each pane renders non-empty content without panicking. This is a
// headless smoke test for widget compatibility with the virtual-terminal
// engine (no GUI required).
func TestShowcaseEveryPaneRenders(t *testing.T) {
	m := newShowcase()
	// Size the model as the engine would.
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 32})
	m = updated.(showcaseModel)

	for i := range m.panes {
		um, _ := m.gotoTab(i)
		sm := um.(showcaseModel)
		v := sm.View()
		content := v.Content
		if strings.TrimSpace(content) == "" {
			t.Errorf("pane %d (%s) rendered empty content", i, m.panes[i].name())
		}
		if !strings.Contains(content, m.panes[i].name()) {
			t.Errorf("pane %d (%s): tab label missing from view", i, m.panes[i].name())
		}
	}
}

// TestShowcasePaneNavigation verifies tab/number navigation moves the cursor
// and that quit works.
func TestShowcasePaneNavigation(t *testing.T) {
	m := newShowcase()
	um, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 32})
	m = um.(showcaseModel)

	um, _ = m.Update(tea.KeyPressMsg{Code: '2'})
	m = um.(showcaseModel)
	if m.tab != 1 {
		t.Fatalf("after pressing '2' tab=%d, want 1 (List)", m.tab)
	}

	um, _ = m.Update(tea.KeyPressMsg{Code: 'q'})
	m = um.(showcaseModel)
	if !m.quitting {
		t.Fatal("pressing 'q' did not trigger quit")
	}
}
