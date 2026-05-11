package main

import (
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type introTickMsg time.Time

const (
	introBootEnd   = 400  // boot lines type out
	introBuildEnd  = 1000 // logo builds line by line
	introHoldEnd   = 1800 // logo holds with tagline
	introShrinkEnd = 2100 // collapses to plain "CHAOSBYTE"
	introByteEnd   = 2500 // morphs through binary → "byte"
	introBlockEnd  = 2700 // single block
	introFadeEnd   = 2900 // blank
)

var introBootLines = []string{
	"chaosbyte boot v0.1.0",
	"",
	"[ok] kernel              loaded",
	"[ok] mesh.chaosbyte.dev  online",
	"[ok] vibes               synced",
	"[ok] tui driver          initialized",
	"[ok] #lobby              ready",
}

func introTickCmd() tea.Cmd {
	return tea.Tick(33*time.Millisecond, func(t time.Time) tea.Msg {
		return introTickMsg(t)
	})
}

func (m model) updateIntro(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	}
	// any other key skips the intro
	return m.finishIntro(), nil
}

func (m model) finishIntro() model {
	m.screen = screenLobby
	m.lobbyJoin()
	m.lobbyInput.Focus()
	return m
}

func (m model) renderIntro(width, height int) string {
	elapsed := time.Since(m.introStart)
	ms := int(elapsed.Milliseconds())

	var content string
	switch {
	case ms < introBootEnd:
		content = renderIntroBoot(ms)
	case ms < introBuildEnd:
		content = renderIntroBuild(ms - introBootEnd)
	case ms < introHoldEnd:
		content = renderIntroHold(ms - introBuildEnd)
	case ms < introShrinkEnd:
		content = renderIntroShrink(ms - introHoldEnd)
	case ms < introByteEnd:
		content = renderIntroByte(ms - introShrinkEnd)
	case ms < introBlockEnd:
		content = renderIntroBlock(ms - introByteEnd)
	case ms < introFadeEnd:
		content = ""
	default:
		content = ""
	}

	skip := lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
		Render("press any key to skip")
	frame := lipgloss.JoinVertical(lipgloss.Center, content, "", "", skip)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, frame)
}

func renderIntroBoot(ms int) string {
	revealed := ms / 55
	if revealed > len(introBootLines) {
		revealed = len(introBootLines)
	}
	var out []string
	for i := 0; i < revealed; i++ {
		line := introBootLines[i]
		style := lipgloss.NewStyle().Foreground(colorOk)
		if strings.HasPrefix(line, "chaosbyte") {
			style = lipgloss.NewStyle().Foreground(colorAccent2).Bold(true)
		}
		out = append(out, style.Render(line))
	}
	// blinking cursor on the last line while typing
	if revealed < len(introBootLines) && ms%500 < 250 {
		cursor := lipgloss.NewStyle().Foreground(colorAccent).Render("█")
		out = append(out, cursor)
	}
	return strings.Join(out, "\n")
}

func renderIntroBuild(ms int) string {
	revealed := ms/100 + 1
	if revealed > len(logoLines) {
		revealed = len(logoLines)
	}
	gradient := []lipgloss.Color{
		colorAccent, colorAccent,
		colorAccent2, colorAccent2,
		colorLike, colorLike,
	}
	var out []string
	for i := 0; i < revealed; i++ {
		out = append(out, lipgloss.NewStyle().
			Foreground(gradient[i%len(gradient)]).
			Bold(true).
			Render(logoLines[i]))
	}
	return strings.Join(out, "\n")
}

func renderIntroHold(ms int) string {
	gradient := []lipgloss.Color{
		colorAccent, colorAccent,
		colorAccent2, colorAccent2,
		colorLike, colorLike,
	}
	var out []string
	for i, line := range logoLines {
		out = append(out, lipgloss.NewStyle().
			Foreground(gradient[i%len(gradient)]).
			Bold(true).
			Render(line))
	}
	logo := strings.Join(out, "\n")

	// pulse the tagline color
	pulse := math.Abs(math.Sin(float64(ms) / 180.0))
	color := colorAccent
	if pulse > 0.5 {
		color = colorAccent2
	}
	tagline := lipgloss.NewStyle().Foreground(color).Italic(true).
		Render("an all-in-one place for devs and vibe coders")
	dots := lipgloss.NewStyle().Foreground(colorMuted).
		Render(strings.Repeat(".", (ms/200)%4))
	connecting := lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
		Render("connecting to #lobby") + dots
	return lipgloss.JoinVertical(lipgloss.Center, logo, "", tagline, "", connecting)
}

func renderIntroShrink(ms int) string {
	// collapse: show fewer logo lines, then plain "CHAOSBYTE"
	progress := float64(ms) / float64(introShrinkEnd-introHoldEnd)
	if progress < 0.4 {
		// outer rows fade first: keep middle rows only
		mid := []string{
			logoLines[1], logoLines[2], logoLines[3], logoLines[4],
		}
		return lipgloss.NewStyle().Foreground(colorAccent2).Bold(true).
			Render(strings.Join(mid, "\n"))
	}
	if progress < 0.7 {
		mid := []string{logoLines[2], logoLines[3]}
		return lipgloss.NewStyle().Foreground(colorAccent2).Bold(true).
			Render(strings.Join(mid, "\n"))
	}
	return lipgloss.NewStyle().Foreground(colorAccent2).Bold(true).
		Render("C H A O S B Y T E")
}

func renderIntroByte(ms int) string {
	progress := float64(ms) / float64(introByteEnd-introShrinkEnd)
	if progress < 0.3 {
		return lipgloss.NewStyle().Foreground(colorAccent).Bold(true).
			Render("01000010")
	}
	if progress < 0.7 {
		return lipgloss.NewStyle().Foreground(colorAccent).Bold(true).
			Render("byte")
	}
	return lipgloss.NewStyle().Foreground(colorAccent).Bold(true).
		Render("b")
}

func renderIntroBlock(ms int) string {
	// flicker the block out
	if ms%140 < 70 {
		return lipgloss.NewStyle().Foreground(colorAccent).Render("▪")
	}
	return lipgloss.NewStyle().Foreground(colorMuted).Render("·")
}
