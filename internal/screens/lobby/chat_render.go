package lobby

import (
	"strings"
	"time"

	"github.com/bchayka/gitstatus/internal/theme"
	"github.com/bchayka/gitstatus/internal/typo"
	"github.com/bchayka/gitstatus/internal/ui"
	"github.com/charmbracelet/lipgloss"
)

// arrivalWindow is how long after a message's At timestamp the entry macro
// keeps animating. After this, the message renders fully revealed.
const arrivalWindow = 3 * time.Second

// renderChatLineAnim returns just the rendered rows. Convenience wrapper
// over renderChatLineAnimDetailed for callers that don't need positional
// metadata.
func renderChatLineAnim(msg ui.ChatMessage, width int, now time.Time) []string {
	rows, _, _ := renderChatLineAnimDetailed(msg, width, now)
	return rows
}

// renderChatLineAnimDetailed returns the rendered rows plus the body text
// and the prefix's rendered width. The caller uses the body + prefix-width
// to build a typo.Layout for the body so the choreographer can fire effects
// against real chat content. This is the typo-based replacement for
// ui.RenderChatLine — builds the static prefix once, types the body in via
// the typo Greet macro for messages still inside the arrival window.
func renderChatLineAnimDetailed(msg ui.ChatMessage, width int, now time.Time) ([]string, string, int) {
	prefix, prefixWidth, bodyStyle := chatPrefix(msg)
	bodyText := bodyForKind(msg)
	bodyWidth := width - prefixWidth - 1
	if bodyWidth < 12 {
		bodyWidth = 12
	}

	layout := typo.Prepare(msgKey(msg), bodyText, bodyWidth)
	layout.BaseStyle = bodyStyle

	state := typo.NewState()
	elapsed := now.Sub(msg.At)
	macro := macroForKind(msg.Kind)
	if macro != nil && elapsed >= 0 && elapsed < arrivalWindow {
		macro(&state, layout, elapsed, now)
	} else {
		// Default: fully revealed, no animation. Normal chat just appears —
		// animation is reserved for rare meaningful moments (joins, mod
		// alerts, @mentions). The quiet baseline is the whole point.
		state.Reveal = 1.0
	}

	bodyRows := typo.Render(layout, &state, now)
	if len(bodyRows) == 0 {
		return []string{prefix}, bodyText, prefixWidth
	}

	pad := strings.Repeat(" ", prefixWidth+1)
	out := make([]string, len(bodyRows))
	for i, row := range bodyRows {
		if i == 0 {
			out[i] = prefix + " " + row
		} else {
			out[i] = pad + row
		}
	}
	return out, bodyText, prefixWidth
}

// chatPrefix returns the styled "12:34 <nick>" prefix plus its rendered
// width and the body style appropriate for the message kind.
func chatPrefix(msg ui.ChatMessage) (string, int, lipgloss.Style) {
	ts := theme.CommitTime.Render(ui.HumanizeTime(msg.At))
	var marker string
	var bodyStyle lipgloss.Style
	switch msg.Kind {
	case ui.ChatJoin:
		marker = lipgloss.NewStyle().Foreground(theme.OK).Bold(true).Render("-->")
		bodyStyle = lipgloss.NewStyle().Foreground(theme.OK).Italic(true)
	case ui.ChatSystem:
		marker = lipgloss.NewStyle().Foreground(theme.Muted).Render("*")
		bodyStyle = lipgloss.NewStyle().Foreground(theme.Muted).Italic(true)
	case ui.ChatAction:
		marker = lipgloss.NewStyle().Foreground(theme.Accent2).Bold(true).Render("*")
		bodyStyle = lipgloss.NewStyle().Foreground(theme.Accent2).Italic(true)
	default:
		nick := strings.TrimPrefix(msg.Author, "@")
		marker = lipgloss.NewStyle().Foreground(ui.NickColor(nick)).Render("<" + nick + ">")
		bodyStyle = lipgloss.NewStyle().Foreground(theme.Fg)
	}
	prefix := ts + " " + marker
	return prefix, lipgloss.Width(prefix), bodyStyle
}

// bodyForKind handles kinds where the visible body needs the Author prepended
// (joins say "@nick entered the chat", actions say "@nick shrugs", etc.).
func bodyForKind(msg ui.ChatMessage) string {
	switch msg.Kind {
	case ui.ChatJoin, ui.ChatAction:
		if msg.Author != "" {
			return msg.Author + " " + msg.Body
		}
	}
	return msg.Body
}

// macroForKind picks the entry animation for each chat kind. Animation is
// reserved for moments that genuinely matter:
//   - Joins: a person arriving is an ARRIVAL (rare, meaningful)
//   - Mod posts / /me actions: explicit expressive moments
//   - Normal chat / system: appear instantly, no animation
//
// The room is quiet by default. Adding animation to every chat line just
// recreates the "ambient noise" problem in the foreground.
func macroForKind(kind ui.ChatKind) typo.Macro {
	switch kind {
	case ui.ChatJoin:
		return typoTypeAt(60)
	case ui.ChatAction:
		return typo.Settle()
	}
	return nil
}

// typoTypeAt wraps typo.Type with a custom per-char speed.
func typoTypeAt(perCharMs int) typo.Macro {
	return func(state *typo.AnimationState, layout *typo.Layout, elapsed time.Duration, now time.Time) bool {
		typo.Type(state, layout, elapsed, perCharMs)
		total := time.Duration(len(layout.Cells)) * time.Duration(perCharMs) * time.Millisecond
		return elapsed >= total
	}
}

// msgKey is a stable layout-cache key derived from the message's content
// and timestamp. Used as Layout.ID.
func msgKey(msg ui.ChatMessage) string {
	return msg.Author + "@" + msg.At.UTC().Format(time.RFC3339Nano)
}
