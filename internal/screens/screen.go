// Package screens defines the Screen interface that every feature screen
// implements, plus the cross-cutting messages screens use to talk to the app
// (navigation requests, flash messages).
//
// Screens never import each other; they communicate by emitting messages that
// the app router catches. This keeps the dependency graph a star with the app
// at the center and screens as leaves.
package screens

import (
	"github.com/bchayka/gitstatus/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

// Screen is the contract every feature screen implements. The interface is
// intentionally small: Init/Update/View mirror tea.Model, the rest is metadata
// the app uses to compose chrome (header, footer, focus handling).
type Screen interface {
	// Init returns any commands the screen wants to fire when first wired up.
	Init() tea.Cmd

	// Update advances the screen's state in response to a message and returns
	// the new screen value. Screens are value types; the returned Screen
	// replaces the old one in the app's screen map.
	Update(msg tea.Msg) (Screen, tea.Cmd)

	// View renders the screen body. The header and footer are handled by the
	// app; width/height are the budget for the body region only.
	View(width, height int) string

	// Name is the stable id used by the router (e.g. "lobby", "news").
	Name() string

	// Title is the human-readable label shown in the header chip.
	Title() string

	// HeaderContext is optional metadata shown after the title (e.g. active
	// channel name, scroll position). Empty string means no extra content.
	HeaderContext() string

	// Footer returns the key hints shown in the status bar.
	Footer() []KeyHint

	// InputFocused returns true when the screen has an active text input and
	// the app's global key handlers should defer to the screen. Without this,
	// pressing 'q' in a text field would quit instead of typing q.
	InputFocused() bool
}

// KeyHint is one entry in the footer status bar.
type KeyHint struct {
	Key, Desc string
}

// Entrant is an optional interface for screens that want a hook the router
// runs when they become active. Used for field-driven entry effects: pulse
// the backdrop, register a welcome overlay, palette shift on navigate.
type Entrant interface {
	OnEnter()
}

// NavigateMsg requests a screen switch. Emitted by Navigate; handled by the
// app router.
type NavigateMsg struct{ Target string }

// Navigate returns a tea.Cmd that asks the app to switch to the named screen.
func Navigate(target string) tea.Cmd {
	return func() tea.Msg { return NavigateMsg{Target: target} }
}

// FlashMsg sets the transient status message shown in the footer.
type FlashMsg struct{ Text string }

// Flash returns a tea.Cmd that posts a footer flash.
func Flash(text string) tea.Cmd {
	return func() tea.Msg { return FlashMsg{Text: text} }
}

// QuitMsg signals an orderly quit (used by /quit and similar slash commands).
type QuitMsg struct{}

// Quit returns a tea.Cmd that quits the app.
func Quit() tea.Cmd {
	return tea.Quit
}

// OpenURL launches the OS browser at url and surfaces the outcome as a footer
// flash. Screens use this for "enter to open" affordances on news items, repo
// listings, spotlight cards, etc. — anywhere a URL is the expected target.
func OpenURL(url string) tea.Cmd {
	if err := ui.OpenURL(url); err != nil {
		return Flash("couldn't open: " + err.Error())
	}
	return Flash("opened: " + url)
}

// Screen ids used as keys in the app's screen map and as Navigate targets.
// The room runs in three places. The lobby is where conversation happens,
// where the moderator surfaces a spotlit project, and where games run as
// chat events. The spotlight screen carries the full reading view for the
// currently surfaced project. The games screen owns the bricks blitz
// while the room watches.
const (
	IntroID     = "intro"
	LobbyID     = "lobby"
	SpotlightID = "spotlight"
	GamesID     = "games"
)
