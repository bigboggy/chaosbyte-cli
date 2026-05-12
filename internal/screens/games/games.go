// Package games is the mini-games launcher. Most entries are placeholders;
// "bug hunter" is the one playable game.
package games

import (
	"fmt"
	"strings"

	"github.com/bchayka/gitstatus/internal/screens"
	"github.com/bchayka/gitstatus/internal/theme"
	"github.com/bchayka/gitstatus/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	stateList state = iota
	statePlayBugHunter
)

type Screen struct {
	games []Game
	idx   int
	state state
	bug   bugHunterState
}

func New() *Screen {
	return &Screen{
		games: seedGames(),
		bug:   newBugHunter(),
	}
}

func (s *Screen) Init() tea.Cmd { return nil }

func (s *Screen) Name() string  { return screens.GamesID }
func (s *Screen) Title() string { return "games" }

func (s *Screen) HeaderContext() string {
	return lipgloss.NewStyle().Foreground(theme.Muted).
		Render(fmt.Sprintf("%d games", len(s.games)))
}

func (s *Screen) Footer() []screens.KeyHint {
	if s.state == statePlayBugHunter {
		return []screens.KeyHint{
			{Key: "0-9", Desc: "type"}, {Key: "enter", Desc: "guess"}, {Key: "r", Desc: "reset"}, {Key: "esc", Desc: "back"},
		}
	}
	return []screens.KeyHint{
		{Key: "j/k", Desc: "move"}, {Key: "enter", Desc: "play"}, {Key: "esc", Desc: "lobby"},
	}
}

func (s *Screen) InputFocused() bool { return false }

// BackToList is called by the app when esc is pressed and we're inside a game;
// it pops back to the launcher list instead of all the way to the lobby.
// Returns true if the screen handled the esc.
func (s *Screen) BackToList() bool {
	if s.state != stateList {
		s.state = stateList
		return true
	}
	return false
}

func (s *Screen) Update(msg tea.Msg) (screens.Screen, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}
	switch s.state {
	case stateList:
		return s.updateList(km)
	case statePlayBugHunter:
		return s, s.updateBugHunter(km)
	}
	return s, nil
}

func (s *Screen) updateList(km tea.KeyMsg) (screens.Screen, tea.Cmd) {
	switch km.String() {
	case "j", "down":
		if s.idx < len(s.games)-1 {
			s.idx++
		}
	case "k", "up":
		if s.idx > 0 {
			s.idx--
		}
	case "enter", "o":
		if s.idx >= len(s.games) {
			return s, nil
		}
		g := s.games[s.idx]
		if !g.Playable {
			return s, screens.Flash(g.Name + " is still in alpha. probably forever.")
		}
		switch g.Name {
		case "bug hunter":
			s.state = statePlayBugHunter
			s.bug = newBugHunter()
		}
	}
	return s, nil
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (s *Screen) View(width, height int) string {
	if s.state == statePlayBugHunter {
		return s.renderBugHunter(width, height)
	}
	return s.renderList(width, height)
}

func (s *Screen) renderList(width, height int) string {
	w := ui.FeedShellWidth(width)
	contentW := w - 2

	title := theme.Title.Render("games · mini-distractions for tired devs")
	subtitle := lipgloss.NewStyle().Foreground(theme.Muted).Italic(true).
		Render("when the compiler is sulking and the standup hasn't started yet")

	var rows []string
	for i, g := range s.games {
		rows = append(rows, renderGameRow(g, contentW, i == s.idx))
	}
	stacked := lipgloss.JoinVertical(lipgloss.Left, title, subtitle, "", strings.Join(rows, "\n"))
	return lipgloss.Place(width, height, lipgloss.Left, lipgloss.Top, stacked)
}

func renderGameRow(g Game, width int, focused bool) string {
	marker := "  "
	if focused {
		marker = "▸ "
	}
	statusTag := lipgloss.NewStyle().Foreground(theme.OK).Render("playable")
	if !g.Playable {
		statusTag = lipgloss.NewStyle().Foreground(theme.Muted).Render("soon™")
	}
	name := lipgloss.NewStyle().Foreground(theme.Accent2).Bold(true).Render(fmt.Sprintf("%-18s", g.Name))
	desc := lipgloss.NewStyle().Foreground(theme.Fg).Render(ui.Truncate(g.Description, width-32))
	line := fmt.Sprintf("%s%s  %s  %s", marker, name, statusTag, desc)
	if focused {
		return theme.BranchItemSel.Width(width).Render(line)
	}
	return theme.BranchItem.Render(line)
}
