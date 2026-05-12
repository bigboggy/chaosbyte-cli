package ui

import (
	"fmt"
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

// ChatTag is a moderator annotation that attaches to a chat message after
// publish. The marker glyph appears in the margin and identifies what the
// moderator saw in the post. Tags carry a born timestamp so the renderer
// can animate their arrival; expired tags get filtered out at the source.
type ChatTag struct {
	Kind   string    // "question", "url", "code", "alert"
	Marker rune      // glyph drawn in the margin, defaults to ✦
	Reason string    // human-readable why, surfaced on focus
	BornAt time.Time // when the moderator attached this tag
}

type ChatMessage struct {
	Author string
	Body   string
	At     time.Time
	Kind   ChatKind

	// Tags are moderator annotations attached after publish. The room's
	// renderer reads these to place the margin glyph and to animate its
	// arrival when the BornAt is recent.
	Tags []ChatTag
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

// FormatMembers returns a "%d online · %d members" style string for headers.
func FormatMembers(online, members int) string {
	return fmt.Sprintf("%d online · %d members", online, members)
}
