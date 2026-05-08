package clipboard

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

func writeOSC52(w io.Writer, text string) error {
	payload := base64.StdEncoding.EncodeToString([]byte(text))
	_, err := fmt.Fprintf(w, "\x1b]52;c;%s\x07", payload)
	return err
}

// Copy mirrors the heuristic used previously in lazyrun (OSC‑52 plus OS helpers).
func Copy(text string, stdout io.Writer) (bool, error) {
	if stdout == nil {
		stdout = os.Stdout
	}
	if writeOSC52(stdout, text) == nil {
		return true, nil
	}

	data := []byte(text)

	if _, err := exec.LookPath("pbcopy"); err == nil {
		cmd := exec.Command("pbcopy")
		cmd.Stdin = bytes.NewReader(data)
		if err := cmd.Run(); err == nil {
			return true, nil
		}
	}
	if _, err := exec.LookPath("xclip"); err == nil {
		cmd := exec.Command("xclip", "-selection", "clipboard")
		cmd.Stdin = bytes.NewReader(data)
		if err := cmd.Run(); err == nil {
			return true, nil
		}
	}
	if _, err := exec.LookPath("wl-copy"); err == nil {
		cmd := exec.Command("wl-copy")
		cmd.Stdin = bytes.NewReader(data)
		if err := cmd.Run(); err == nil {
			return true, nil
		}
	}
	return false, errors.New("clipboard unavailable")
}
