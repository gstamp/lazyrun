package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"

	"lazyrun/internal/tui/draw"
)

var (
	hdr     = draw.Style{Fg: "#ff79c6", Bold: true}
	dimTxt  = draw.Style{Fg: "#6272a4"}
	hlTxt   = draw.Style{Fg: "#f8f8f2"}
	metaTxt = draw.Style{Fg: "#bd93f9"}
	okTxt   = draw.Style{Fg: "#50fa7b"}
	liveTag = draw.Style{Fg: "#ff5555", Bold: true}

	modalBorder = draw.Style{Fg: "#ff5555"}
	helpBorder  = draw.Style{Fg: "#bd93f9"}

	plainBorder = draw.Style{Fg: "#585858"}
	focusBorder = draw.Style{Fg: "#8be9fd"}

	jobAccent = draw.Style{Fg: "#ffb86c", Bold: true}

	backdropFill = draw.Style{Fg: "#44475a"}
)

func paneChromeX() int {
	return draw.PaneInteriorTrim
}

func paneChromeY() int {
	return draw.PaneVerticalBorderTrim
}

func paneInnerWidth(outerW int) int {
	return draw.PaneInteriorWidth(outerW)
}

func paneBodyRows(outerH int) int {
	return draw.PaneBodySlots(outerH)
}

func logsBubbleSize(outerW, outerH int) (innerW, innerH int) {
	return draw.PaneInteriorWidth(outerW), draw.ViewportInteriorHeight(outerH)
}

func limitRenderedLineWidth(block string, maxCells int) string {
	if maxCells < 1 {
		maxCells = 1
	}

	parts := strings.Split(block, "\n")

	for idx, ln := range parts {

		switch {

		case ansi.StringWidth(ln) > maxCells:

			parts[idx] = ansi.Truncate(ln, maxCells, "…")

		default:
			//

		}
	}

	return strings.Join(parts, "\n")
}

func footerRibbon(m Model) string {
	box := footerBlock(m)

	box = limitRenderedLineWidth(box, max(1, m.width))

	return draw.FitBlock(box, max(1, m.width))
}

func mainContentHeight(m Model) int {

	fh := draw.Height(footerRibbon(m))

	switch {

	case fh < 1:

		fh = 1

	default:

		//

	}

	h := m.height - fh

	switch {

	case h < 1:

		return 1

	default:

		return h

	}

}

func paneInnerListRows(outerPaneHeight int) int {

	return paneBodyRows(outerPaneHeight)

}

func detailsLineBudget(innerListRows int) int {
	return clamp(innerListRows*4, 16, 800)
}

func composeView(m Model) string {
	switch {
	case m.width <= 0 || m.height <= 0:
		return ""

	case m.helpOpen:
		return renderHelp(m.width, m.height)

	default:

		screen := composeDash(m)

		switch m.view {

		case viewLogs:

			screen = composeLogs(m)

		default:
			//

		}

		switch m.pending {

		case confirmOff:
			//

		default:

			return backdropDialog(m.width, m.height, renderConfirmOverlay(m.pending, m.maybeRunTitle(), m.width))
		}

		return screen

	}
}

func backdropDialog(w, h int, dlg string) string {
	return draw.Place(w, h, dlg, '░', backdropFill)
}

func relayout(m Model) Model {
	switch {
	case m.width <= 0 || m.height <= 0:
		return m

	default:
		//

	}

	mainH := mainContentHeight(m)

	headLines := draw.Height(logsBanner(m))

	vpOuterW := max(1, m.width)

	vpOuterH := max(1, mainH-headLines)

	vpInnerW, vpInnerH := logsBubbleSize(vpOuterW, vpOuterH)

	m.viewport.Width = vpInnerW

	m.viewport.Height = vpInnerH

	return m
}

