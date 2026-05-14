package ui

import (
	"strings"
	"time"

	"github.com/bchayka/gitstatus/internal/theme"
	"github.com/charmbracelet/lipgloss"
)

// ChatKind classifies a chat message for rendering. Lobby and spotlight both
// use this; the ui package is the natural shared home.
type ChatKind int

const (
	ChatNormal ChatKind = iota
	ChatSystem
	ChatAction
	ChatJoin
)

type ChatMessage struct {
	Author string
	Body   string
	At     time.Time
	Kind   ChatKind
}

// RenderChatLine produces one or more visual lines for a single message, wrapping
// the body to bodyWidth and color-coding the author by nick hash for normal
// messages. The returned slice is meant to be joined with "\n".
func RenderChatLine(msg ChatMessage, width int) []string {
	ts := theme.CommitTime.Render(HumanizeTime(msg.At))
	var prefix string
	var bodyStyle lipgloss.Style
	body := msg.Body

	switch msg.Kind {
	case ChatJoin:
		prefix = lipgloss.NewStyle().Foreground(theme.OK).Bold(true).Render("-->")
		bodyStyle = lipgloss.NewStyle().Foreground(theme.OK).Italic(true)
		body = msg.Author + " " + body
	case ChatSystem:
		prefix = lipgloss.NewStyle().Foreground(theme.Muted).Render("*")
		bodyStyle = lipgloss.NewStyle().Foreground(theme.Muted).Italic(true)
	case ChatAction:
		prefix = lipgloss.NewStyle().Foreground(theme.Accent2).Bold(true).Render("*")
		bodyStyle = lipgloss.NewStyle().Foreground(theme.Accent2).Italic(true)
		body = msg.Author + " " + body
	default:
		nick := strings.TrimPrefix(msg.Author, "@")
		prefix = lipgloss.NewStyle().Foreground(NickColor(nick)).Render("<" + nick + ">")
		bodyStyle = lipgloss.NewStyle().Foreground(theme.Fg)
	}

	header := ts + " " + prefix + " "
	headerW := lipgloss.Width(header)
	bodyW := width - headerW - 2
	if bodyW < 12 {
		bodyW = 12
	}
	wrapped := Wrap(body, bodyW)
	parts := strings.Split(wrapped, "\n")

	out := make([]string, 0, len(parts))
	out = append(out, header+bodyStyle.Render(parts[0]))
	pad := strings.Repeat(" ", headerW)
	for _, p := range parts[1:] {
		out = append(out, pad+bodyStyle.Render(p))
	}
	return out
}

// NickColor maps a nickname to a deterministic palette color so chat lines
// have visual continuity across the scrollback.
func NickColor(nick string) lipgloss.Color {
	palette := []lipgloss.Color{
		theme.Accent, theme.Accent2, theme.OK, theme.Warn, theme.Like,
	}
	h := 0
	for _, c := range nick {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return palette[h%len(palette)]
}

