package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"lazyrun/internal/clipboard"
)

func navigateDelta(msg tea.KeyMsg) (int, bool) {
	switch msg.Type {
	case tea.KeyDown:
		return 1, true
	case tea.KeyUp:
		return -1, true
	default:
		switch strings.ToLower(msg.String()) {
		case "j":
			return 1, true
		case "k":
			return -1, true
		default:
			return 0, false
		}
	}
}

func pagePulse(msg tea.KeyMsg) int {
	switch msg.Type {
	case tea.KeyPgDown:
		return +1
	case tea.KeyPgUp:
		return -1
	default:
		return 0
	}
}

func paneRotate(p Pane, delta int) Pane {
	idx := (((int(p) + delta) % 3) + 3) % 3
	return Pane(idx)
}

func moveBranches(m Model, delta int) (Model, tea.Cmd) {
	n := len(m.branches)
	if n == 0 {
		m = relayout(m)
		return m, nil
	}
	prev := m.branchIdx
	idx := clamp(prev+delta, 0, n-1)
	if idx == prev {
		m = relayout(m)
		return m, nil
	}
	m.branchIdx = idx
	m.runIdx = 0
	m.focusRunID = 0
	return m, m.syncRunsJobsCmd()
}

func pageBranches(m Model, dir int, step int) (Model, tea.Cmd) {
	n := len(m.branches)
	if n == 0 {
		m = relayout(m)
		return m, nil
	}
	prev := m.branchIdx
	delta := dir * step
	idx := clamp(prev+delta, 0, n-1)
	if idx == prev {
		m = relayout(m)
		return m, nil
	}
	m.branchIdx = idx
	m.runIdx = 0
	m.focusRunID = 0
	return m, m.syncRunsJobsCmd()
}

func moveRuns(m Model, delta int) (Model, tea.Cmd) {
	n := len(m.runs)
	if n == 0 {
		m = relayout(m)
		return m, nil
	}
	prev := m.runIdx
	idx := clamp(prev+delta, 0, n-1)
	if idx == prev {
		m = relayout(m)
		return m, nil
	}
	m.runIdx = idx
	m.focusRunID = m.runs[idx].ID
	return m, m.loadJobsCmd()
}

func pageRuns(m Model, dir int, step int) (Model, tea.Cmd) {
	n := len(m.runs)
	if n == 0 {
		m = relayout(m)
		return m, nil
	}
	prev := m.runIdx
	delta := dir * step
	idx := clamp(prev+delta, 0, n-1)
	if idx == prev {
		m = relayout(m)
		return m, nil
	}
	m.runIdx = idx
	m.focusRunID = m.runs[idx].ID
	return m, m.loadJobsCmd()
}

func handleKeys(m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	}

	if m.pending != confirmOff {
		ls := strings.ToLower(msg.String())
		if ls == "y" {
			op := m.pending
			m.pending = confirmOff
			return m, m.execConfirm(op)
		}
		if ls == "n" || msg.Type == tea.KeyEsc || strings.EqualFold(msg.String(), "esc") {
			m.pending = confirmOff
			return m, nil
		}
		return m, nil
	}

	if strings.EqualFold(strings.TrimSpace(msg.String()), "?") {
		m.helpOpen = !m.helpOpen
		return m, nil
	}

	if m.helpOpen {
		switch msg.Type {
		case tea.KeyEsc:
			m.helpOpen = false
			return m, nil
		default:
			if strings.EqualFold(msg.String(), "esc") {
				m.helpOpen = false
				return m, nil
			}
			return m, nil
		}
	}

	if strings.EqualFold(strings.TrimSpace(msg.String()), "q") {
		return m, tea.Quit
	}

	if msg.Type == tea.KeyEsc || strings.EqualFold(msg.String(), "esc") {
		if m.view == viewLogs {
			m.view = viewDash
			m.status = m.statusBase()
			m.viewport.SetContent("")
			m = relayout(m)
			return m, nil
		}
	}

	if m.view == viewLogs && !m.helpOpen {
		next, cmd := m.viewport.Update(msg)
		m.viewport = next
		return m, cmd
	}

	const step = 12

	switch msg.Type {
	case tea.KeyTab:
		m.focus = paneRotate(m.focus, +1)
		m = relayout(m)
		return m, nil
	case tea.KeyShiftTab:
		m.focus = paneRotate(m.focus, -1)
		m = relayout(m)
		return m, nil
	}

	switch msg.String() {
	case "1":
		m.focus = PaneBranches
	case "2":
		m.focus = PaneRuns
	case "3":
		m.focus = PaneDetails
	default:
		goto nav
	}
	m = relayout(m)
	return m, nil

