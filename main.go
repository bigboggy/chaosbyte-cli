// chaosbyte — a TUI lobby for devs and vibe coders.
//
// All real work lives in internal/. main.go is just the entrypoint that wires
// the bubbletea program to internal/app. The lobby is hosted by a local
// broker so the rules-v0 moderator runs the same code path here as it does
// inside the SSH server (cmd/chaosbyte-server).
package main

import (
	"fmt"
	"os"

	"github.com/bchayka/gitstatus/internal/app"
	"github.com/bchayka/gitstatus/internal/room"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	nick := os.Getenv("USER")
	if nick == "" {
		nick = "boggy"
	}
	broker := room.New()
	defer broker.Stop()
	p := tea.NewProgram(app.New("@"+nick, broker), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "chaosbyte: %v\n", err)
		os.Exit(1)
	}
}
