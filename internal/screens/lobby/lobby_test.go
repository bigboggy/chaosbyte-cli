package lobby

import (
	"testing"

	"github.com/bchayka/gitstatus/internal/config"
	"github.com/bchayka/gitstatus/internal/screens"
	tea "github.com/charmbracelet/bubbletea"
)

// TestSlashGamesNavigates asserts that typing "/games" + enter in the lobby
// emits a NavigateMsg targeting the games screen. The user reported the
// command did nothing; this nails down whether the bug is in the slash
// dispatch or somewhere upstream.
func TestSlashGamesNavigates(t *testing.T) {
	s := New("@boggy", nil, config.DefaultChaosbyte())
	s.input.SetValue("/games")

	out, cmd := s.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if out == nil {
		t.Fatal("handleKey returned nil screen")
	}
	if cmd == nil {
		t.Fatal("handleKey returned nil cmd — submit dropped the slash command")
	}
	msg := cmd()
	nav, ok := msg.(screens.NavigateMsg)
	if !ok {
		t.Fatalf("expected NavigateMsg, got %T (%+v)", msg, msg)
	}
	if nav.Target != screens.GamesID {
		t.Fatalf("Target = %q, want %q", nav.Target, screens.GamesID)
	}
}

// TestSlashSpotlightNavigates is the same check for /spotlight.
func TestSlashSpotlightNavigates(t *testing.T) {
	s := New("@boggy", nil, config.DefaultChaosbyte())
	s.input.SetValue("/spotlight")

	_, cmd := s.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("nil cmd")
	}
	msg := cmd()
	nav, ok := msg.(screens.NavigateMsg)
	if !ok {
		t.Fatalf("expected NavigateMsg, got %T", msg)
	}
	if nav.Target != screens.SpotlightID {
		t.Fatalf("Target = %q, want %q", nav.Target, screens.SpotlightID)
	}
}
