// Package typo is the Pretext-style content engine: text becomes Layouts
// (immutable, per-content), animation lives in AnimationState (mutable,
// per-tick), and Render is the single hot path. Replaces the field grid
// backdrop with a vocabulary of moves applied directly to UI content.
//
// Architecture mirrors Cheng Lou's Pretext (github.com/chenglou/pretext):
//   - Prepare expensive layout once and cache it
//   - Apply cheap animation state every frame
//   - Per-glyph addressable, no reflow
//
// We don't take Pretext's DOM-reflow perf win — monospace terminal doesn't
// have that problem — but we take its programming model.
package typo

import (
	"strings"
	"time"

	"github.com/bchayka/gitstatus/internal/ui"
	"github.com/charmbracelet/lipgloss"
)

// Cell is one character at a (col, row) position within a Layout. Position
// is relative to the layout origin, not the screen. The Compositor adds the
// screen offset at render time.
type Cell struct {
	Char rune
	Col  int
	Row  int
}

// Layout is the immutable shape of a piece of text — wrapped, with per-char
// addresses. Compute once via Prepare; reuse for every render frame.
type Layout struct {
	ID      string
	Cells   []Cell
	Width   int
	Height  int
	Created time.Time

	// BaseStyle is the persistent style applied to every cell before
	// AnimationState overrides. Default is theme.Fg foreground, no
	// background. Set this for kind-specific styling: ChatJoin gets a
	// green base, ChatSystem gets muted, etc.
	BaseStyle lipgloss.Style
}

// Prepare wraps text to maxWidth and returns a Layout where every char is
// addressable by (Col, Row). Newlines and word boundaries are honored via
// the existing ui.Wrap helper so behavior matches the rest of the app.
func Prepare(id, text string, maxWidth int) *Layout {
	if maxWidth < 4 {
		maxWidth = 4
	}
	wrapped := ui.Wrap(text, maxWidth)
	var cells []Cell
	maxCol := 0
	row := 0
	for _, line := range strings.Split(wrapped, "\n") {
		col := 0
		for _, r := range line {
			cells = append(cells, Cell{Char: r, Col: col, Row: row})
			col++
		}
		if col > maxCol {
			maxCol = col
		}
		row++
	}
	return &Layout{
		ID:        id,
		Cells:     cells,
		Width:     maxCol,
		Height:    row,
		Created:   time.Now(),
		BaseStyle: lipgloss.NewStyle(),
	}
}
