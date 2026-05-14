// Package app is the bubbletea Model that wires everything together.
//
// The app owns the map of Screens (intro + lobby) and the viewport dimensions.
// Each Screen is responsible for its own state — the dependency graph is a
// star: app → screens, screens never reach back into app or sideways.
package app

import (
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

	width, height int
}

// New constructs the app with all screens wired up. The intro screen is the
// initial active screen; it emits Navigate(lobby) when its animation ends.
func New() *App {
	return &App{
		screens: map[string]screens.Screen{
			screens.IntroID: intro.New(),
			screens.LobbyID: lobby.New(),
		},
		current: screens.IntroID,
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
