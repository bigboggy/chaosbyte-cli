// vibespace is the local single-user entry point for the workshop. It
// runs the bubbletea app directly in the terminal, with no SSH, no
// broker, and no Wish middleware. Use it during development to see
// rendered output the same way an end user would, without the SSH
// rendering path in the middle.
//
// The flagship Vibespace theme and config are baked in via
// config.DefaultVibespace(); other teams' configs are an SSH-server
// concern and aren't exposed locally.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/bchayka/gitstatus/internal/app"
	"github.com/bchayka/gitstatus/internal/config"
	"github.com/bchayka/gitstatus/internal/theme"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	noAltScreen := flag.Bool("no-alt-screen", false, "render inline instead of switching to the alt screen, useful when diagnosing render-diff issues")
	flag.Parse()

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

	nick := "@boggy"
	m := app.New(nick, nil, cfg)
	opts := []tea.ProgramOption{tea.WithMouseCellMotion()}
	if !*noAltScreen {
		opts = append(opts, tea.WithAltScreen())
	}
	p := tea.NewProgram(m, opts...)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
