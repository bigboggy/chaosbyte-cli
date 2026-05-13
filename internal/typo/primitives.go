package typo

import (
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Primitives are pure state mutators. Each takes the current AnimationState,
// the Layout it applies to, an elapsed duration from start, and primitive-
// specific params; mutates state in place. Caller tracks elapsed.
//
// Macros (in macros.go) compose these to express intentions.

// Type reveals cells left-to-right at perCharMs per char. Sets Reveal
// proportionally; once elapsed exceeds the total duration, Reveal locks
// to 1.0 and the primitive is idempotent.
func Type(state *AnimationState, layout *Layout, elapsed time.Duration, perCharMs int) {
	if layout == nil || len(layout.Cells) == 0 {
		state.Reveal = 1.0
		return
	}
	if perCharMs <= 0 {
		perCharMs = 50
	}
	total := time.Duration(len(layout.Cells)) * time.Duration(perCharMs) * time.Millisecond
	if elapsed >= total {
		state.Reveal = 1.0
		state.RevealFromEnd = false
		return
	}
	if elapsed < 0 {
		state.Reveal = 0
		return
	}
	state.Reveal = float64(elapsed) / float64(total)
	state.RevealFromEnd = false
}

// Wipe is Type's inverse direction. Reveal still goes 0..1 but cells reveal
// from the right end first when fromRight is true.
func Wipe(state *AnimationState, layout *Layout, elapsed time.Duration, totalMs int, fromRight bool) {
	if layout == nil || totalMs <= 0 {
		state.Reveal = 1.0
		return
	}
	total := time.Duration(totalMs) * time.Millisecond
	if elapsed >= total {
		state.Reveal = 1.0
		state.RevealFromEnd = false
		return
	}
	if elapsed < 0 {
		state.Reveal = 0
		return
	}
	state.Reveal = float64(elapsed) / float64(total)
	state.RevealFromEnd = fromRight
}

// Drop applies a downward offset that decays from fromRows to 0 over the
// duration. Eased with a quadratic curve so it accelerates like gravity.
func Drop(state *AnimationState, elapsed time.Duration, totalMs int, fromRows int) {
	if totalMs <= 0 {
		state.DropOffset = 0
		return
	}
	total := time.Duration(totalMs) * time.Millisecond
	if elapsed >= total {
		state.DropOffset = 0
		return
	}
	if elapsed < 0 {
		state.DropOffset = float64(fromRows)
		return
	}
	t := float64(elapsed) / float64(total)
	// Inverse quadratic ease — fast fall, soft land
	state.DropOffset = float64(fromRows) * (1 - t*t)
}

// Tint applies a color blend that ramps in over fadeMs then holds at full.
// Set Tint via state.Tint before calling; this primitive only controls the
// blend amount.
func Tint(state *AnimationState, elapsed time.Duration, fadeMs int, color lipgloss.Color) {
	state.TintActive = true
	state.Tint = color
	if fadeMs <= 0 {
		state.TintBlend = 1.0
		return
	}
	fade := time.Duration(fadeMs) * time.Millisecond
	if elapsed >= fade {
		state.TintBlend = 1.0
		return
	}
	if elapsed < 0 {
		state.TintBlend = 0
		return
	}
	state.TintBlend = float64(elapsed) / float64(fade)
}

// Pulse sets a window during which the state should render with brief
// intensity. The renderer reads PulseUntil; the visual treatment is up to
// the renderer (typically Bold + slight brightness lift).
func Pulse(state *AnimationState, durationMs int, now time.Time) {
	if durationMs <= 0 {
		durationMs = 80
	}
	state.PulseUntil = now.Add(time.Duration(durationMs) * time.Millisecond)
}

// Scramble flips the per-cell random-glyph override for the duration.
// Returns once elapsed exceeds duration so the state can settle back.
func Scramble(state *AnimationState, elapsed time.Duration, durationMs int) {
	if durationMs <= 0 {
		state.Scramble = false
		return
	}
	state.Scramble = elapsed < time.Duration(durationMs)*time.Millisecond
}

// Cascade animates Reveal alongside a "settle from the left" effect. Cells
// past the settle point render as random long-ramp glyphs; cells before it
// have locked to the target rune. Used for split-flap-style title changes.
func Cascade(state *AnimationState, layout *Layout, elapsed time.Duration, totalMs int) {
	if layout == nil || totalMs <= 0 {
		state.Reveal = 1.0
		state.CascadeActive = false
		return
	}
	total := time.Duration(totalMs) * time.Millisecond
	state.Reveal = 1.0
	if elapsed >= total {
		state.CascadeActive = false
		state.CascadeSettle = 1.0
		return
	}
	if elapsed < 0 {
		state.CascadeActive = true
		state.CascadeSettle = 0
		return
	}
	state.CascadeActive = true
	state.CascadeSettle = float64(elapsed) / float64(total)
}
