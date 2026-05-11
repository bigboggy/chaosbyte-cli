// Package ui provides layout, text, and chat-rendering primitives that are
// shared across screens. It depends on theme but never on a specific screen
// package, so importing it never risks a cycle.
package ui

import (
	"strings"

	"github.com/bchayka/gitstatus/internal/theme"
	"github.com/charmbracelet/lipgloss"
)

const (
	MinWidth  = 80
	MinHeight = 22
)

// FeedShellWidth returns the centered content width budget. Mirrors the popup
// sizing so a screen's main column lines up visually with overlays.
func FeedShellWidth(termW int) int {
	w := termW * 80 / 100
	if w > 100 {
		w = 100
	}
	if w < 60 {
		w = 60
	}
	if w > termW-2 {
		w = termW - 2
	}
	return w
}

// PopupSize returns reasonable dimensions for a modal overlay.
func PopupSize(termW, termH int) (w, h int) {
	w = termW * 80 / 100
	if w > 90 {
		w = 90
	}
	if w < 50 {
		w = 50
	}
	h = 14
	if h > termH-2 {
		h = termH - 2
	}
	if h < 8 {
		h = 8
	}
	return
}

// PadToHeight pads s with trailing newlines to fill h rows, or clips it if it
// already exceeds h. Use after composing a body block so its outer container's
// height stays stable across frames.
func PadToHeight(s string, h int) string {
	if h <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	if len(lines) >= h {
		return strings.Join(lines[:h], "\n")
	}
	return s + strings.Repeat("\n", h-len(lines))
}

// Divider returns a horizontal rule of the given width in the low-contrast
// border color.
func Divider(width int) string {
	return lipgloss.NewStyle().Foreground(theme.BorderLo).Render(strings.Repeat("─", width))
}