func logsBanner(m Model) string {

	titlePlain := truncatePlain("Aggregated plaintext logs • Esc exits", max(1, m.width))

	title := hlTxt.Sprint(titlePlain)

	stat := metaTxt.Sprint(truncatePlain(strings.TrimSpace(m.status), max(1, m.width)))

	head := draw.JoinVerticalLeft(title, stat)

	return limitRenderedLineWidth(head, max(1, m.width))
}

func columnWidths(total int) (brW, rsW, dsW int) {

	switch {

	case total <= 0:

		return 0, 0, 0

	default:
		//

	}

	if total < 24 {

		brW = total / 3

		rsW = total / 3

		dsW = total - brW - rsW

		return brW, rsW, dsW
	}

	brW = max(10, total*24/100)

	rsW = max(12, total*38/100)

	dsW = total - brW - rsW

	switch {

	case dsW >= 14:

	default:

		need := 14 - dsW

		switch {

		case rsW-need >= 12:

			rsW -= need

			dsW += need

		case brW-need >= 8:

			brW -= need

			dsW += need

		default:
			//

		}

	}

	return brW, rsW, dsW

}

func composeDash(m Model) string {

	foot := footerRibbon(m)

	mainH := max(1, m.height-draw.Height(foot))

	brW, rsW, dsW := columnWidths(m.width)

	rowBudget := paneInnerListRows(mainH)

	detailBudget := detailsLineBudget(rowBudget)

	brIW := paneInnerWidth(brW)

	rsIW := paneInnerWidth(rsW)

	dsIW := paneInnerWidth(dsW)

	brCol := renderPane(PaneBranches, m,
		hdr.Sprint("[1] Branches"),

		branchesBody(m, brIW, rowBudget),

		brW,

		mainH,
	)

	rsCol := renderPane(PaneRuns, m,
		hdr.Sprint("[2] Runs"),

		runsBody(m, rsIW, rowBudget),

		rsW,

		mainH,
	)

	dsCol := renderPane(PaneDetails, m,
		hdr.Sprint("[3] Details"),

		detailsBody(m, dsIW, detailBudget),

		dsW,

		mainH,
	)

	grid := draw.JoinHorizontalTop(brCol, rsCol, dsCol)

	return draw.JoinVerticalLeft(grid, foot)

}

func composeLogs(m Model) string {

	foot := footerRibbon(m)

	mainH := max(1, m.height-draw.Height(foot))

	head := logsBanner(m)

	headLines := draw.Height(head)

	vpOuterW := max(1, m.width)

	vpOuterH := max(1, mainH-headLines)

	vpInnerCols := paneInnerWidth(vpOuterW)

	vpRendered := limitRenderedLineWidth(m.viewport.View(), vpInnerCols)

	view := draw.RoundedBox(vpOuterW, vpOuterH, plainBorder, vpRendered)

	stack := draw.JoinVerticalLeft(head, view)

	return draw.JoinVerticalLeft(stack, foot)

}

func renderPane(which Pane, m Model, title, body string, w, h int) string {

	br := plainBorder

	switch {

	case which == m.focus:

		br = focusBorder

	default:
		//

	}

	switch {

	case w < 4:

		return strings.TrimRight(title+"\n"+body, "\n")

	case h < 3:

		return strings.TrimRight(title+"\n"+body, "\n")

	default:

		//

	}

	inner := paneInnerWidth(w)

	payload := draw.JoinVerticalLeft(title, strings.TrimRight(body, "\n"))

	payload = limitRenderedLineWidth(payload, inner)

	return draw.RoundedBox(w, h, br, payload)

}

