package lobby

import (
	"strings"
	"testing"

	"github.com/bchayka/gitstatus/internal/config"
	"github.com/bchayka/gitstatus/internal/screens"
	tea "github.com/charmbracelet/bubbletea"
)

// typeKeys drives the screen through one runes-per-message keystroke per
// character, matching how the bubbletea runtime feeds individual key
// presses into the model. This is what catches divergence between unit
// tests that fast-path the input value and the actual user path.
func typeKeys(t *testing.T, s *Screen, text string) {
	t.Helper()
	for _, r := range text {
		_, _ = s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
}

// TestSlashSpotlightNavigates is the smoke test that the slash dispatch
// emits the right NavigateMsg. /games was retired when the games-in-chat
// model replaced the separate-screen one; the test for that command went
// with it.
func TestSlashSpotlightNavigates(t *testing.T) {
	s := New("@boggy", nil, config.DefaultVibespace())
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

// TestSlashBlitzTypedCharByChar drives /blitz the same way the bubbletea
// runtime does, one KeyRunes message per character, then KeyEnter. This
// catches the case where SetValue-based unit tests pass but the runtime
// path silently drops the command. If this fails, the bug is in the
// keystroke routing, not the slash handler.
func TestSlashBlitzTypedCharByChar(t *testing.T) {
	s := New("@boggy", nil, config.DefaultVibespace())
	typeKeys(t, s, "/blitz")
	if got := s.input.Value(); got != "/blitz" {
		t.Fatalf("after typing /blitz the input should hold /blitz; got %q", got)
	}
	_, _ = s.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if s.blitz == nil {
		t.Fatal("/blitz typed char-by-char should still set s.blitz")
	}
}

// TestSlashBlitzPostsAnnouncement asserts that running /blitz appends
// the system + mod ChatActions to the active channel, carries the
// target word into the mod's body, and surfaces the target prominently
// in the top bar. The chat body cascades through the Settle macro so
// it won't show plain text during the first second of the round; the
// top bar is the always-readable reference.
func TestSlashBlitzPostsAnnouncement(t *testing.T) {
	s := New("@boggy", nil, config.DefaultVibespace())
	before := len(s.channels[s.chatActive].Messages)
	s.input.SetValue("/blitz")

	_, _ = s.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if s.blitz == nil {
		t.Fatal("/blitz should set the screen's active blitz")
	}

	after := len(s.channels[s.chatActive].Messages)
	added := after - before
	if added < 2 {
		t.Fatalf("/blitz should append at least 2 messages (system + mod); got %d new", added)
	}

	target := s.blitz.Target()
	if target == "" {
		t.Fatal("blitz target should be non-empty")
	}

	// The mod message should carry the target word in its body.
	foundMod := false
	for _, msg := range s.channels[s.chatActive].Messages[before:] {
		if strings.Contains(msg.Body, target) {
			foundMod = true
			break
		}
	}
	if !foundMod {
		t.Fatalf("none of the appended messages contain the target word %q", target)
	}

	// The top bar must show the target word in plain text (no cascade)
	// so the round is always-readable even while the mod line scrambles.
	view := s.View(120, 24)
	firstLine := strings.SplitN(view, "\n", 2)[0]
	if !strings.Contains(firstLine, "TARGET") {
		t.Fatalf("top bar should contain TARGET label during a round; got %q", firstLine)
	}
	if !strings.Contains(firstLine, target) {
		t.Fatalf("top bar should contain target %q; got %q", target, firstLine)
	}
}
