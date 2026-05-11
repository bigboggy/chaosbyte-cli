// chaosbyte — a TUI lobby for devs and vibe coders.
//
// All real work lives in internal/. main.go is just the entrypoint that wires
// the bubbletea program to internal/app.
package main

import (
	"fmt"
	"os"

	"github.com/bchayka/gitstatus/internal/app"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	p := tea.NewProgram(app.New(), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "chaosbyte: %v\n", err)
		os.Exit(1)
	}
}
