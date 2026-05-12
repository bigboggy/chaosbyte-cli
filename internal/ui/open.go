package ui

import (
	"errors"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
)

// OpenURL launches the OS default browser at the given URL. It validates that
// the URL has an http/https scheme to avoid invoking arbitrary handlers (e.g.
// file://, mailto:, custom schemes), and starts the browser in the background
// so the TUI doesn't block waiting for it.
//
// Returns an error if the URL is malformed, the scheme is unsupported, or the
// OS-specific launcher isn't available.
func OpenURL(rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return errors.New("empty url")
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("only http/https urls are supported")
	}

	cmd, err := browserCommand(rawURL)
	if err != nil {
		return err
	}
	return cmd.Start()
}

// browserCommand returns the platform-specific exec.Cmd that opens url in the
// default browser. Split out from OpenURL so platforms can be tested or
// extended independently.
func browserCommand(rawURL string) (*exec.Cmd, error) {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", rawURL), nil
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL), nil
	default:
		// linux, *bsd, etc — xdg-open is the de-facto standard
		return exec.Command("xdg-open", rawURL), nil
	}
}
