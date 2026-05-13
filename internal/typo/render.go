package typo

import (
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/bchayka/gitstatus/internal/theme"
	"github.com/charmbracelet/lipgloss"
)

// longRamp is the glyph pool for Scramble + Cascade. Long enough to feel
// chaotic, short enough that a few sequential picks don't repeat obviously.
const longRamp = " .'`,:;Il!i><~+_-?][}{1)(|/\\tfjrxnuvczXYUJCLQ0OZmwqpdbkhao*#MW&8%B@$"

// Render walks a Layout, applies its AnimationState, and returns one rendered
// string per row of the Layout's natural height. Caller composes the rows
// into a final screen position (a Screen's View typically lipgloss.JoinVertical
// of these).
//
// This is the single hot path. Add new visual treatment here, not in
// primitives, primitives only mutate state.
func Render(layout *Layout, state *AnimationState, now time.Time) []string {
	if layout == nil {
		return nil
	}
	total := len(layout.Cells)
	if total == 0 {
		return make([]string, layout.Height)
	}

	revealCount := total
	if state.Reveal < 1.0 {
		revealCount = int(float64(total)*clamp01(state.Reveal) + 0.5)
	}

	// Choose effective foreground style. Tint blends on top of base.
	style := layout.BaseStyle
	if !hasStyle(style) {
		style = lipgloss.NewStyle().Foreground(theme.Fg)
	}
	if state.TintActive && state.TintBlend > 0 {
		// For now: full swap when blend > 0.5, else original. (TODO: real
		// HCL blend.)
		if state.TintBlend >= 0.5 {
			style = style.Foreground(state.Tint)
		}
	}
	if now.Before(state.PulseUntil) {
		style = style.Bold(true)
	}
	if state.Alpha < 1.0 && state.Alpha > 0 {
		// Crude alpha: at <0.5 render as muted, at >=0.5 use base.
		if state.Alpha < 0.5 {
			style = style.Foreground(theme.Muted)
		}
	}

	// Scramble / Cascade RNGs are time-bucketed so the chars flap every
	// ~80ms instead of every frame.
	var rng *rand.Rand
	if state.Scramble || state.CascadeActive {
		rng = rand.New(rand.NewSource(now.UnixNano() / 80))
	}

	dropRowOffset := int(state.DropOffset)
	dropRows := layout.Height + dropRowOffset
	if dropRows < 1 {
		dropRows = 1
	}
	out := make([]string, dropRows)

	for i, c := range layout.Cells {
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

		// Cascade settle gate: cells past the settle frontier scramble,
		// before settle they lock to target.
		ch := c.Char
		if state.CascadeActive {
			settleCol := int(float64(layout.Width)*state.CascadeSettle + 0.5)
			if c.Col >= settleCol && rng != nil {
				ch = rune(longRamp[rng.Intn(len(longRamp))])
			}
		} else if state.Scramble && rng != nil {
			ch = rune(longRamp[rng.Intn(len(longRamp))])
		}

		r := c.Row + dropRowOffset
		if r < 0 || r >= len(out) {
			continue
		}
		// Wave offset: per-row horizontal shift. Cells in the same row share
		// the offset so cells still emit in column order; neighbouring rows
		// get phase-shifted offsets so the layout reads as a snake wave
		// passing through. Negative offsets clamp to zero because the emit
		// path can only pad rightward.
		targetCol := c.Col
		if state.WaveActive {
			row := waveRowOffset(state, c.Row)
			if row > 0 {
				targetCol += row
			}
		}
		// Pad the row to targetCol before emitting the styled char.
		existing := out[r]
		visualWidth := lipgloss.Width(existing)
		if targetCol > visualWidth {
			out[r] = existing + strings.Repeat(" ", targetCol-visualWidth) + style.Render(string(ch))
		} else {
			// Row already has content past this col; append (overlap is the
			// caller's responsibility to avoid by sizing maxWidth properly).
			out[r] = existing + style.Render(string(ch))
		}
	}
	return out
}

// waveRowOffset returns the horizontal shift for one row of a layout under
// an active Wave. The wave runs through row index with a fixed step so
// adjacent rows are out of phase by ~0.7 radians, and the global WavePhase
// scrolls the wave through time.
func waveRowOffset(state *AnimationState, row int) int {
	angle := state.WavePhase + float64(row)*0.7
	return int(math.Round(state.WaveAmp * math.Sin(angle)))
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// hasStyle reports whether a lipgloss.Style has any non-default settings.
// Used as a sentinel for "use default."
func hasStyle(s lipgloss.Style) bool {
	// Heuristic: if foreground or background is set, treat as "user style."
	return s.GetForeground() != lipgloss.NoColor{} || s.GetBackground() != lipgloss.NoColor{} ||
		s.GetBold() || s.GetItalic()
}
