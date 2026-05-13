package typo

import (
	"time"

	"github.com/charmbracelet/lipgloss"
)

// AnimationState is the per-tick mutable state for one Layout. Primitives
// mutate this; the renderer reads it. Default state is "show this layout
// as-is", fully revealed, no tint, no drop, full alpha.
type AnimationState struct {
	// BornAt is when this layout was first staged. Primitives use it to
	// derive elapsed from a current Tick time.
	BornAt time.Time

	// Reveal is the fraction of cells visible, 0..1. Driven by Type and
	// Wipe. RevealFromEnd inverts the direction (right-to-left wipe).
	Reveal        float64
	RevealFromEnd bool

	// DropOffset shifts all cells down by this many rows (can be fractional).
	// Negative offsets land cells above their natural row. Driven by Drop.
	DropOffset float64

	// Alpha is the opacity 0..1. Used by fade primitives.
	Alpha float64

	// Tint overrides the foreground color when TintActive is true. TintBlend
	// 0..1 lets a Tint primitive interpolate in.
	Tint       lipgloss.Color
	TintActive bool
	TintBlend  float64

	// Scramble overrides each rune with a random glyph from the long ramp.
	// Used for countdown digits in last 5s, /me actions, mod confusion.
	Scramble bool

	// PulseUntil holds a future time; while in the window cells render with
	// a brief intensity bump. Set by the Pulse primitive.
	PulseUntil time.Time

	// Cascade renders revealed cells with a per-frame random glyph that
	// settles to the target rune, split-flap display behavior. Driven by
	// Cascade primitive over its window. Settle fraction 0..1 sets how many
	// cells from the left have already locked.
	CascadeActive bool
	CascadeSettle float64

	// Wave shifts each rendered row horizontally by an amount that follows
	// a sine wave through the row index. Cells in a single row share the
	// same offset (so the existing left-to-right emit logic in Render
	// still works), but neighbouring rows get phase-shifted offsets so
	// the layout looks like it's breathing through a snake wave. Driven
	// by blitz.Paint during a /blitz round.
	WaveActive bool
	WavePhase  float64
	WaveAmp    float64
}

// NewState returns a default "show as-is" state. Layouts that don't animate
// just use this; primitives create variants.
func NewState() AnimationState {
	return AnimationState{
		BornAt:        time.Now(),
		Reveal:        1.0,
		RevealFromEnd: false,
		Alpha:         1.0,
	}
}

// Done reports whether every active animation in this state has resolved ,
// caller can drop the state if true.
func (s *AnimationState) Done(now time.Time) bool {
	if s.Reveal < 1.0 {
		return false
	}
	if s.DropOffset != 0 {
		return false
	}
	if s.Scramble {
		return false
	}
	if s.CascadeActive {
		return false
	}
	if now.Before(s.PulseUntil) {
		return false
	}
	if s.TintActive && s.TintBlend > 0 && s.TintBlend < 1 {
		return false
	}
	if s.WaveActive {
		return false
	}
	return true
}
