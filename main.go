// vibespace — a TUI lobby for devs and vibe coders.
//
// This is the local-mode entrypoint: one user, one hub, one bubbletea program.
// The SSH-server entrypoint lives in cmd/server.
package main

import (
	"fmt"
	"os"
	"os/user"

	"github.com/bchayka/gitstatus/internal/app"
	"github.com/bchayka/gitstatus/internal/hub"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	h := hub.New()
	p := tea.NewProgram(app.New(localUser(), "", "", h, nil), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "vibespace: %v\n", err)
		os.Exit(1)
	}
}

func localUser() string {
	if u, err := user.Current(); err == nil && u.Username != "" {
		return "@" + u.Username
	}
	return "@local"
}
