package draw

import (
	"fmt"
	"strings"
)

// Style is minimal inline styling — truecolor escape codes.
type Style struct {
	Fg   string // "#rrggbb" or ""
	Bg   string // "#rrggbb" or ""
	Bold bool
}

const Reset = "\x1b[0m"

func (s Style) ansiPrefix() string {
	var buf strings.Builder

	if s.Fg != "" {
		if rf, gf, bf, ok := parseHex(s.Fg); ok {
			fmt.Fprintf(&buf, "\x1b[38;2;%d;%d;%dm", rf, gf, bf)
		}
	}

	if s.Bg != "" {
		if rf, gf, bf, ok := parseHex(s.Bg); ok {
			fmt.Fprintf(&buf, "\x1b[48;2;%d;%d;%dm", rf, gf, bf)
		}
	}

	if s.Bold {
		buf.WriteString("\x1b[1m")
	}

	return buf.String()
}

// Sprint prefixes text with the style escape sequence and terminates with Reset.
func (s Style) Sprint(parts ...string) string {
	pre := s.ansiPrefix()
	if pre == "" && !s.Bold && s.Fg == "" && s.Bg == "" {
		return strings.Join(parts, "")
	}

	return pre + strings.Join(parts, "") + Reset
}

func parseHex(hex string) (r, g, b int, ok bool) {
	hex = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(hex), "#"))
	if len(hex) != 6 {
		return 0, 0, 0, false
	}

	var rr, gg, bb byte
	n, err := fmt.Sscanf(strings.ToLower(hex), "%2x%2x%2x", &rr, &gg, &bb)
	if err != nil || n != 3 {
		return 0, 0, 0, false
	}

	return int(rr), int(gg), int(bb), true
}