nav:
	switch strings.ToLower(msg.String()) {
	case "w":
		if m.branchName() == "" {
			m.status = "cannot dispatch workflow without selecting a branch"
			return m, nil
		}
		m.status = "issuing workflow_dispatch…"
		return m, m.workflowDispatchCmd()
	case "r":
		m.status = m.statusBase() + " • manual refresh"
		return m, tea.Batch(m.syncRunsJobsCmd(), m.clock())
	}

	switch msg.String() {
	case "R":
		if _, ok := m.selectedRun(); !ok {
			m.status = "pick a workflow run first"
			return m, nil
		}
		m.pending = confirmRerunRun
		return m, nil
	case "c":
		if _, ok := m.selectedRun(); !ok {
			m.status = "pick a workflow run first"
			return m, nil
		}
		m.pending = confirmCancelRun
		return m, nil
	case "F", "f":
		run, ok := m.selectedRun()
		if !ok {
			m.status = "pick a workflow run first"
			return m, nil
		}
		if !rerunFailedEligible(run) {
			m.status = "rerun failed is only eligible on failures / timeouts"
			return m, nil
		}
		m.pending = confirmRerunFailed
		return m, nil

	case "y":
		run, ok := m.selectedRun()
		if !ok || strings.TrimSpace(run.HTMLURL) == "" {
			m.status = "no run URL to copy yet"
			return m, nil
		}
		if _, err := clipboard.Copy(run.HTMLURL, os.Stdout); err != nil {
			m.status = fmt.Sprintf("clipboard • %v", err)
			return m, nil
		}
		m.status = fmt.Sprintf("copied run URL (%s)", truncString(run.HTMLURL, 120))
		return m, nil

	case "Y":
		body := stripLogNoise(m.cachedLog)
		if strings.TrimSpace(body) == "" {
			m.status = "aggregate logs unavailable — fetch with Enter/l first"
			return m, nil
		}
		if _, err := clipboard.Copy(body, os.Stdout); err != nil {
			m.status = fmt.Sprintf("clipboard • %v", err)
			return m, nil
		}
		m.status = fmt.Sprintf("copied logs (~%d bytes)", len([]rune(body)))
		return m, nil

	default:
	}

	if msg.Type == tea.KeyEnter {
		return openLogsIfAllowed(m)
	}
	if strings.EqualFold(msg.String(), "l") {
		return openLogsIfAllowed(m)
	}

	if delta, ok := navigateDelta(msg); ok {
		switch m.focus {
		case PaneBranches:
			mm, cmd := moveBranches(m, delta)
			return mm, cmd
		case PaneRuns:
			mm, cmd := moveRuns(m, delta)
			return mm, cmd
		case PaneDetails:
			m = relayout(m)
			return m, nil
		}
	}

	if pulse := pagePulse(msg); pulse != 0 {
		switch m.focus {
		case PaneBranches:
			mm, cmd := pageBranches(m, pulse, step)
			return mm, cmd
		case PaneRuns:
			mm, cmd := pageRuns(m, pulse, step)
			return mm, cmd
		}
	}

	switch msg.String() {
	case "g":
		switch m.focus {
		case PaneBranches:
			mm, cmd := moveBranches(m, -m.branchIdx)
			return mm, cmd
		case PaneRuns:
			if len(m.runs) > 0 {
				m.runIdx = 0
				m.focusRunID = m.runs[0].ID
				return m, m.loadJobsCmd()
			}
			m = relayout(m)
			return m, nil
		}
	case "G":
		switch m.focus {
		case PaneBranches:
			target := len(m.branches) - 1 - m.branchIdx
			mm, cmd := moveBranches(m, target)
			return mm, cmd
		case PaneRuns:
			if len(m.runs) > 0 {
				m.runIdx = len(m.runs) - 1
				m.focusRunID = m.runs[m.runIdx].ID
				return m, m.loadJobsCmd()
			}
			m = relayout(m)
			return m, nil
		}
	default:
	}

	m = relayout(m)
	return m, nil
}

func openLogsIfAllowed(m Model) (Model, tea.Cmd) {
	if !(m.focus == PaneRuns || m.focus == PaneDetails) {
		return m, nil
	}
	if len(m.runs) == 0 {
		m.status = "no workflow runs for this slice"
		return m, nil
	}
	m.view = viewLogs
	m.viewport.SetContent("Collecting aggregated job logs...")
	m.status = "downloading plaintext logs..."
	m = relayout(m)
	return m, m.fetchLogsCmd()
}