func branchesBody(m Model, wrap int, maxRows int) string {

	switch {

	case len(m.branches) == 0:

		return dimTxt.Sprint("(waiting for branches…)")

	default:

		//

	}

	wrap = max(1, wrap)

	limit := clamp(maxRows, 8, 1024)

	builder := strings.Builder{}

	for idx, branch := range m.branches {

		marker := ""

		switch {

		case branch.IsDefault:

			marker = "○ "

		case branch.IsProtected:

			marker = "🔒 "

		default:

			marker = "  "
		}

		sel := ""

		switch {

		case idx == m.branchIdx && m.focus == PaneBranches:

			sel = "› "
		default:

			//

		}

		plain := truncatePlain(sel+marker+branch.Name, wrap)

		txt := hlTxt.Sprint(plain)

		switch {

		case idx == m.branchIdx:

			txt = okTxt.Sprint(plain)

		default:

			//

		}

		builder.WriteString(txt)

		builder.WriteByte('\n')

		if idx+1 >= limit && len(m.branches) > limit {
			builder.WriteString(dimTxt.Sprint(fmt.Sprintf("⋯ +%d more branches", len(m.branches)-(idx+1))))
			builder.WriteByte('\n')
			break
		}

	}

	return strings.TrimSuffix(builder.String(), "\n")
}

func runsBody(m Model, wrap int, maxRows int) string {

	switch {

	case len(m.runs) == 0:

		return dimTxt.Sprint("(GitHub Actions has no cached runs)")

	default:

		//

	}

	wrap = max(1, wrap)

	limit := clamp(maxRows, 8, 1024)

	builder := strings.Builder{}

	for idx, run := range m.runs {

		line := fmt.Sprintf("%s #%d %s — %s",

			runGlyph(run),

			run.RunNumber,

			strings.TrimSpace(run.Workflow),

			run.Display)

		sel := ""

		switch {

		case idx == m.runIdx && m.focus == PaneRuns:

			sel = "› "
		default:

			//

		}

		plain := truncatePlain(sel+line, wrap)

		row := hlTxt.Sprint(plain)

		switch {

		case idx == m.runIdx:

			row = okTxt.Sprint(plain)

		default:

			//

		}

		builder.WriteString(row)

		builder.WriteByte('\n')

		if idx+1 >= limit && len(m.runs) > limit {
			builder.WriteString(dimTxt.Sprint(fmt.Sprintf("⋯ +%d more runs", len(m.runs)-(idx+1))))
			builder.WriteByte('\n')
			break
		}

	}

	return strings.TrimSuffix(builder.String(), "\n")

}

