package games

import (
	"fmt"
	"time"

	"github.com/bchayka/gitstatus/internal/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type bugHunterState struct {
	target    int
	guess     int
	attempts  int
	hint      string
	done      bool
	startedAt time.Time
}

func newBugHunter() bugHunterState {
	t := int(time.Now().UnixNano() % 100)
	if t < 1 {
		t = 1
	}
	return bugHunterState{
		target:    t,
		hint:      "i'm thinking of a line number between 1 and 100. press 0-9 to dial, enter to guess.",
		startedAt: time.Now(),
	}
}

func (s *Screen) updateBugHunter(km tea.KeyMsg) tea.Cmd {
	bh := &s.bug
	if bh.done {
		switch km.String() {
		case "enter", " ", "r":
			s.bug = newBugHunter()
		}
		return nil
	}
	key := km.String()
	if len(key) == 1 && key >= "0" && key <= "9" {
		d := int(key[0] - '0')
		if bh.guess == 0 {
			bh.guess = d
		} else if bh.guess < 100 {
			bh.guess = bh.guess*10 + d
			if bh.guess > 999 {
				bh.guess = bh.guess % 1000
			}
		}
		return nil
	}
	switch key {
	case "backspace":
		bh.guess /= 10
	case "enter":
		if bh.guess < 1 || bh.guess > 100 {
			bh.hint = "out of bounds. 1-100, friend."
			return nil
		}
		bh.attempts++
		if bh.guess == bh.target {
			bh.done = true
			elapsed := time.Since(bh.startedAt).Truncate(time.Second)
			bh.hint = fmt.Sprintf("FOUND IT. line %d, in %d attempts, in %s. ship it.", bh.target, bh.attempts, elapsed)
		} else if bh.guess < bh.target {
			bh.hint = fmt.Sprintf("nope, %d is too early. the bug is deeper.", bh.guess)
		} else {
			bh.hint = fmt.Sprintf("nope, %d is too late. the bug is earlier.", bh.guess)
		}
		bh.guess = 0
	case "r":
		s.bug = newBugHunter()
	}
	return nil
}

func (s *Screen) renderBugHunter(width, height int) string {
	bh := s.bug
	title := theme.Title.Render("bug hunter")
	subtitle := lipgloss.NewStyle().Foreground(theme.Muted).Italic(true).
		Render("guess the line number where the bug is hiding (1-100)")

	guess := lipgloss.NewStyle().Foreground(theme.Accent2).Bold(true).
		Render(fmt.Sprintf("your guess: %d", bh.guess))
	attempts := lipgloss.NewStyle().Foreground(theme.Muted).
		Render(fmt.Sprintf("attempts: %d", bh.attempts))

	hintColor := theme.Warn
	if bh.done {
		hintColor = theme.OK
	}
	hint := lipgloss.NewStyle().Foreground(hintColor).Render(bh.hint)

	keys := lipgloss.NewStyle().Foreground(theme.Muted).Italic(true).
		Render("0-9 type · backspace delete · enter guess · r reset · esc back to games")

	box := lipgloss.JoinVertical(lipgloss.Left,
		title, subtitle, "",
		guess, attempts, "",
		hint, "",
		keys,
	)
	framed := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Accent).
		Padding(1, 2).
		Width(60).
		Render(box)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, framed)
}
