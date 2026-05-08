package tui

import "lazyrun/internal/github"

// Pane identifies which dashboard column absorbs list motion.
type Pane int

const (
	PaneBranches Pane = iota
	PaneRuns
	PaneDetails
)

const (
	viewDash int = iota
	viewLogs
)

type tickMsg struct{}

type branchesMsg struct {
	branches []github.Branch
	err      error
}

type syncRunsJobsMsg struct {
	runs    []github.WorkflowRun
	jobs    []github.Job
	runIdx  int
	runID   int64
	err     error
	jobsErr error
}

type jobsMsg struct {
	jobs []github.Job
	err  error
}

type logsFetchMsg struct {
	text string
	err  error
}

type wfDispatchMsg struct {
	name string
	err  error
}

type actionMsg struct {
	op  ConfirmKind
	err error
}
