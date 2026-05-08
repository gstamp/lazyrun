package draw

import (
	"strings"
)

// PaneInteriorTrim is border (2) + horizontal padding (2) for dashboard / log panes.
const PaneInteriorTrim = 4

// PaneVerticalBorderTrim counts only the top and bottom border rows.
const PaneVerticalBorderTrim = 2

// PaneInteriorWidth returns the number of columns available for text inside a rounded pane.
func PaneInteriorWidth(outer int) int {
	switch {
	case outer <= PaneInteriorTrim:
		return 1

	default:

		return outer - PaneInteriorTrim
	}
}

// PaneBodySlots returns how many body rows can live under a one-line title inside a pane.
func PaneBodySlots(outerH int) int {
	const titleLines = 1

	avail := outerH - PaneVerticalBorderTrim - titleLines

	switch {
	case avail < 1:
		return 1

	default:

		return avail
	}
}

// ViewportInteriorHeight is the inner line budget for a bordered log viewport (no title row).
func ViewportInteriorHeight(outerH int) int {
	switch v := outerH - PaneVerticalBorderTrim; {
	case v < 1:
		return 1

	default:

		return v
	}
}

// RoundedBox draws a rounded single-line border around plainText lines, clipping/padding to outerW x outerH cells.
func RoundedBox(outerW, outerH int, border Style, plainText string) string {
	switch {
	case outerW < 4:
		return plainText

	case outerH < 3:
		return plainText

	default:
		//

	}

	innerCols := PaneInteriorWidth(outerW)
	innerRows := outerH - PaneVerticalBorderTrim

	var src []string

	switch {

	case strings.TrimSpace(plainText) == "":

		src = nil

	default:

		src = strings.Split(strings.TrimRight(plainText, "\n"), "\n")
	}

	body := make([]string, innerRows)

	for row := range innerRows {

		switch {

		case row < len(src):
			body[row] = FitWidth(src[row], innerCols)

		default:

			body[row] = strings.Repeat(" ", innerCols)
		}
	}

	var b strings.Builder

	top := "╭" + strings.Repeat("─", outerW-2) + "╮"

	b.WriteString(border.Sprint(top))
	b.WriteString(Reset)
	b.WriteByte('\n')

	edge := border.Sprint("│") + Reset

	for _, ln := range body {
		b.WriteString(edge)
		b.WriteByte(' ')
		b.WriteString(ln)
		b.WriteByte(' ')
		b.WriteString(edge)
		b.WriteByte('\n')
	}

	bot := "╰" + strings.Repeat("─", outerW-2) + "╯"

	b.WriteString(border.Sprint(bot))
	b.WriteString(Reset)

	return b.String()
}

// DoubleBox renders a double-line border with light padding inside the frame.
func DoubleBox(outerW, outerH int, border Style, inner string) string {
	switch {
	case outerW < 4:
		return inner

	case outerH < 3:
		return inner

	default:
		//
	}

	innerCols := outerW - PaneInteriorTrim
	if innerCols < 1 {
		innerCols = 1
	}

	innerRows := outerH - PaneVerticalBorderTrim

	var src []string

	switch {

	case strings.TrimSpace(inner) == "":

		src = nil

	default:

		src = strings.Split(strings.TrimRight(inner, "\n"), "\n")
	}

	rows := make([]string, innerRows)

	for idx := range innerRows {
		var line string

		switch {
		case idx < len(src):
			line = FitWidth(src[idx], innerCols)

		default:
			line = strings.Repeat(" ", innerCols)

		}

		rows[idx] = line

	}

	var b strings.Builder

	top := "╔" + strings.Repeat("═", outerW-2) + "╗"

	b.WriteString(border.Sprint(top))
	b.WriteString(Reset)
	b.WriteByte('\n')

	left := border.Sprint("║") + Reset
	right := border.Sprint("║") + Reset

	for _, row := range rows {
		b.WriteString(left)
		b.WriteByte(' ')
		b.WriteString(row)
		b.WriteByte(' ')
		b.WriteString(right)
		b.WriteByte('\n')

	}

	bottom := "╚" + strings.Repeat("═", outerW-2) + "╝"

	b.WriteString(border.Sprint(bottom))
	b.WriteString(Reset)

	return b.String()
}
