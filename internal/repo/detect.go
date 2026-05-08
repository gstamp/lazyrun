package repo

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var remoteRegexp = regexp.MustCompile(`github\.com[:/]([^/]+)/([^/]+?)(?:\.git)?$`)

// Resolve returns owner/repo inferred from git remote URL.
func Resolve() (owner, slug string, err error) {
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return "", "", fmt.Errorf("no git remote 'origin': %w", err)
	}
	remote := strings.TrimSpace(string(out))
	matches := remoteRegexp.FindStringSubmatch(remote)
	if len(matches) != 3 {
		return "", "", fmt.Errorf("not a GitHub remote: %s", remote)
	}
	owner, slug = matches[1], strings.TrimSuffix(matches[2], ".git")
	return owner, slug, nil
}
