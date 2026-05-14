package app

import (
	"github.com/bchayka/gitstatus/internal/screens"
	"github.com/bchayka/gitstatus/internal/screens/lobby"
	tea "github.com/charmbracelet/bubbletea"
)

// Update is the top-level message handler. It manages three concerns:
//
//  1. Global side-effects (screen changes, quit) that any screen can emit via
//     screens.Navigate / tea.Quit.
//  2. Global keyboard shortcuts (esc back to lobby, ctrl+c quit) that apply
//     when the current screen isn't holding a text input.
//  3. Forwarding the remaining messages to the active screen.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {

	case tea.WindowSizeMsg:
		a.width = m.Width
		a.height = m.Height
		// Broadcast so screens with cached widths (textareas) can resize.
		for id, s := range a.screens {
			ns, _ := s.Update(msg)
			a.screens[id] = ns
		}
		return a, nil

	case screens.NavigateMsg:
		return a, a.navigate(m.Target)

	case tea.KeyMsg:
		return a.handleKey(m)
	}

	return a, a.updateScreen(msg)
}

// navigate switches the active screen. Entering the lobby for the first time
// triggers the "@boggy entered" join message.
func (a *App) navigate(target string) tea.Cmd {
	if _, ok := a.screens[target]; !ok {
		return nil
	}
	a.current = target
	if target == screens.LobbyID {
		if lob, ok := a.screens[screens.LobbyID].(*lobby.Screen); ok {
			lob.EnsureJoined()
		}
	}
	return nil
}

// handleKey runs global key bindings (esc → lobby, ctrl+c → quit) before
// delegating to the active screen. Screens that own a text input
// (InputFocused()==true) get every key without interception, otherwise typing
// "q" in a chat box would quit instead of typing q.
func (a *App) handleKey(km tea.KeyMsg) (tea.Model, tea.Cmd) {
	scr := a.activeScreen()

	// Intro is fullscreen and consumes all keys (any key skips).
	if a.current == screens.IntroID {
		return a, a.updateScreen(km)
	}

	if scr.InputFocused() {
		return a, a.updateScreen(km)
	}

	switch km.String() {
	case "ctrl+c":
		return a, tea.Quit
	case "esc", "q":
		return a, a.navigate(screens.LobbyID)
	}
	return a, a.updateScreen(km)
}
