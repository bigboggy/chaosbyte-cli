package typo

import (
	"math"
	"strings"
	"time"

	"github.com/bchayka/gitstatus/internal/theme"
	"github.com/charmbracelet/lipgloss"
)

// Compositor builds a final rendered string by drawing Layouts (with their
// AnimationStates) and CellTransforms (from the Choreographer) into a fixed-
// size 2D grid. Use this when a Screen needs the full Pretext pipeline —
// multiple Layouts composited and event Effects displacing cells across
// their natural bounds.
//
// Typical flow per frame:
//
//	c := NewCompositor(width, height)
//	transforms := choreographer.Tick(now)
//	borrowed := IndexTransforms(transforms)  // sourceID -> cell indices
//	for each message:
//	    c.DrawLayout(msgLayout, msgState, borrowed[msgLayout.ID], x, y, now)
//	c.DrawTransforms(transforms, sourceOrigins, now)
//	output := c.Render()
type Compositor struct {
	width, height int
	cells         [][]renderedCell
}

type renderedCell struct {
	char  rune
	style lipgloss.Style
	set   bool
}

// LayoutOrigin pairs a Layout with the screen position of its (0, 0) cell.
// Used by DrawTransforms to compute displaced cell positions.
type LayoutOrigin struct {
	Layout *Layout
	X, Y   int
}

// NewCompositor returns a blank compositor sized (width, height). Cells
// outside this size are clipped silently.
func NewCompositor(width, height int) *Compositor {
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	cells := make([][]renderedCell, height)
	for i := range cells {
		cells[i] = make([]renderedCell, width)
	}
	return &Compositor{width: width, height: height, cells: cells}
}

// DrawLayout writes the layout's revealed cells into the compositor grid at
// (originX + cell.Col, originY + cell.Row), respecting the AnimationState.
// Cells whose indices appear in `borrowed` are skipped — those get drawn
// later by DrawTransforms at their displaced positions.
func (c *Compositor) DrawLayout(
	layout *Layout,
	state *AnimationState,
	borrowed map[int]bool,
	originX, originY int,
	now time.Time,
) {
	if layout == nil {
		return
	}
	total := len(layout.Cells)
	revealCount := total
	if state.Reveal < 1.0 {
		revealCount = int(float64(total)*state.Reveal + 0.5)
	}

	style := effectiveStyle(layout, state, now)

	for i, cell := range layout.Cells {
		// Reveal gate
		visible := false
		if state.RevealFromEnd {
			visible = i >= (total - revealCount)
		} else {
			visible = i < revealCount
		}
		if !visible {
			continue
		}
		// Skip cells the Choreographer is animating elsewhere on this frame.
		if borrowed[i] {
			continue
		}
		c.put(originX+cell.Col, originY+cell.Row, cell.Char, style)
	}
}

// DrawTransforms writes each transform's cell at its displaced screen
// position. origins lets it look up the source cell's rune and natural
// position by SourceLayoutID. Transforms whose layout isn't in origins are
// silently dropped (the source went out of scope).
func (c *Compositor) DrawTransforms(
	transforms []CellTransform,
	origins map[string]LayoutOrigin,
	now time.Time,
) {
	for _, tr := range transforms {
		origin, ok := origins[tr.SourceLayoutID]
		if !ok || origin.Layout == nil {
			continue
		}
		if tr.SourceCellIdx < 0 || tr.SourceCellIdx >= len(origin.Layout.Cells) {
			continue
		}
		cell := origin.Layout.Cells[tr.SourceCellIdx]
		x := origin.X + cell.Col + int(math.Round(tr.OffsetX))
		y := origin.Y + cell.Row + int(math.Round(tr.OffsetY))

		// Borrowed cells render with the layout's effective style + a bold
		// emphasis so they read as "in transit." Could be refined per
		// EffectKind in a later pass.
		style := lipgloss.NewStyle().Foreground(theme.Fg).Bold(true)
		if hasStyle(origin.Layout.BaseStyle) {
			style = origin.Layout.BaseStyle.Bold(true)
		}
		c.put(x, y, cell.Char, style)
	}
}

// Render walks the grid row by row and builds the final styled string. Empty
// cells emit a space (preserves alignment). Adjacent cells with identical
// styles are coalesced into one Render call to keep ANSI output compact.
func (c *Compositor) Render() string {
	var b strings.Builder
	emptyStyle := lipgloss.NewStyle()
	for y := 0; y < c.height; y++ {
		x := 0
		for x < c.width {
			cell := c.cells[y][x]
			if !cell.set {
				// Trim a run of unset cells into one space block
				runStart := x
				for x < c.width && !c.cells[y][x].set {
					x++
				}
				b.WriteString(emptyStyle.Render(strings.Repeat(" ", x-runStart)))
				continue
			}
			// Coalesce a run of same-style cells
			runStart := x
			currentStyle := cell.style
			var chars []rune
			for x < c.width && c.cells[y][x].set && stylesEqual(c.cells[y][x].style, currentStyle) {
				chars = append(chars, c.cells[y][x].char)
				x++
			}
			b.WriteString(currentStyle.Render(string(chars)))
			_ = runStart
		}
		if y < c.height-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// IndexTransforms groups CellTransforms by SourceLayoutID and returns a map
// of layoutID -> set-of-cell-indices. Used by callers to quickly look up
// which cells of a layout are currently borrowed.
func IndexTransforms(transforms []CellTransform) map[string]map[int]bool {
	out := map[string]map[int]bool{}
	for _, tr := range transforms {
		set := out[tr.SourceLayoutID]
		if set == nil {
			set = map[int]bool{}
			out[tr.SourceLayoutID] = set
		}
		set[tr.SourceCellIdx] = true
	}
	return out
}

// effectiveStyle resolves a Layout's BaseStyle with its current
// AnimationState applied: tint, alpha, pulse, etc.
func effectiveStyle(layout *Layout, state *AnimationState, now time.Time) lipgloss.Style {
	style := layout.BaseStyle
	if !hasStyle(style) {
		style = lipgloss.NewStyle().Foreground(theme.Fg)
	}
	if state.TintActive && state.TintBlend >= 0.5 {
		style = style.Foreground(state.Tint)
	}
	if state.Alpha > 0 && state.Alpha < 0.5 {
		style = style.Foreground(theme.Muted)
	}
	if now.Before(state.PulseUntil) {
		style = style.Bold(true)
	}
	return style
}

func (c *Compositor) put(x, y int, ch rune, style lipgloss.Style) {
	if y < 0 || y >= c.height || x < 0 || x >= c.width {
		return
	}
	c.cells[y][x] = renderedCell{char: ch, style: style, set: true}
}

// stylesEqual is a coarse equality check for two lipgloss.Style values.
// Avoids reflecting on every field; checks the visible ones.
func stylesEqual(a, b lipgloss.Style) bool {
	return a.GetForeground() == b.GetForeground() &&
		a.GetBackground() == b.GetBackground() &&
		a.GetBold() == b.GetBold() &&
		a.GetItalic() == b.GetItalic() &&
		a.GetUnderline() == b.GetUnderline()
}
