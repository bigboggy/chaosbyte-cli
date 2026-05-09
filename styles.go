package main

import "github.com/charmbracelet/lipgloss"

var (
	colorBg       = lipgloss.Color("#1a1b26")
	colorFg       = lipgloss.Color("#c0caf5")
	colorMuted    = lipgloss.Color("#565f89")
	colorAccent   = lipgloss.Color("#7aa2f7")
	colorAccent2  = lipgloss.Color("#bb9af7")
	colorOk       = lipgloss.Color("#9ece6a")
	colorWarn     = lipgloss.Color("#e0af68")
	colorLike     = lipgloss.Color("#f7768e")
	colorBorderHi = lipgloss.Color("#7aa2f7")
	colorBorderLo = lipgloss.Color("#3b4261")
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent2).
			Padding(0, 1)

	statusStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 1)

	branchItemStyle = lipgloss.NewStyle().
			Foreground(colorFg).
			Padding(0, 1)

	branchItemSelStyle = lipgloss.NewStyle().
				Foreground(colorBg).
				Background(colorAccent).
				Bold(true).
				Padding(0, 1)

	commitSHAStyle = lipgloss.NewStyle().
			Foreground(colorWarn).
			Bold(true)

	commitAuthorStyle = lipgloss.NewStyle().
				Foreground(colorOk)

	commitTimeStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)

	commitMsgStyle = lipgloss.NewStyle().
			Foreground(colorFg)

	likeStyle = lipgloss.NewStyle().
			Foreground(colorLike)

	likedStyle = lipgloss.NewStyle().
			Foreground(colorLike).
			Bold(true)

	commentCountStyle = lipgloss.NewStyle().
				Foreground(colorAccent)

	commentAuthorStyle = lipgloss.NewStyle().
				Foreground(colorAccent2).
				Bold(true)

	commentBodyStyle = lipgloss.NewStyle().
				Foreground(colorFg)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	promptStyle = lipgloss.NewStyle().
			Foreground(colorOk).
			Bold(true)
)

func paneStyle(focused bool) lipgloss.Style {
	border := colorBorderLo
	if focused {
		border = colorBorderHi
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(0, 1)
}
