package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Game struct {
	Name        string
	Description string
	Playable    bool
}

type gameState int

const (
	gameStateList gameState = iota
	gameStateBugHunter
)

type bugHunterState struct {
	Target    int
	Guess     int
	Attempts  int
	Hint      string
	Done      bool
	StartedAt time.Time
}

func newBugHunter() bugHunterState {
	// deterministic-ish: seed from current second so it changes between sessions
	t := int(time.Now().UnixNano() % 100)
	if t < 1 {
		t = 1
	}
	return bugHunterState{
		Target:    t,
		Hint:      "i'm thinking of a line number between 1 and 100. press 0-9 to dial, enter to guess.",
		StartedAt: time.Now(),
	}
}

func seedGames() []Game {
	return []Game{
		{"bug hunter", "guess which line the bug is on (1-100). lower attempts = more dignity.", true},
		{"sha sprint", "memorize a 7-char SHA in 3s, then type it back. coming soon.", false},
		{"vibe roulette", "spin the wheel of vibes. land on 'ship it' or 'rewrite in rust'. coming soon.", false},
		{"rubber duck", "explain your bug to a duck. the duck has opinions. coming soon.", false},
		{"git blame bingo", "fill the card with classic blame quotes. coming soon.", false},
	}
}

func (m model) updateGames(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.gameState {
	case gameStateList:
		return m.updateGamesList(msg)
	case gameStateBugHunter:
		return m.updateBugHunter(msg)
	}
	return m, nil
}

func (m model) updateGamesList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.gameIdx < len(m.games)-1 {
			m.gameIdx++
		}
	case "k", "up":
		if m.gameIdx > 0 {
			m.gameIdx--
		}
	case "enter", "o":
		if m.gameIdx < len(m.games) {
			if m.games[m.gameIdx].Playable {
				switch m.games[m.gameIdx].Name {
				case "bug hunter":
					m.gameState = gameStateBugHunter
					m.bugHunter = newBugHunter()
				}
			} else {
				m.setFlash(m.games[m.gameIdx].Name + " is still in alpha. probably forever.")
			}
		}
	}
	return m, nil
}

func (m model) updateBugHunter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	bh := &m.bugHunter
	if bh.Done {
		switch msg.String() {
		case "enter", " ", "r":
			m.bugHunter = newBugHunter()
		}
		return m, nil
	}
	s := msg.String()
	if len(s) == 1 && s >= "0" && s <= "9" {
		d := int(s[0] - '0')
		if bh.Guess == 0 {
			bh.Guess = d
		} else if bh.Guess < 100 {
			bh.Guess = bh.Guess*10 + d
			if bh.Guess > 999 {
				bh.Guess = bh.Guess % 1000
			}
		}
		return m, nil
	}
	switch msg.String() {
	case "backspace":
		bh.Guess /= 10
	case "enter":
		if bh.Guess < 1 || bh.Guess > 100 {
			bh.Hint = "out of bounds. 1-100, friend."
			return m, nil
		}
		bh.Attempts++
		if bh.Guess == bh.Target {
			bh.Done = true
			elapsed := time.Since(bh.StartedAt).Truncate(time.Second)
			bh.Hint = fmt.Sprintf("FOUND IT. line %d, in %d attempts, in %s. ship it.", bh.Target, bh.Attempts, elapsed)
		} else if bh.Guess < bh.Target {
			bh.Hint = fmt.Sprintf("nope, %d is too early. the bug is deeper.", bh.Guess)
		} else {
			bh.Hint = fmt.Sprintf("nope, %d is too late. the bug is earlier.", bh.Guess)
		}
		bh.Guess = 0
	case "r":
		m.bugHunter = newBugHunter()
	}
	return m, nil
}

func (m model) renderGames(width, height int) string {
	switch m.gameState {
	case gameStateBugHunter:
		return m.renderBugHunter(width, height)
	}
	return m.renderGamesList(width, height)
}

func (m model) renderGamesList(width, height int) string {
	w := feedShellWidth(width)
	contentW := w - 2

	title := titleStyle.Render("games · mini-distractions for tired devs")
	subtitle := lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
		Render("when the compiler is sulking and the standup hasn't started yet")

	var rows []string
	for i, g := range m.games {
		marker := "  "
		if i == m.gameIdx {
			marker = "▸ "
		}
		nameStyle := lipgloss.NewStyle().Foreground(colorAccent2).Bold(true)
		statusTag := lipgloss.NewStyle().Foreground(colorOk).Render("playable")
		if !g.Playable {
			statusTag = lipgloss.NewStyle().Foreground(colorMuted).Render("soon™")
		}
		name := nameStyle.Render(fmt.Sprintf("%-18s", g.Name))
		desc := lipgloss.NewStyle().Foreground(colorFg).Render(truncate(g.Description, contentW-32))
		line := fmt.Sprintf("%s%s  %s  %s", marker, name, statusTag, desc)
		if i == m.gameIdx {
			line = branchItemSelStyle.Width(contentW).Render(line)
		} else {
			line = branchItemStyle.Render(line)
		}
		rows = append(rows, line)
	}

	stacked := lipgloss.JoinVertical(lipgloss.Left, title, subtitle, "", strings.Join(rows, "\n"))
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Top, stacked)
}

func (m model) renderBugHunter(width, height int) string {
	bh := m.bugHunter
	title := titleStyle.Render("bug hunter")
	subtitle := lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
		Render("guess the line number where the bug is hiding (1-100)")

	guess := lipgloss.NewStyle().Foreground(colorAccent2).Bold(true).
		Render(fmt.Sprintf("your guess: %d", bh.Guess))
	attempts := lipgloss.NewStyle().Foreground(colorMuted).
		Render(fmt.Sprintf("attempts: %d", bh.Attempts))

	hintColor := colorWarn
	if bh.Done {
		hintColor = colorOk
	}
	hint := lipgloss.NewStyle().Foreground(hintColor).Render(bh.Hint)

	keys := lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
		Render("0-9 type · backspace delete · enter guess · r reset · esc back to games")

	box := lipgloss.JoinVertical(lipgloss.Left,
		title, subtitle, "",
		guess, attempts, "",
		hint, "",
		keys,
	)
	framed := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccent).
		Padding(1, 2).
		Width(60).
		Render(box)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, framed)
}
