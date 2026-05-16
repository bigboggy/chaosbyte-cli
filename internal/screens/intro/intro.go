// Package intro is the boot animation that plays once at startup.
//
// The animation is purely a function of elapsed time since Start was called,
// so the renderer is stateless apart from that timestamp. A 33ms tick keeps
// the View redraw rate at ~30fps until the final phase, at which point the
// screen emits Navigate(lobby) and stops ticking.
package intro

import (
	"time"

	"github.com/bigboggy/vibespace/internal/screens"
	"github.com/bigboggy/vibespace/internal/theme"
	tea "github.com/charmbracelet/bubbletea"
)

// TickMsg fires at ~30fps while the intro is on screen. The screen schedules
// the next one from its Update.
type TickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(33*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// Screen is the intro screen. It holds only the start timestamp and the
// shared styles handle; everything else is derived per-frame.
type Screen struct {
	styles *theme.Styles
	start  time.Time
	done   bool
}

// New returns an intro screen whose clock starts now.
func New(styles *theme.Styles) *Screen {
	return &Screen{styles: styles, start: time.Now()}
}

func (s *Screen) Init() tea.Cmd { return tickCmd() }

func (s *Screen) Update(msg tea.Msg) (screens.Screen, tea.Cmd) {
	switch msg.(type) {
	case TickMsg:
		if s.done {
			return s, nil
		}
		if time.Since(s.start).Milliseconds() >= phaseFadeEnd {
			s.done = true
			return s, screens.Navigate(screens.LobbyID)
		}
		return s, tickCmd()
	case tea.KeyMsg:
		// any key skips the intro
		if !s.done {
			s.done = true
			return s, screens.Navigate(screens.LobbyID)
		}
	}
	return s, nil
}

func (s *Screen) Name() string  { return screens.IntroID }
func (s *Screen) Title() string { return "intro" }

func (s *Screen) HeaderContext() string { return "" }
func (s *Screen) InputFocused() bool    { return false }

func (s *Screen) Footer() []screens.KeyHint {
	return []screens.KeyHint{
		{Key: "any key", Desc: "skip"},
		{Key: "ctrl+c", Desc: "quit"},
	}
}
