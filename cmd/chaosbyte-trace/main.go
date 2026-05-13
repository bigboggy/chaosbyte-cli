// chaosbyte-trace drives the App's Update method directly with synthetic
// messages so we can verify what state /blitz produces without going
// through a TTY. It mirrors the bubbletea event loop: dispatches each
// command's returned Cmd, expands NavigateMsg into a follow-up Update,
// and renders the final View.
package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/bchayka/gitstatus/internal/app"
	"github.com/bchayka/gitstatus/internal/config"
	"github.com/bchayka/gitstatus/internal/field"
	"github.com/bchayka/gitstatus/internal/screens"
	"github.com/bchayka/gitstatus/internal/theme"
	tea "github.com/charmbracelet/bubbletea"
)

type modelLike interface {
	Update(tea.Msg) (tea.Model, tea.Cmd)
	View() string
}

func dispatch(model tea.Model, msg tea.Msg) tea.Model {
	m, cmd := model.Update(msg)
	for cmd != nil {
		out := cmd()
		if out == nil {
			break
		}
		// Skip ticks; we'll inject them manually.
		switch out.(type) {
		case field.TickMsg:
			cmd = nil
			continue
		}
		m, cmd = m.Update(out)
	}
	return m
}

func main() {
	cfg := config.DefaultChaosbyte()
	theme.Apply(theme.Palette{
		Bg: cfg.Theme.Bg, Fg: cfg.Theme.Fg, Muted: cfg.Theme.Muted,
		Accent: cfg.Theme.Accent, Accent2: cfg.Theme.Accent2,
		BorderHi: cfg.Theme.BorderHi, BorderLo: cfg.Theme.BorderLo,
	})

	var m tea.Model = app.New("@boggy", nil, cfg)
	// Initialize.
	if c := m.(*app.App).Init(); c != nil {
		// drain init cmds
		_ = c
	}

	// Window size so View renders.
	m = dispatch(m, tea.WindowSizeMsg{Width: 120, Height: 30})

	// Navigate intro -> lobby directly (simulates intro's any-key behavior).
	m = dispatch(m, screens.NavigateMsg{Target: screens.LobbyID})

	// Type /blitz char by char.
	for _, r := range "/blitz" {
		m = dispatch(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	// Enter.
	m = dispatch(m, tea.KeyMsg{Type: tea.KeyEnter})

	// A few simulated ticks to let any state-change render.
	now := time.Now()
	for i := 0; i < 5; i++ {
		m = dispatch(m, field.TickMsg(now.Add(time.Duration(i*16)*time.Millisecond)))
	}

	view := m.View()
	// Strip ANSI for readability.
	stripped := stripAnsi(view)
	fmt.Println("--- View after /blitz ---")
	fmt.Println(stripped)
	fmt.Println("--- end view ---")
	if strings.Contains(stripped, "TARGET") {
		fmt.Println("RESULT: top bar contains TARGET. /blitz fired and rendered.")
	} else if strings.Contains(stripped, "BLITZ") {
		fmt.Println("RESULT: banner contains BLITZ. /blitz fired.")
	} else {
		fmt.Println("RESULT: no TARGET or BLITZ token found in rendered View. /blitz did not visibly fire.")
	}
}

func stripAnsi(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
