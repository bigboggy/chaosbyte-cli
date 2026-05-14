// Package theme defines a Theme value (a named color palette) and a Styles
// value that pairs a Theme with a per-session *lipgloss.Renderer.
//
// Screens hold a *Styles and build all visuals through it. Swapping the theme
// at runtime is a single field write — every subsequent render picks up the
// new colors. Per-session renderers mean truecolor / 256 / 16-color clients
// all get appropriate output without a process-wide override.
package theme

import (
	"sort"

	"github.com/charmbracelet/lipgloss"
)

// Theme is a named color palette. Colors are stored as lipgloss.Color so they
// flow through the per-session renderer's downgrade logic unchanged.
type Theme struct {
	ID          string // lowercase slug used by /theme <id>
	DisplayName string

	Bg, Fg, Muted      lipgloss.Color
	Accent, Accent2    lipgloss.Color
	OK, Warn, Like     lipgloss.Color
	BorderLo, BorderHi lipgloss.Color
}

// LogoGradient returns the per-row colors used by the intro logo. Three
// accents repeated so a 6-row logo gets two rows per color.
func (t Theme) LogoGradient() []lipgloss.Color {
	return []lipgloss.Color{t.Accent, t.Accent, t.Accent2, t.Accent2, t.Like, t.Like}
}

// registry holds every known theme keyed by ID.
var registry = map[string]Theme{}

func register(t Theme) {
	registry[t.ID] = t
}

// Get returns the theme with the given ID; ok=false if unknown.
func Get(id string) (Theme, bool) {
	t, ok := registry[id]
	return t, ok
}

// IDs returns the registered theme IDs in sorted order. Used by /theme to list
// what's available.
func IDs() []string {
	out := make([]string, 0, len(registry))
	for id := range registry {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// DefaultID is the theme new sessions start on.
const DefaultID = "tokyonight"

// Default returns the default Theme.
func Default() Theme {
	t, _ := Get(DefaultID)
	return t
}

func init() {
	register(Theme{
		ID:          "tokyonight",
		DisplayName: "Tokyo Night",
		Bg:          "#1a1b26",
		Fg:          "#c0caf5",
		Muted:       "#565f89",
		Accent:      "#7aa2f7",
		Accent2:     "#bb9af7",
		OK:          "#9ece6a",
		Warn:        "#e0af68",
		Like:        "#f7768e",
		BorderLo:    "#3b4261",
		BorderHi:    "#7aa2f7",
	})
	register(Theme{
		ID:          "catppuccin",
		DisplayName: "Catppuccin Mocha",
		Bg:          "#1e1e2e",
		Fg:          "#cdd6f4",
		Muted:       "#6c7086",
		Accent:      "#89b4fa",
		Accent2:     "#cba6f7",
		OK:          "#a6e3a1",
		Warn:        "#f9e2af",
		Like:        "#f5c2e7",
		BorderLo:    "#45475a",
		BorderHi:    "#89b4fa",
	})
	register(Theme{
		ID:          "dracula",
		DisplayName: "Dracula",
		Bg:          "#282a36",
		Fg:          "#f8f8f2",
		Muted:       "#6272a4",
		Accent:      "#8be9fd",
		Accent2:     "#bd93f9",
		OK:          "#50fa7b",
		Warn:        "#f1fa8c",
		Like:        "#ff79c6",
		BorderLo:    "#44475a",
		BorderHi:    "#bd93f9",
	})
	register(Theme{
		ID:          "gruvbox",
		DisplayName: "Gruvbox Dark",
		Bg:          "#282828",
		Fg:          "#ebdbb2",
		Muted:       "#928374",
		Accent:      "#83a598",
		Accent2:     "#d3869b",
		OK:          "#b8bb26",
		Warn:        "#fabd2f",
		Like:        "#fb4934",
		BorderLo:    "#3c3836",
		BorderHi:    "#fabd2f",
	})
	register(Theme{
		ID:          "nord",
		DisplayName: "Nord",
		Bg:          "#2e3440",
		Fg:          "#eceff4",
		Muted:       "#4c566a",
		Accent:      "#88c0d0",
		Accent2:     "#b48ead",
		OK:          "#a3be8c",
		Warn:        "#ebcb8b",
		Like:        "#bf616a",
		BorderLo:    "#3b4252",
		BorderHi:    "#88c0d0",
	})
}
