// Package app is the bubbletea Model that wires everything together.
//
// The app owns:
//   - the map of Screens (one per feature)
//   - the id of the active screen
//   - the viewport dimensions
//   - a transient flash message (set via screens.Flash; auto-clears after 3s)
//
// The app does NOT own any per-screen state — each Screen is responsible for
// its own data. This keeps the dependency graph a star: app → screens, and
// screens never reach back into app or sideways into each other.
package app

import (
	"time"

	"github.com/bchayka/gitstatus/internal/room"
	"github.com/bchayka/gitstatus/internal/screens"
	"github.com/bchayka/gitstatus/internal/screens/ambient"
	"github.com/bchayka/gitstatus/internal/screens/discussions"
	"github.com/bchayka/gitstatus/internal/screens/games"
	"github.com/bchayka/gitstatus/internal/screens/intro"
	"github.com/bchayka/gitstatus/internal/screens/lobby"
	"github.com/bchayka/gitstatus/internal/screens/news"
	"github.com/bchayka/gitstatus/internal/screens/resources"
	"github.com/bchayka/gitstatus/internal/screens/spotlight"
	"github.com/bchayka/gitstatus/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

// tickMsg fires every second; used to expire the flash message.
type tickMsg time.Time

func tickEvery() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// App is the top-level bubbletea Model.
type App struct {
	screens map[string]screens.Screen
	current string

	width, height int

	flash   string
	flashAt time.Time
}

// New constructs the app with all screens wired up. nick is the user's chat
// handle (e.g. "@boggy"); broker carries shared room state across sessions
// when chaosbyte runs as an SSH server. broker may be nil for fully-local
// single-session mode. The intro screen is the initial active screen; it
// emits Navigate(lobby) when its animation ends.
func New(nick string, broker *room.Broker) *App {
	a := &App{
		screens: map[string]screens.Screen{
			screens.IntroID:       intro.New(),
			screens.LobbyID:       lobby.New(nick, broker),
			screens.NewsID:        news.New(),
			screens.ResourcesID:   resources.New(),
			screens.SpotlightID:   spotlight.New(),
			screens.GamesID:       games.New(),
			screens.DiscussionsID: discussions.New(),
			screens.AmbientID:     ambient.New(),
		},
		current: screens.IntroID,
	}
	return a
}

func (a *App) Init() tea.Cmd {
	cmds := []tea.Cmd{tickEvery()}
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

// setFlash starts a transient footer message that the tick will clear.
func (a *App) setFlash(text string) {
	a.flash = text
	a.flashAt = time.Now()
}

// expireFlash zeros the flash if it's older than 3 seconds.
func (a *App) expireFlash() {
	if !a.flashAt.IsZero() && time.Since(a.flashAt) > 3*time.Second {
		a.flash = ""
		a.flashAt = time.Time{}
	}
}

// updateScreen forwards a message to the active screen and writes the result
// back into the map.
func (a *App) updateScreen(msg tea.Msg) tea.Cmd {
	ns, cmd := a.activeScreen().Update(msg)
	a.screens[a.current] = ns
	return cmd
}

// Tiny adapter so bubbletea sees us as a value-style tea.Model while we keep
// pointer semantics internally. Bubbletea calls Update on a copy of the value,
// but because *App is the actual model the pointer keeps state alive across
// frames.
func (a *App) View() string {
	if a.width < ui.MinWidth || a.height < ui.MinHeight {
		return tooSmall(a.width, a.height)
	}
	return a.renderFrame()
}
