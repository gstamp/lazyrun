package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"lazyrun/internal/github"
)

// Model renders the Bubble Tea UI for Lazyrun.
type Model struct {
	Owner string
	Repo  string

	width  int
	height int

	branches []github.Branch
	runs     []github.WorkflowRun
	jobs     []github.Job

	branchIdx int
	runIdx    int

	focus   Pane
	view    int
	pending ConfirmKind

	syncing  bool
	helpOpen bool

	status   string
	lastPoll time.Time

	viewport viewport.Model

	logBusy  bool
	logsText string
	logErr   string
	logRunID int64

	cachedLog  string
	focusRunID int64
}

// New instantiates an Actions dashboard Model.
func New(owner, slug string) Model {
	vp := viewport.New(0, 0)
	vp.MouseWheelEnabled = false
	return Model{
		Owner:    owner,
		Repo:     slug,
		lastPoll: time.Now(),

		status: "loading branches via gh…",

		viewport: vp,
	}
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.loadBranchesCmd(), m.clock())
}

// Update satisfies tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m = relayout(m)
		return m, nil

	case branchesMsg:
		if msg.err != nil {
			m.status = shortErr("branch list", msg.err)
			m.branches = nil
			m.syncing = false
			return m, nil
		}
		m.branches = msg.branches
		if len(msg.branches) == 0 {
			m.branchIdx = 0
			m.status = fmt.Sprintf("%s/%s • no branches", m.Owner, m.Repo)
			return m, m.clock()
		}
		m.branchIdx = defaultBranchIdx(msg.branches)
		m.status = strings.TrimSpace(fmt.Sprintf("%s/%s • branches: %d", m.Owner, m.Repo, len(m.branches)))
		return m, tea.Batch(m.syncRunsJobsCmd(), m.clock())

	case syncRunsJobsMsg:
		m.syncing = false
		if msg.err != nil {
			m.status = shortErr("refresh runs", msg.err)
			m.runs = nil
			m.jobs = nil
			return m, m.clock()
		}
		m.runs = msg.runs
		m.runIdx = msg.runIdx
		m.focusRunID = msg.runID
		m.lastPoll = time.Now()

		if msg.jobsErr != nil {
			m.jobs = nil
			m.status = shortErr("jobs", msg.jobsErr)
		} else {
			m.jobs = msg.jobs
			dur := pollInterval(m.runs)
			tag := ""
			if anyLive(msg.runs) {
				tag = " • LIVE (3s poll)"
			} else {
				tag = fmt.Sprintf(" • polling every %v", dur.Round(time.Millisecond))
			}
			m.status = strings.TrimSpace(m.statusBase() + tag)
		}

		return m, m.clock()

	case jobsMsg:
		if msg.err != nil {
			m.jobs = nil
			m.status = shortErr("jobs", msg.err)
			return m, nil
		}
		m.jobs = msg.jobs
		return m, nil

	case logsFetchMsg:
		m.logBusy = false
		m = relayout(m)
		if msg.err != nil {
			m.view = viewDash
			m.logErr = msg.err.Error()
			m.status = shortErr("logs download", msg.err)
			m.viewport.SetContent("")
			return m, nil
		}
		m.logErr = ""
		m.logsText = msg.text
		m.cachedLog = msg.text
		text := strings.TrimSuffix(msg.text, "\n")
		m.viewport.SetContent(text)
		m.viewport.GotoTop()
		m.status = "logs — Esc exits"
		return m, nil

	case wfDispatchMsg:
		m.syncing = false
		if msg.err != nil {
			m.status = shortErr("workflow dispatch", msg.err)
			return m, m.clock()
		}
		m.status = fmt.Sprintf("workflow %q dispatched on %s", msg.name, m.branchName())
		return m, tea.Batch(m.syncRunsJobsCmd(), m.clock())

	case actionMsg:
		m.pending = confirmOff
		if msg.err != nil {
			m.status = fmt.Sprintf("%s • %v", verbLabel(msg.op), msg.err)
		} else {
			m.status = fmt.Sprintf("%s requested • refreshing", verbLabel(msg.op))
		}
		return m, tea.Batch(m.syncRunsJobsCmd(), m.clock())

	case tickMsg:
		if m.width == 0 || m.helpOpen || m.pending != confirmOff || m.syncing || m.view == viewLogs {
			return m, m.clock()
		}
		if len(m.branchName()) == 0 {
			return m, m.clock()
		}
		if time.Since(m.lastPoll) < pollInterval(m.runs) {
			return m, m.clock()
		}
		m.syncing = true
		m.status = m.statusBase() + " • refreshing…"
		return m, tea.Batch(m.syncRunsJobsCmd(), m.clock())

	case tea.KeyMsg:
		next, cmd := handleKeys(m, msg)
		return next, cmd
	}

	return m, nil
}

// View satisfies tea.Model.
func (m Model) View() string {
	return composeView(m)
}

func (m Model) statusBase() string {
	branch := m.branchName()
	if branch != "" {
		return fmt.Sprintf("%s/%s @ %s", m.Owner, m.Repo, branch)
	}
	return fmt.Sprintf("%s/%s", m.Owner, m.Repo)
}