func detailsBody(m Model, wrap, lineBudget int) string {

	run, picked := m.selectedRun()

	switch {

	case !picked:

		return dimTxt.Sprint("(select a highlighted run)")

	default:

		//

	}

	budget := clamp(lineBudget, 12, 2000)

	lines := make([]string, 0, budget)

	push := func(value string) {

		switch {

		case len(lines) >= budget:

			return

		default:

			//

		}

		value = strings.TrimRight(value, "\n")

		switch strings.TrimSpace(value) {

		case "":

			return

		default:

			//

		}

		lines = append(lines, value)

	}

	push(kvLine("Title", run.Display, wrap))

	push(kvLine(

		"Workflow",

		strings.TrimSpace(run.Workflow)+"@"+strings.TrimSpace(run.Branch),

		wrap,
	))

	push(kvLine("Glyph", runGlyph(run), wrap))

	push(kvLine("Status", strings.ToUpper(strings.TrimSpace(run.Status)), wrap))

	switch {

	case run.Conclusion == nil || strings.TrimSpace(*run.Conclusion) == "":

	default:

		push(kvLine("Conclusion", strings.TrimSpace(*run.Conclusion), wrap))
	}

	push(kvLine("Event", run.Event, wrap))

	push(kvLine("Actor", run.Actor, wrap))

	push(kvLine("Timeline", run.CreatedAt+" → "+run.UpdatedAt, wrap))

	push(kvLine("SHA", shortSHA(run.HeadSha), wrap))

	push(kvLine("URL", run.HTMLURL, wrap))

	divider := clamp(wrap, 4, 256)

	push(metaTxt.Sprint(strings.Repeat("─", divider)))

	switch {

	case len(m.jobs) == 0:

		push(dimTxt.Sprint("(no hydrated jobs listing yet — tap r refresh)"))

		return strings.Join(lines, "\n")

	default:
		//

	}

	push(jobAccent.Sprint(fmt.Sprintf("%d hydrated jobs captured", len(m.jobs))))

	const stepBudget = 10

outer:

	for ji, job := range m.jobs {

		if len(lines) >= budget {
			push(dimTxt.Sprint("⋯ job rows trimmed for viewport budget"))
			break outer
		}

		header := fmt.Sprintf("[%02d] %s — %s Δ %s",

			ji+1,

			strings.TrimSpace(job.Name),

			strings.TrimSpace(strings.ToUpper(job.Status)),

			formatSpan(job.StartedAt, job.CompletedAt))

		push(hlTxt.Sprint(truncatePlain(header, wrap)))

		switch {

		case len(job.Steps) == 0:

			push(dimTxt.Sprint("   └ (Steps empty in API snapshot)"))

			continue outer

		default:
			//

		}

		for idx := 0; idx < len(job.Steps) && idx < stepBudget; idx++ {

			switch {

			case len(lines) >= budget:

				push(dimTxt.Sprint("   ⋯ step rows trimmed"))

				break outer

			default:

				//

			}

			step := job.Steps[idx]

			txt := fmt.Sprintf("   └ %02d %s %-12s Δ %s",

				step.Number,

				step.Name,

				strings.ToLower(strings.TrimSpace(step.Status)),

				formatSpan(step.StartedAt, step.CompletedAt))

			push(dimTxt.Sprint(truncatePlain(txt, wrap)))

		}

		switch {

		case len(job.Steps) <= stepBudget:
		default:

			push(dimTxt.Sprint(fmt.Sprintf("   ⋯ suppressed %d step rows beyond cap",

				len(job.Steps)-stepBudget)))
		}

	}

	return strings.Join(lines, "\n")

}

func truncatePlain(s string, width int) string {

	switch {

	case width < 1:

		width = 1

	default:

		//

	}

	return ansi.Truncate(s, width, "…")

}

func kvLine(kind, plain string, innerW int) string {

	kind = strings.TrimSpace(kind)

	plain = strings.TrimSpace(plain)

	switch plain {

	case "":
		return ""

	default:
		//

	}

	innerW = max(1, innerW)

	switch kind {

	case "":
		return hlTxt.Sprint(ansi.Truncate(plain, innerW, "…"))

	default:
		//

	}

	labelSrc := kind + ":"

	switch {

	case ansi.StringWidth(labelSrc) >= innerW-2:

		return hlTxt.Sprint(ansi.Truncate(labelSrc+" "+plain, innerW, "…"))

	default:
		//

	}

	labelCap := min(22, max(4, innerW/3))

	label := ansi.Truncate(labelSrc, labelCap, "…")

	labelCells := ansi.StringWidth(label)

	valBudget := innerW - labelCells - 1

	switch {

	case valBudget < 4:

		return hlTxt.Sprint(ansi.Truncate(labelSrc+" "+plain, innerW, "…"))

	default:

		//

	}

	return draw.JoinHorizontalTop(

		metaTxt.Sprint(label),

		hlTxt.Sprint(ansi.Truncate(plain, valBudget, "…")),
	)

}

func footerBlock(m Model) string {

	wide := max(1, m.width)

	rowA := truncatePlain(m.status, wide)

	hint := truncatePlain(fmt.Sprintf("%s pane · 1/2/3 focus · Tab/⇧Tab · ?", englishFocusPane(m.focus)), wide)

	poll := truncatePlain("poll cadence • "+
		pollInterval(m.runs).Truncate(250*time.Millisecond).String(), wide)

	switch {

	case anyLive(m.runs):

		return draw.JoinVerticalLeft(
			metaTxt.Sprint(rowA),

			liveTag.Sprint(truncatePlain("LIVE • ≈3s cadence", wide)),

			metaTxt.Sprint(hint),
		)

	default:

		return draw.JoinVerticalLeft(

			metaTxt.Sprint(rowA),

			dimTxt.Sprint(poll),

			metaTxt.Sprint(hint),
		)

	}

}

