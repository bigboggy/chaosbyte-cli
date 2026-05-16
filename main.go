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
	"github.com/bchayka/gitstatus/internal/store"
	"github.com/bchayka/gitstatus/internal/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func main() {
	h := hub.New()
	// Local-mode SQLite lives under the user's config dir so profiles persist
	// across runs without polluting the working directory.
	dbPath := localDBPath()
	data, err := store.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vibespace: store: %v\n", err)
		os.Exit(1)
	}
	defer data.Close()

	styles := theme.New(lipgloss.DefaultRenderer(), theme.Default())
	p := tea.NewProgram(
		app.New(styles, localUser(), "", "", h, nil, data),
		tea.WithAltScreen(), tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "vibespace: %v\n", err)
		os.Exit(1)
	}
}

func localDBPath() string {
	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		root := dir + "/vibespace"
		_ = os.MkdirAll(root, 0o700)
		return root + "/vibespace.db"
	}
	return "./vibespace.db"
}

func localUser() string {
	if u, err := user.Current(); err == nil && u.Username != "" {
		return "@" + u.Username
	}
	return "@local"
}
