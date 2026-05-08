package tui

import (
	"strings"

	"lazyrun/internal/github"
)

func (m Model) branchName() string {
	if len(m.branches) == 0 {
		return ""
	}
	idx := clamp(m.branchIdx, 0, len(m.branches)-1)
	return m.branches[idx].Name
}

func (m Model) maybeRunTitle() string {
	run, ok := m.selectedRun()
	if !ok {
		return ""
	}
	return strings.TrimSpace(run.Display)
}

func (m Model) selectedRun() (github.WorkflowRun, bool) {
	if len(m.runs) == 0 || m.runIdx < 0 || m.runIdx >= len(m.runs) {
		return github.WorkflowRun{}, false
	}
	return m.runs[m.runIdx], true
}