func englishFocusPane(p Pane) string {

	switch p {

	case PaneBranches:

		return "Branches"

	case PaneRuns:

		return "Runs"

	default:

		return "Details"

	}

}

func shortSHA(sha string) string {

	sha = strings.TrimSpace(sha)

	switch sha {

	case "":

		return "—"

	default:

		//

	}

	switch {

	case len(sha) <= 7:

		return sha

	default:

		return sha[:7]

	}

}

func renderConfirmOverlay(kind ConfirmKind, runTitle string, screenW int) string {

	msg := ""

	switch kind {

	case confirmCancelRun:

		msg = "Cancel run?\nUses POST `/actions/runs/{id}/cancel`."

	case confirmRerunRun:

		msg = "Re-run highlighted workflow?\nUses POST `/actions/runs/{id}/rerun`."

	case confirmRerunFailed:

		msg = "Re-run failed jobs?\nUses POST `/actions/runs/{id}/rerun-failed-jobs`."

	default:

		msg = "Confirm guarded action."

	}

	switch strings.TrimSpace(runTitle) {

	case "":

	default:

		msg = fmt.Sprintf("%s\n\nRun snapshot:\n%s", msg,
			truncatePlain(runTitle, max(48, screenW-16)))
	}

	msg += "\n\n[y] confirm · [n]/Esc discard"

	styled := hlTxt.Sprint(msg)

	scan := strings.Split(styled, "\n")

	longest := 0

	for _, ln := range scan {

		width := ansi.StringWidth(ln)

		switch {

		case width > longest:

			longest = width

		default:
			//

		}

	}

	outerW := longest + paneChromeX() + 8
	maxOuter := screenW - 8
	if maxOuter < 40 {
		maxOuter = screenW - 4
	}
	if maxOuter < 40 {
		maxOuter = 40
	}
	outerW = min(maxOuter, max(48, outerW))

	dialog := draw.DoubleBox(outerW, len(scan)+4, modalBorder, styled)

	return dialog

}

func renderHelp(w, h int) string {

	cheat := strings.TrimSpace(strings.Join([]string{

		"lazyrun · Actions workflow dashboard",

		"",

		"Navigate: Tab/ShiftTab panes • 123 jump • j/k (↑↓) move • PgUp/PgDn page • g/G extremes",

		"",

		"Logs: Enter or l aggregates logs • viewport scroll • Esc leaves overlay",

		"",

		"Runs: r refresh slice • c / R / F confirm • W dispatches earliest YAML",

		"",

		"Clipboard: y copies URL • Y copies fetched logs `{}` sanitized",

		"",

		"? toggles cheatsheet • q / Ctrl+C quits Bubble Tea",
	}, "\n"))

	styled := hlTxt.Sprint(cheat)

	nodes := strings.Split(styled, "\n")

	longest := 0

	for _, ln := range nodes {

		width := ansi.StringWidth(ln)

		switch {

		case width > longest:

			longest = width

		default:
			//

		}

	}

	outerW := longest + paneChromeX() + 8
	maxOuter := w - 8
	if maxOuter < 48 {
		maxOuter = w - 4
	}
	outerW = min(maxOuter, max(56, outerW))

	outerH := min(h-6, len(nodes)+paneChromeY()+8)
	if outerH < len(nodes)+paneChromeY()+4 {
		outerH = len(nodes) + paneChromeY() + 4
	}

	panel := draw.DoubleBox(outerW, outerH, helpBorder, styled)

	return backdropDialog(w, h, panel)

}
