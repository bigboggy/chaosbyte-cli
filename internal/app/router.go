package app

import (
	"github.com/bchayka/gitstatus/internal/screens"
	"github.com/bchayka/gitstatus/internal/screens/discussions"
	"github.com/bchayka/gitstatus/internal/screens/games"
	"github.com/bchayka/gitstatus/internal/screens/lobby"
	tea "github.com/charmbracelet/bubbletea"
)

// Update is the top-level message handler. It manages three concerns:
//
//  1. Global side-effects (screen changes, flash, quit) that any screen can
//     emit via screens.Navigate / screens.Flash / tea.Quit.
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

	case tickMsg:
		a.expireFlash()
		return a, tickEvery()

	case screens.NavigateMsg:
		return a, a.navigate(m.Target)

	case screens.FlashMsg:
		a.setFlash(m.Text)
		return a, nil

	case tea.KeyMsg:
		return a.handleKey(m)
	}

	return a, a.updateScreen(msg)
}

// navigate switches the active screen. Some transitions have side effects:
// going to the lobby for the first time triggers the "@boggy entered" join
// message; entering games/discussions drops any sub-mode back to its launcher.
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
	// Field-driven entry effect: screens that implement Entrant get a hook
	// to pulse their backdrop, register a welcome overlay, or otherwise
	// drive the engine into the moment.
	if ent, ok := a.screens[target].(screens.Entrant); ok {
		ent.OnEnter()
	}
	// Re-initialise the activated screen so screens that drive their own
	// ticks (intro, ambient) can schedule a fresh tick chain. App.Init only
	// runs once at program start; without this hook a screen with its own
	// tick stays dormant after a re-entry.
	return a.screens[target].Init()
}

// handleKey runs global key bindings (esc → lobby, ctrl+c → quit, q → lobby)
// before delegating to the active screen. Screens that own a text input
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
	case "esc":
		if a.interceptEsc() {
			return a, nil
		}
		return a, a.navigate(screens.LobbyID)
	case "q":
		return a, a.navigate(screens.LobbyID)
	}
	return a, a.updateScreen(km)
}

// interceptEsc lets screens with sub-modes (discussions popups, games launcher)
// pop one level before esc falls through to "back to lobby".
func (a *App) interceptEsc() bool {
	switch s := a.screens[a.current].(type) {
	case *discussions.Screen:
		return s.BackOut()
	case *games.Screen:
		return s.BackToList()
	}
	return false
}
