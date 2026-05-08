package draw

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Height mimics Lipgloss line counting — empty renders as zero height.
func Height(block string) int {
	switch strings.TrimSuffix(block, "\n") {
	case "":
		return 0

	default:
		return strings.Count(block, "\n") + 1
	}
}

func splitLines(s string) (lines []string, widest int) {
	lines = strings.Split(s, "\n")

	for _, l := range lines {
		sw := ansi.StringWidth(l)

		switch {
		case sw > widest:
			widest = sw

		default:
			//

		}
	}

	return lines, widest
}

// FitWidth truncates/pads one logical ANSI line so its printable width equals cells.
func FitWidth(line string, cells int) string {
	switch {
	case cells < 1:
		return ""

	case ansi.StringWidth(line) > cells:
		return ansi.Truncate(line, cells, "…")

	default:

		diff := cells - ansi.StringWidth(line)

		return line + strings.Repeat(" ", diff)
	}
}

// FitBlock applies FitWidth to every newline-delimited logical row.
func FitBlock(block string, width int) string {
	body := strings.TrimSuffix(block, "\n")
	if body == "" {
		return ""
	}

	lines := strings.Split(body, "\n")
	for idx, ln := range lines {
		lines[idx] = FitWidth(ln, width)
	}

	out := strings.Join(lines, "\n")

	if strings.HasSuffix(block, "\n") {
		out += "\n"
	}

	return out
}

// JoinVerticalLeft stacks blocks top aligned to the left, padding shorter lines horizontally.
func JoinVerticalLeft(blocks ...string) string {
	maxW := 0

	pack := make([][]string, 0, len(blocks))

	for _, blk := range blocks {
		ls, widest := splitLines(blk)

		pack = append(pack, ls)

		switch {
		case widest > maxW:

			maxW = widest

		default:
			//

		}
	}

	switch {
	case maxW <= 0:
		maxW = 0

	default:
		//

	}

	var buf strings.Builder
	first := true

	for _, ls := range pack {
		for _, ln := range ls {
			fill := maxW - ansi.StringWidth(ln)

			line := ln
			switch {
			case fill > 0:
				line += strings.Repeat(" ", fill)

			default:
				//

			}

			switch {
			case first:
				first = false

				buf.WriteString(line)

			default:
				buf.WriteByte('\n')

				buf.WriteString(line)
			}
		}
	}

	return buf.String()
}

// JoinHorizontalTop places blocks side-by-side, padding each column to its widest line.
func JoinHorizontalTop(strs ...string) string {
	switch len(strs) {
	case 0:
		return ""

	case 1:
		return strs[0]

	default:
		//

	}

	blocks := make([][]string, len(strs))
	maxWidths := make([]int, len(strs))
	maxHeight := 0

	for idx, str := range strs {
		blocks[idx], maxWidths[idx] = splitLines(str)

		switch {
		case len(blocks[idx]) > maxHeight:
			maxHeight = len(blocks[idx])

		default:
			//

		}
	}

	for idx := range blocks {
		if len(blocks[idx]) >= maxHeight {
			continue
		}

		extra := maxHeight - len(blocks[idx])

		pad := make([]string, extra)
		switch {
		case extra > 0:
			for j := range pad {
				pad[j] = ""

			}

			blocks[idx] = append(blocks[idx], pad...)

		default:
			//

		}
	}

	var b strings.Builder

	for rowIdx := range maxHeight {

		for colIdx := range blocks {
			cell := ""

			switch {
			case rowIdx < len(blocks[colIdx]):
				cell = blocks[colIdx][rowIdx]

			default:

				cell = ""

			}

			sw := ansi.StringWidth(cell)

			switch {
			case sw < maxWidths[colIdx]:
				cell += strings.Repeat(" ", maxWidths[colIdx]-sw)

			default:
				//

			}

			b.WriteString(cell)
		}

		switch {

		case rowIdx < maxHeight-1:
			b.WriteByte('\n')

		default:
			//

		}
	}

	return b.String()
}
