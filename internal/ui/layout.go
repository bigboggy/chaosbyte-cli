// Package ui provides layout, text, and chat-rendering primitives that are
// shared across screens. It depends on theme but never on a specific screen
// package, so importing it never risks a cycle.
package ui

import (
	"strings"

	"github.com/bigboggy/vibespace/internal/theme"
)

const (
	// MinWidth/MinHeight are the soft floors below which we show a
	// "terminal too small" error instead of rendering. Kept permissive so
	// resized panes and small terminals still work.
	MinWidth  = 40
	MinHeight = 10
)

// FeedShellWidth returns the content width budget for a screen body. We use
// almost the full terminal width (minus a 2-cell right gutter) so content
// is left-aligned and uses the available horizontal space — no artificial
// readability cap, no centered column with big margins.
func FeedShellWidth(termW int) int {
	if termW <= 2 {
		return termW
	}
	return termW - 2
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
func Divider(st *theme.Styles, width int) string {
	return st.NewStyle().Foreground(st.BorderLo).Render(strings.Repeat("─", width))
}
