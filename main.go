// The local entry point for the platform. The same Go binary that hosts
// the SSH server runs locally as a single-session client against the
// in-process broker. The flagship Vibespace configuration loads here. A
// different team's room is the same binary loaded with a different config,
// which is the entire point of the platform layer.
//
// Local mode bypasses pubkey auth: a sentinel LocalPrincipal stands in for
// a real user. Production deployments use cmd/vibespace-server.
package main

import (
	"fmt"
	"os"

	"github.com/bchayka/gitstatus/internal/app"
	"github.com/bchayka/gitstatus/internal/config"
	"github.com/bchayka/gitstatus/internal/identity"
	"github.com/bchayka/gitstatus/internal/room"
	"github.com/bchayka/gitstatus/internal/store/memory"
	"github.com/bchayka/gitstatus/internal/theme"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	cfg := config.DefaultVibespace()
	theme.Apply(theme.Palette{
		Bg:       cfg.Theme.Bg,
		Fg:       cfg.Theme.Fg,
		Muted:    cfg.Theme.Muted,
		Accent:   cfg.Theme.Accent,
		Accent2:  cfg.Theme.Accent2,
		BorderHi: cfg.Theme.BorderHi,
		BorderLo: cfg.Theme.BorderLo,
	})
	broker := room.New(cfg.Slug, nil, nil, memory.New())
	defer broker.Stop()
	principal := identity.LocalPrincipal()
	p := tea.NewProgram(app.New(principal, broker, cfg), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "vibespace: %v\n", err)
		os.Exit(1)
	}
}
