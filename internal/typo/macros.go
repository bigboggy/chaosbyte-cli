package typo

import (
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Macro is the intention layer over primitives. Screens and the mod call
// macros; macros decompose into timed primitive sequences. This is the
// durable API surface — primitives are an implementation detail.
//
// A Macro takes its target state + layout and elapsed time since the macro
// started; it mutates the state and returns done=true once the intention
// has fully resolved.
type Macro func(state *AnimationState, layout *Layout, elapsed time.Duration, now time.Time) (done bool)

// Greet is the macro for "this just arrived." Used for normal chat lines
// arriving in scrollback. Types the body left-to-right at 30ms/char.
//
// Per-user voice (UserStyle) can override the per-char speed and direction;
// for now this is the default.
func Greet() Macro {
	const perCharMs = 30
	return func(state *AnimationState, layout *Layout, elapsed time.Duration, _ time.Time) bool {
		Type(state, layout, elapsed, perCharMs)
		total := time.Duration(len(layout.Cells)) * time.Duration(perCharMs) * time.Millisecond
		return elapsed >= total
	}
}

// Amplify says "the room should attend to this." Tints accent + Pulse +
// (later: slight grow scale). For now: Tint blend over 200ms + Pulse window.
func Amplify(color string) Macro {
	const fadeMs = 200
	const pulseMs = 400
	const totalMs = 1200
	return func(state *AnimationState, layout *Layout, elapsed time.Duration, now time.Time) bool {
		Tint(state, elapsed, fadeMs, palette(color))
		if elapsed < time.Duration(pulseMs)*time.Millisecond {
			Pulse(state, pulseMs, now)
		}
		return elapsed >= time.Duration(totalMs)*time.Millisecond
	}
}

// Mourn is the macro for "this is leaving." Slow Wipe right-to-left over
// 800ms. Used for departing users, ended spotlights.
func Mourn() Macro {
	const totalMs = 800
	return func(state *AnimationState, layout *Layout, elapsed time.Duration, _ time.Time) bool {
		// Wipe-out: keep the cells past the wipe point hidden.
		// We invert: Reveal still goes 0->1 but acts as "how much is GONE."
		t := float64(elapsed) / float64(time.Duration(totalMs)*time.Millisecond)
		if t > 1 {
			t = 1
		}
		state.Reveal = 1.0 - t
		state.RevealFromEnd = true
		return elapsed >= time.Duration(totalMs)*time.Millisecond
	}
}

// Storm is the macro for serious moments: red Tint + Pulse + Scramble briefly.
// Used for build breaks, deploy failures.
func Storm() Macro {
	const totalMs = 1500
	const scrambleMs = 400
	return func(state *AnimationState, layout *Layout, elapsed time.Duration, now time.Time) bool {
		Tint(state, elapsed, 150, palette("warn"))
		Scramble(state, elapsed, scrambleMs)
		if elapsed < time.Duration(200)*time.Millisecond {
			Pulse(state, 600, now)
		}
		return elapsed >= time.Duration(totalMs)*time.Millisecond
	}
}

// Quiet softens a Layout: alpha fade + muted tint. Used for old chat
// decaying out, mod-quiets-the-room moments.
func Quiet() Macro {
	const totalMs = 2000
	return func(state *AnimationState, layout *Layout, elapsed time.Duration, _ time.Time) bool {
		t := float64(elapsed) / float64(time.Duration(totalMs)*time.Millisecond)
		if t > 1 {
			t = 1
		}
		state.Alpha = 1.0 - 0.5*t // settles at 50% alpha (still readable)
		return elapsed >= time.Duration(totalMs)*time.Millisecond
	}
}

// Settle is the macro for "this is deciding what it is." Scramble for 600ms,
// then Cascade-settle to the target text over 400ms. Used for mod-led
// arrivals, "loading" states, /me actions.
func Settle() Macro {
	const scrambleMs = 600
	const cascadeMs = 400
	const totalMs = scrambleMs + cascadeMs
	return func(state *AnimationState, layout *Layout, elapsed time.Duration, _ time.Time) bool {
		if elapsed < time.Duration(scrambleMs)*time.Millisecond {
			Scramble(state, elapsed, scrambleMs)
			state.Reveal = 1.0
			return false
		}
		// Cascade settle phase
		Cascade(state, layout, elapsed-time.Duration(scrambleMs)*time.Millisecond, cascadeMs)
		state.Scramble = false
		return elapsed >= time.Duration(totalMs)*time.Millisecond
	}
}

// CascadeTo is the macro for changing a layout's content in place — the
// classic split-flap board. Used for spotlight title rotation, header chip
// on screen switch, score milestone in games.
func CascadeTo() Macro {
	const totalMs = 800
	return func(state *AnimationState, layout *Layout, elapsed time.Duration, _ time.Time) bool {
		Cascade(state, layout, elapsed, totalMs)
		return elapsed >= time.Duration(totalMs)*time.Millisecond
	}
}

// palette resolves a named color to its lipgloss.Color. Strings rather than
// untyped constants so the macro API stays serializable for future LLM
// tool-use.
func palette(name string) lipgloss.Color {
	switch name {
	case "accent":
		return lipgloss.Color("#7aa2f7")
	case "accent2":
		return lipgloss.Color("#bb9af7")
	case "ok":
		return lipgloss.Color("#9ece6a")
	case "warn":
		return lipgloss.Color("#e0af68")
	case "like":
		return lipgloss.Color("#f7768e")
	case "muted":
		return lipgloss.Color("#565f89")
	}
	return lipgloss.Color("#c0caf5") // theme.Fg default
}
