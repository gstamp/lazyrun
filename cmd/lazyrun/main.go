package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"lazyrun/internal/repo"
	"lazyrun/internal/tui"
)

func parseRepoArgs(args []string) (owner, name string, err error) {
	switch len(args) {
	case 0:
		o, r, e := repo.Resolve()
		return o, r, e
	case 2:
		if strings.Contains(args[0], "/") || strings.Contains(args[1], "/") {
			return "", "", fmt.Errorf(`usage: lazyrun [owner/repo]` + "\nor: lazyrun owner repo")
		}
		return args[0], strings.TrimSuffix(args[1], ".git"), nil
	case 1:
		part := strings.TrimSuffix(args[0], ".git")
		if strings.Contains(part, "/") {
			segs := strings.SplitN(part, "/", 2)
			if len(segs) != 2 || segs[0] == "" || segs[1] == "" {
				return "", "", fmt.Errorf("invalid slug %q (expected owner/repo)", args[0])
			}
			return segs[0], segs[1], nil
		}
		return "", "", fmt.Errorf(`usage: lazyrun [owner/repo]` + "\nor: lazyrun owner repo")
	default:
		return "", "", fmt.Errorf(`usage: lazyrun [owner/repo]` + "\nor: lazyrun owner repo")
	}
}

func main() {
	owner, name, err := parseRepoArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	p := tea.NewProgram(tui.New(owner, name), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
