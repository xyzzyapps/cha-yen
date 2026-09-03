package main

import (
	"fmt"
	"time"

	"charm.land/bubbles/v2/cursor"
	"charm.land/bubbles/v2/paginator"
	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/stopwatch"
	"charm.land/bubbles/v2/timer"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// progTickMsg drives the progress bar forward independently of its animation.
type progTickMsg time.Time

// widgetsPane demonstrates the async/animated components.
type widgetsPane struct {
	sp    spinner.Model
	prog  progress.Model
	timer timer.Model
	sw    stopwatch.Model
	cur   cursor.Model
	pg    paginator.Model
}

func newWidgetsPane() pane {
	sp := spinner.New(spinner.WithSpinner(spinner.Dot))
	prog := progress.New(progress.WithWidth(30))
	tm := timer.New(5 * time.Second)
	sw := stopwatch.New(stopwatch.WithInterval(time.Second))
	cur := cursor.New()
	cur.SetChar("█")
	pg := paginator.New()
	pg.PerPage = 5
	pg.SetTotalPages(23)
	return &widgetsPane{sp: sp, prog: prog, timer: tm, sw: sw, cur: cur, pg: pg}
}

func (p *widgetsPane) name() string   { return "Widgets" }
func (p *widgetsPane) Blur()          {}
func (p *widgetsPane) Focus() tea.Cmd { return p.cur.Focus() }

func (p *widgetsPane) Init() tea.Cmd {
	return tea.Batch(
		p.timer.Init(), p.sw.Init(),
		progTick(),
	)
}

func progTick() tea.Cmd {
	return tea.Tick(400*time.Millisecond, func(t time.Time) tea.Msg { return progTickMsg(t) })
}

// Update forwards every message to every component; each self-sustaining
// ticker (spinner, timer, stopwatch) reschedules itself via the returned Cmd.
func (p *widgetsPane) Update(msg tea.Msg) (pane, tea.Cmd) {
	var cmds []tea.Cmd
	var c tea.Cmd

	switch msg := msg.(type) {
	case progTickMsg:
		cmds = append(cmds, p.prog.IncrPercent(0.05))
		if p.prog.Percent() >= 1 {
			p.prog.SetPercent(0)
		}
		cmds = append(cmds, progTick())
	case tea.KeyPressMsg:
		switch msg.String() {
		case "left", "h":
			p.pg.PrevPage()
		case "right", "l":
			p.pg.NextPage()
		}
	default:
	}

	p.sp, c = p.sp.Update(msg)
	cmds = append(cmds, c)
	p.prog, c = p.prog.Update(msg)
	cmds = append(cmds, c)
	p.timer, c = p.timer.Update(msg)
	cmds = append(cmds, c)
	p.sw, c = p.sw.Update(msg)
	cmds = append(cmds, c)
	p.cur, c = p.cur.Update(msg)
	cmds = append(cmds, c)

	return p, tea.Batch(cmds...)
}

var wLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Bold(true)

func (p *widgetsPane) View(w, h int) string {
	page := fmt.Sprintf("page %d/%d", p.pg.Page+1, p.pg.TotalPages)
	return wLabel.Render("spinner") + "    " + p.sp.View() + " loading\n\n" +
		wLabel.Render("progress") + "   " + p.prog.View() + "\n\n" +
		wLabel.Render("timer") + "      " + p.timer.View() + "\n\n" +
		wLabel.Render("stopwatch") + "  " + p.sw.View() + "\n\n" +
		wLabel.Render("cursor") + "     " + p.cur.View() + " (blinking block)\n\n" +
		wLabel.Render("paginator") + "  " + p.pg.View() + "  " + page
}

func (p *widgetsPane) mouseMode() tea.MouseMode { return tea.MouseModeNone }
