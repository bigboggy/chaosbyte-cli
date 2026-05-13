// Package theme holds the runtime palette and shared lipgloss styles. The
// palette ships with the flagship Vibespace defaults and the platform loads
// a different team's config at startup by calling Apply with their colors.
// Screens read the package-level vars; they update once at startup and
// stay stable for the lifetime of the session.
//
// The shipped defaults are a near-black ground with parchment-cream body
// text, a muted phosphor green for positive marks, and a muted gold for
// the moderator's voice. Three text colors is the working set. Anything
// else is a moment.
package theme

import "github.com/charmbracelet/lipgloss"

// Palette holds the seven runtime colors the theme exposes. Read it via
// the package-level vars; mutate the live palette via Apply.
type Palette struct {
	Bg       lipgloss.Color
	Fg       lipgloss.Color
	Muted    lipgloss.Color
	Accent   lipgloss.Color
	Accent2  lipgloss.Color
	BorderHi lipgloss.Color
	BorderLo lipgloss.Color
}

// DefaultPalette is the flagship Vibespace color set. New teams override
// any subset via Apply; whatever they leave at zero falls back to these.
func DefaultPalette() Palette {
	return Palette{
		Bg:       lipgloss.Color("#0a0a0c"),
		Fg:       lipgloss.Color("#e6dccb"),
		Muted:    lipgloss.Color("#7d7a72"),
		Accent:   lipgloss.Color("#7a9a6a"),
		Accent2:  lipgloss.Color("#b3962a"),
		BorderHi: lipgloss.Color("#25252d"),
		BorderLo: lipgloss.Color("#1a1a1f"),
	}
}

// The package-level palette vars read by every screen. Initialized to the
// flagship defaults; replaced when Apply runs.
var (
	Bg       = DefaultPalette().Bg
	Fg       = DefaultPalette().Fg
	Muted    = DefaultPalette().Muted
	Accent   = DefaultPalette().Accent
	Accent2  = DefaultPalette().Accent2
	BorderHi = DefaultPalette().BorderHi
	BorderLo = DefaultPalette().BorderLo

	// OK and Like share the green register.
	OK   = Accent
	Like = Accent

	// Warn and Fault share the gold register. The distinction lives in the
	// marker glyph rather than in a second color.
	Warn  = Accent2
	Fault = Accent2
)

// Apply replaces the runtime palette with the team-supplied colors. Must
// be called before any screen renders, typically once at startup from main
// or the SSH session handler. Zero-valued fields in the input fall through
// to the flagship defaults so a team can override a single color without
// re-declaring the rest.
func Apply(p Palette) {
	def := DefaultPalette()
	if p.Bg == "" {
		p.Bg = def.Bg
	}
	if p.Fg == "" {
		p.Fg = def.Fg
	}
	if p.Muted == "" {
		p.Muted = def.Muted
	}
	if p.Accent == "" {
		p.Accent = def.Accent
	}
	if p.Accent2 == "" {
		p.Accent2 = def.Accent2
	}
	if p.BorderHi == "" {
		p.BorderHi = def.BorderHi
	}
	if p.BorderLo == "" {
		p.BorderLo = def.BorderLo
	}
	Bg = p.Bg
	Fg = p.Fg
	Muted = p.Muted
	Accent = p.Accent
	Accent2 = p.Accent2
	BorderHi = p.BorderHi
	BorderLo = p.BorderLo
	OK = Accent
	Like = Accent
	Warn = Accent2
	Fault = Accent2
}
