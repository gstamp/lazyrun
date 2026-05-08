package draw

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Place centers content inside a WxH canvas filled with fill (colored if fillStyle set).
func Place(width, height int, content string, fill rune, fillStyle Style) string {
	switch {
	case width <= 0 || height <= 0:
		return ""

	default:
		//

	}

	ch := string(fill)

	fillRun := func(n int) string {
		switch {
		case n <= 0:
			return ""

		default:
			seg := strings.Repeat(ch, n)

			switch {
			case fillStyle.Fg != "" || fillStyle.Bg != "" || fillStyle.Bold:
				return fillStyle.Sprint(seg)

			default:

				return seg
			}
		}
	}

	fillLine := func() string {
		return fillRun(width)
	}

	body := strings.TrimSuffix(content, "\n")

	var lines []string

	switch body {
	case "":
		lines = []string{""}

	default:
		lines = strings.Split(body, "\n")
	}

	chLen := len(lines)

	top := (height - chLen) / 2
	bottom := height - chLen - top

	switch {
	case top < 0:
		top = 0

	default:
		//

	}

	switch {
	case bottom < 0:
		bottom = 0

	default:
		//

	}

	var rows []string

	for idx := 0; idx < top; idx++ {
		rows = append(rows, fillLine())
	}

	for _, ln := range lines {
		sw := ansi.StringWidth(ln)

		switch {
		case sw > width:
			ln = ansi.Truncate(ln, width, "…")

			sw = ansi.StringWidth(ln)

		default:
			//

		}

		left := (width - sw) / 2
		right := width - sw - left

		rows = append(rows, fillRun(left)+ln+fillRun(right))
	}

	for idx := 0; idx < bottom; idx++ {
		rows = append(rows, fillLine())
	}

	for len(rows) < height {
		rows = append(rows, fillLine())
	}

	for len(rows) > height {
		rows = rows[:height]
	}

	return strings.Join(rows, "\n")
}
