package theme

import "github.com/charmbracelet/lipgloss"

// Styles pairs a per-session lipgloss.Renderer with a Theme. Embedding both
// means call sites can use the renderer's NewStyle alongside theme colors as
// promoted fields: s.NewStyle().Foreground(s.Accent).
//
// Theme is mutable — /theme replaces the embedded value via SetTheme. Since
// screens hold a *Styles, the swap is visible app-wide on the next render.
type Styles struct {
	*lipgloss.Renderer
	Theme
}

// New pairs a renderer with a theme. The renderer carries the client's color
// profile so styles built via this Styles will downgrade gracefully on
// terminals without truecolor.
func New(r *lipgloss.Renderer, t Theme) *Styles {
	if r == nil {
		r = lipgloss.DefaultRenderer()
	}
	return &Styles{Renderer: r, Theme: t}
}

// SetTheme swaps the active palette. The renderer is preserved so client
// capabilities (color depth, dark/light) carry over.
func (s *Styles) SetTheme(t Theme) {
	s.Theme = t
}

// ---------------------------------------------------------------------------
// Common compound styles. One-off styles stay inline at the call site:
//   s.NewStyle().Foreground(s.Accent2).Bold(true)
// — anything reused across files lives here.
// ---------------------------------------------------------------------------

func (s *Styles) Status() lipgloss.Style {
	return s.NewStyle().Foreground(s.Muted).Padding(0, 1)
}

func (s *Styles) HelpKey() lipgloss.Style {
	return s.NewStyle().Foreground(s.Accent).Bold(true)
}

func (s *Styles) HelpDesc() lipgloss.Style {
	return s.NewStyle().Foreground(s.Muted)
}

func (s *Styles) CommitTime() lipgloss.Style {
	return s.NewStyle().Foreground(s.Muted).Italic(true)
}

// Pane is the rounded-border overlay container. Border switches between
// high- and low-contrast based on focus.
func (s *Styles) Pane(focused bool) lipgloss.Style {
	border := s.BorderLo
	if focused {
		border = s.BorderHi
	}
	return s.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(0, 1)
}
