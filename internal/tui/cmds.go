package tui

import (
	"errors"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"lazyrun/internal/github"
)

func (m Model) clock() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return tickMsg{} })
}

func (m Model) loadBranchesCmd() tea.Cmd {
	owner, slug := m.Owner, m.Repo
	return func() tea.Msg {
		branches, err := github.ListBranches(owner, slug)
		return branchesMsg{branches: branches, err: err}
	}
}

func (m Model) syncRunsJobsCmd() tea.Cmd {
	owner, slug := m.Owner, m.Repo
	ref := m.branchName()
	preIdx := m.runIdx
	preRun := m.focusRunID

	return func() tea.Msg {
		runs, err := github.ListRuns(owner, slug, ref, 40)
		if err != nil {
			return syncRunsJobsMsg{err: err}
		}
		idx := matchRunIdx(runs, preRun, preIdx)

		switch len(runs) {
		case 0:
			return syncRunsJobsMsg{runs: runs, jobs: nil, runIdx: 0}
		default:
			idx = clamp(idx, 0, len(runs)-1)
			run := runs[idx]
			jobList, jErr := github.ListJobs(owner, slug, run.ID)
			return syncRunsJobsMsg{
				runs:    runs,
				jobs:    jobList,
				runIdx:  idx,
				runID:   run.ID,
				jobsErr: jErr,
			}
		}
	}
}

func (m Model) loadJobsCmd() tea.Cmd {
	run, ok := m.selectedRun()
	if !ok {
		return func() tea.Msg { return jobsMsg{} }
	}
	owner, slug := m.Owner, m.Repo
	return func() tea.Msg {
		jobList, err := github.ListJobs(owner, slug, run.ID)
		return jobsMsg{jobs: jobList, err: err}
	}
}

func (m Model) fetchLogsCmd() tea.Cmd {
	run, ok := m.selectedRun()
	if !ok {
		return nil
	}
	owner, slug := m.Owner, m.Repo
	runID := run.ID
	return func() tea.Msg {
		text, err := github.CollectRunLogs(owner, slug, runID)
		return logsFetchMsg{text: text, err: err}
	}
}

func (m Model) workflowDispatchCmd() tea.Cmd {
	owner, slug := m.Owner, m.Repo
	ref := m.branchName()
	return func() tea.Msg {
		flows, err := github.ListWorkflows(owner, slug)
		if err != nil {
			return wfDispatchMsg{err: err}
		}
		wf, err := pickYAMLWorkflow(flows)
		if err != nil {
			return wfDispatchMsg{err: err}
		}
		if err := github.DispatchWorkflow(owner, slug, wf.ID, ref, nil); err != nil {
			return wfDispatchMsg{name: wf.Name, err: err}
		}
		return wfDispatchMsg{name: wf.Name}
	}
}

func (m Model) execConfirm(kind ConfirmKind) tea.Cmd {
	run, ok := m.selectedRun()
	if !ok {
		return func() tea.Msg {
			return actionMsg{op: kind, err: errors.New("no run selected")}
		}
	}
	owner, slug := m.Owner, m.Repo
	id := run.ID
	switch kind {
	case confirmCancelRun:
		return func() tea.Msg {
			err := github.CancelRun(owner, slug, id)
			return actionMsg{op: confirmCancelRun, err: err}
		}
	case confirmRerunRun:
		return func() tea.Msg {
			err := github.Rerun(owner, slug, id)
			return actionMsg{op: confirmRerunRun, err: err}
		}
	case confirmRerunFailed:
		return func() tea.Msg {
			err := github.RerunFailed(owner, slug, id)
			return actionMsg{op: confirmRerunFailed, err: err}
		}
	default:
		return func() tea.Msg {
			return actionMsg{op: kind, err: errors.New("unknown confirmation")}
		}
	}
}
