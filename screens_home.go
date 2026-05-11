package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type homeTile struct {
	key   string
	title string
	hint  string
	to    screen
}

func (m model) homeTiles() []homeTile {
	chatUnread := 0
	for _, c := range m.channels {
		chatUnread += c.Unread
	}
	spotIdx, secs := m.spotlightRotation()
	spotHint := "live · 5:00"
	if spotIdx < len(m.spotlights) {
		spotHint = fmt.Sprintf("live · %s", mmss(secs))
	}
	totalCommits := 0
	for _, b := range m.branches {
		totalCommits += len(b.Commits)
	}

	return []homeTile{
		{"1", "chat", fmt.Sprintf("%d channels · %d unread", len(m.channels), chatUnread), screenChat},
		{"2", "news", fmt.Sprintf("%d stories · HN + more", len(m.newsItems)), screenNews},
		{"3", "resources", fmt.Sprintf("%d skills · %d repos", len(m.skillsTrending), len(m.repos)), screenResources},
		{"4", "spotlight", spotHint, screenSpotlight},
		{"5", "games", fmt.Sprintf("%d mini-games", len(m.games)), screenGames},
		{"6", "discussions", fmt.Sprintf("%d branches · %d commits", len(m.branches), totalCommits), screenDiscussions},
	}
}

func (m model) updateHome(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	tiles := m.homeTiles()
	if len(tiles) == 0 {
		return m, nil
	}
	switch msg.String() {
	case "j", "down":
		if m.homeIdx+homeCols < len(tiles) {
			m.homeIdx += homeCols
		}
		return m, nil
	case "k", "up":
		if m.homeIdx-homeCols >= 0 {
			m.homeIdx -= homeCols
		}
		return m, nil
	case "h", "left":
		if m.homeIdx%homeCols > 0 {
			m.homeIdx--
		}
		return m, nil
	case "l", "right":
		if m.homeIdx%homeCols < homeCols-1 && m.homeIdx+1 < len(tiles) {
			m.homeIdx++
		}
		return m, nil
	case "enter", " ":
		t := tiles[m.homeIdx]
		return m.jumpTo(t.to), nil
	}
	return m, nil
}

func (m model) renderHome(width, height int) string {
	tiles := m.homeTiles()

	logo := m.renderLogo()
	tagline := lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
		Render("an all-in-one place for devs and vibe coders")

	chatUnread := 0
	for _, c := range m.channels {
		chatUnread += c.Unread
	}
	_, secs := m.spotlightRotation()
	statusLine := lipgloss.NewStyle().Foreground(colorOk).Render(
		fmt.Sprintf("• %d unread in chat   • spotlight rotates in %s   • %d branches active",
			chatUnread, mmss(secs), len(m.branches)))

	const tileW = 26
	const tileH = 6
	rows := []string{}
	row := []string{}
	for i, t := range tiles {
		row = append(row, renderHomeTile(t, i == m.homeIdx, tileW, tileH))
		if (i+1)%homeCols == 0 {
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, row...))
			row = []string{}
		}
	}
	if len(row) > 0 {
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, row...))
	}
	grid := strings.Join(rows, "\n")

	instructions := renderHomeInstructions()

	stacked := lipgloss.JoinVertical(lipgloss.Center,
		logo,
		"",
		tagline,
		"",
		statusLine,
		"",
		grid,
		"",
		instructions,
	)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, stacked)
}

func renderHomeTile(t homeTile, focused bool, w, h int) string {
	keyStyle := lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	titleStyle := lipgloss.NewStyle().Foreground(colorFg).Bold(true)
	hintStyle := lipgloss.NewStyle().Foreground(colorMuted)

	if focused {
		keyStyle = keyStyle.Foreground(colorBg).Background(colorAccent)
		titleStyle = titleStyle.Foreground(colorAccent2)
	}

	keyTag := keyStyle.Padding(0, 1).Render(t.key)
	title := titleStyle.Render(t.title)
	hint := hintStyle.Render(t.hint)

	body := lipgloss.JoinVertical(lipgloss.Left,
		keyTag+"  "+title,
		"",
		hint,
	)

	border := colorBorderLo
	if focused {
		border = colorAccent
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(0, 1).
		Width(w).
		Height(h).
		Render(body)
}

func renderHomeInstructions() string {
	mk := func(k, d string) string {
		return helpKeyStyle.Render(k) + " " + helpDescStyle.Render(d)
	}
	row1 := strings.Join([]string{
		mk("1-6", "jump to area"),
		mk("hjkl", "move"),
		mk("enter", "open"),
	}, "   ·   ")
	row2 := strings.Join([]string{
		mk("esc", "back to home"),
		mk("H", "home"),
		mk("q", "quit"),
	}, "   ·   ")
	return lipgloss.JoinVertical(lipgloss.Center, row1, row2)
}
