// Package app is the bubbletea Model that wires everything together.
//
// One App per session. App owns the intro + lobby screens for this session;
// chat state itself lives in the shared *hub.Hub passed to New.
package app

import (
	"github.com/bchayka/gitstatus/internal/hub"
	"github.com/bchayka/gitstatus/internal/screens"
	"github.com/bchayka/gitstatus/internal/screens/intro"
	"github.com/bchayka/gitstatus/internal/screens/lobby"
	"github.com/bchayka/gitstatus/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

// App is the top-level bubbletea Model.
type App struct {
	screens map[string]screens.Screen
	current string
	lobby   *lobby.Screen // kept for Cleanup

	width, height int
}

// New constructs a session app. meUser is the participant's display name (e.g.
// "@boggy"); h is the shared chat backend. The intro screen is the initial
// active screen; it emits Navigate(lobby) when its animation ends.
func New(meUser string, h *hub.Hub) *App {
	lob := lobby.New(meUser, h)
	return &App{
		screens: map[string]screens.Screen{
			screens.IntroID: intro.New(),
			screens.LobbyID: lob,
		},
		current: screens.IntroID,
		lobby:   lob,
	}
}

func (a *App) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, s := range a.screens {
		if c := s.Init(); c != nil {
			cmds = append(cmds, c)
		}
	}
	return tea.Batch(cmds...)
}

// Cleanup releases per-session resources (hub subscription). Safe to call more
// than once.
func (a *App) Cleanup() {
	if a.lobby != nil {
		a.lobby.Cleanup()
	}
}

// activeScreen returns the screen referenced by a.current, falling back to the
// lobby if the id is somehow stale.
func (a *App) activeScreen() screens.Screen {
	if s, ok := a.screens[a.current]; ok {
		return s
	}
	return a.screens[screens.LobbyID]
}

// updateScreen forwards a message to the active screen and writes the result
// back into the map.
func (a *App) updateScreen(msg tea.Msg) tea.Cmd {
	ns, cmd := a.activeScreen().Update(msg)
	a.screens[a.current] = ns
	return cmd
}

func (a *App) View() string {
	if a.width < ui.MinWidth || a.height < ui.MinHeight {
		return tooSmall(a.width, a.height)
	}
	return a.renderFrame()
}
