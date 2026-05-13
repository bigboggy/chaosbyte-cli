// Package theme holds the runtime palette and shared lipgloss styles.
// Screens read the package-level color vars; the platform mutates the
// live palette via Apply at session start, and the /themes slash command
// swaps between named registered themes at runtime.
//
// Two palettes ship by default: "boggy" is the original Tokyo-night-style
// dark theme (cool blues and purples), "workshop" is the parchment cream
// with muted phosphor green and gold. Teams can register their own by
// adding to Themes during init, or override values inline via the team's
// .toml config.
package theme

import (
	"sort"

	"github.com/charmbracelet/lipgloss"
)

// Palette holds the runtime colors. OK / Warn / Like default to Accent /
// Accent2 / Accent if left zero, which preserves the older two-accent
// behavior for palettes that don't want a distinct status register.
type Palette struct {
	Name     string
	Bg       lipgloss.Color
	Fg       lipgloss.Color
	Muted    lipgloss.Color
	Accent   lipgloss.Color
	Accent2  lipgloss.Color
	OK       lipgloss.Color
	Warn     lipgloss.Color
	Like     lipgloss.Color
	BorderHi lipgloss.Color
	BorderLo lipgloss.Color
}

// Themes is the registry the /themes slash command lists and switches
// between. Adding a new theme is a single map entry; nothing else needs
// to change. Keep names lowercase and short.
var Themes = map[string]Palette{
	"boggy": {
		Name:     "boggy",
		Bg:       lipgloss.Color("#1a1b26"),
		Fg:       lipgloss.Color("#c0caf5"),
		Muted:    lipgloss.Color("#565f89"),
		Accent:   lipgloss.Color("#7aa2f7"),
		Accent2:  lipgloss.Color("#bb9af7"),
		OK:       lipgloss.Color("#9ece6a"),
		Warn:     lipgloss.Color("#e0af68"),
		Like:     lipgloss.Color("#f7768e"),
		BorderHi: lipgloss.Color("#7aa2f7"),
		BorderLo: lipgloss.Color("#3b4261"),
	},
	"workshop": {
		Name:     "workshop",
		Bg:       lipgloss.Color("#0a0a0c"),
		Fg:       lipgloss.Color("#e6dccb"),
		Muted:    lipgloss.Color("#7d7a72"),
		Accent:   lipgloss.Color("#7a9a6a"),
		Accent2:  lipgloss.Color("#b3962a"),
		BorderHi: lipgloss.Color("#25252d"),
		BorderLo: lipgloss.Color("#1a1a1f"),
	},
}

// DefaultPalette returns the flagship Vibespace palette ("boggy"). Used as
// the fallback when team configs leave fields empty.
func DefaultPalette() Palette {
	return Themes["boggy"]
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

	OK    = DefaultPalette().OK
	Warn  = DefaultPalette().Warn
	Like  = DefaultPalette().Like
	Fault = Accent2

	// Active is the name of the currently applied theme. The /themes
	// command reads this to mark the active entry in its listing.
	Active = DefaultPalette().Name
)

// Apply replaces the runtime palette with the supplied colors. Zero-valued
// fields fall through to the flagship defaults, so a team can override a
// single color without restating the rest. OK / Warn / Like default to
// Accent / Accent2 / Accent if not specified.
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
	if p.OK == "" {
		p.OK = p.Accent
	}
	if p.Warn == "" {
		p.Warn = p.Accent2
	}
	if p.Like == "" {
		p.Like = p.Accent
	}
	Bg = p.Bg
	Fg = p.Fg
	Muted = p.Muted
	Accent = p.Accent
	Accent2 = p.Accent2
	BorderHi = p.BorderHi
	BorderLo = p.BorderLo
	OK = p.OK
	Warn = p.Warn
	Like = p.Like
	Fault = p.Warn
	if p.Name != "" {
		Active = p.Name
	}
}

// ApplyByName looks up a named theme in the registry and applies it.
// Returns false if the name is unknown so callers can post a clear error.
func ApplyByName(name string) bool {
	p, ok := Themes[name]
	if !ok {
		return false
	}
	Apply(p)
	return true
}

// ListThemes returns the registry's theme names in stable sorted order.
func ListThemes() []string {
	out := make([]string, 0, len(Themes))
	for k := range Themes {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
