package tui

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"

	"lazyrun/internal/github"
)

// ConfirmKind identifies mutating workflows requiring acknowledgement.
type ConfirmKind uint8

const (
	confirmOff ConfirmKind = iota
	confirmCancelRun
	confirmRerunRun
	confirmRerunFailed
)

var braceStrip = regexp.MustCompile(`\{[^}]*\}`)

func pollInterval(runs []github.WorkflowRun) time.Duration {
	for _, r := range runs {
		if liveStatus(strings.ToLower(r.Status)) {
			return 3 * time.Second
		}
	}
	return 20 * time.Second
}

func liveStatus(s string) bool {
	switch strings.TrimSpace(s) {
	case "queued", "in_progress", "pending", "waiting", "requested":
		return true
	default:
		return false
	}
}

func anyLive(runs []github.WorkflowRun) bool {
	for _, r := range runs {
		if liveStatus(strings.ToLower(r.Status)) {
			return true
		}
	}
	return false
}

func runGlyph(run github.WorkflowRun) string {
	status := strings.ToLower(strings.TrimSpace(run.Status))
	switch status {
	case "queued", "pending", "waiting", "requested":
		return "⏳"
	case "in_progress":
		return "▶"
	case "completed":
		if run.Conclusion == nil || strings.TrimSpace(*run.Conclusion) == "" {
			return "✓"
		}
		switch strings.ToLower(strings.TrimSpace(*run.Conclusion)) {
		case "success":
			return "✓"
		case "failure":
			return "✗"
		case "cancelled":
			return "⊘"
		case "skipped":
			return "⏭"
		case "timed_out":
			return "⏱"
		default:
			return "●"
		}
	default:
		return "•"
	}
}

func rerunFailedEligible(run github.WorkflowRun) bool {
	if !strings.EqualFold(run.Status, "completed") || run.Conclusion == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(*run.Conclusion)) {
	case "failure", "timed_out":
		return true
	default:
		return false
	}
}

func pickYAMLWorkflow(flows []github.Workflow) (*github.Workflow, error) {
	for i := range flows {
		p := strings.ToLower(flows[i].Path)
		if strings.HasSuffix(p, ".yml") || strings.HasSuffix(p, ".yaml") {
			return &flows[i], nil
		}
	}
	return nil, errors.New("no active YAML workflows")
}

func defaultBranchIdx(branches []github.Branch) int {
	for i := range branches {
		if branches[i].IsDefault {
			return i
		}
	}
	return 0
}

func matchRunIdx(runs []github.WorkflowRun, runID int64, prevIdx int) int {
	idx := clamp(prevIdx, 0, len(runs)-1)
	if runID == 0 {
		return idx
	}
	found := false
	for i := range runs {
		if runs[i].ID == runID {
			idx = i
			found = true
			break
		}
	}
	if found {
		return idx
	}
	return clamp(prevIdx, 0, len(runs)-1)
}

func clamp(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	switch {
	case v < lo:
		return lo
	case v > hi:
		return hi
	default:
		return v
	}
}

func shortErr(ctx string, err error) string {
	msg := strings.TrimSpace(err.Error())
	if len(msg) > 220 {
		msg = msg[:220] + "…"
	}
	return fmt.Sprintf("%s: %s", ctx, msg)
}

func verbLabel(kind ConfirmKind) string {
	switch kind {
	case confirmCancelRun:
		return "cancel run"
	case confirmRerunRun:
		return "rerun"
	case confirmRerunFailed:
		return "rerun failed jobs"
	default:
		return "action"
	}
}

func stripLogNoise(txt string) string {
	return braceStrip.ReplaceAllString(txt, "")
}

func truncString(s string, w int) string {
	if w < 1 {
		w = 1
	}

	return ansi.Truncate(s, w, "…")
}

func formatSpan(started, ended *string) string {
	var sStart, eEnd time.Time
	var okStart, okEnd bool
	if started != nil {
		ts, err := time.Parse(time.RFC3339, *started)
		if err == nil {
			okStart = true
			sStart = ts
		}
	}
	if ended != nil {
		ts, err := time.Parse(time.RFC3339, *ended)
		if err == nil {
			okEnd = true
			eEnd = ts
		}
	}
	if okStart && okEnd {
		return eEnd.Sub(sStart).Round(time.Millisecond).String()
	}
	if okStart && !okEnd {
		return time.Since(sStart).Round(time.Millisecond).String() + "(running)"
	}
	return "--"
}
