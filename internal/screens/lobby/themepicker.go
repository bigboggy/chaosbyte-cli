package lobby

import (
	"fmt"
	"strings"

	"github.com/bchayka/gitstatus/internal/theme"
	"github.com/charmbracelet/lipgloss"
)

// themePickerState is the modal picker shown by `/theme`. It owns its own
// selection and remembers the theme ID that was active when the picker opened,
// so Esc restores the original look after live-preview navigation.
type themePickerState struct {
	themes []theme.Theme
	idx    int
	origID string
}

// openThemePicker initializes the picker with the registered themes and
// highlights the current one.
func (s *Screen) openThemePicker() {
	ids := theme.IDs()
	themes := make([]theme.Theme, 0, len(ids))
	selected := 0
	for i, id := range ids {
		t, _ := theme.Get(id)
		themes = append(themes, t)
		if id == s.styles.Theme.ID {
			selected = i
		}
	}
	s.themePicker = &themePickerState{
		themes: themes,
		idx:    selected,
		origID: s.styles.Theme.ID,
	}
}

func (s *Screen) themePickerVisible() bool { return s.themePicker != nil }

// moveThemePicker advances the highlighted row and live-previews the new
// theme by swapping it onto the shared *Styles immediately.
func (s *Screen) moveThemePicker(delta int) {
	p := s.themePicker
	if p == nil || len(p.themes) == 0 {
		return
	}
	p.idx = (p.idx + delta + len(p.themes)) % len(p.themes)
	s.styles.SetTheme(p.themes[p.idx])
}

// closeThemePicker either keeps the current selection (apply=true) or restores
// the theme that was active when the picker opened (apply=false).
func (s *Screen) closeThemePicker(apply bool) {
	p := s.themePicker
	if p == nil {
		return
	}
	if !apply {
		if t, ok := theme.Get(p.origID); ok {
			s.styles.SetTheme(t)
		}
	} else if len(p.themes) > 0 {
		applied := p.themes[p.idx]
		s.postSystem(fmt.Sprintf("theme set to %s (%s)", applied.ID, applied.DisplayName))
	}
	s.themePicker = nil
}

// themePickerHeight is the row count the picker occupies in the layout.
// One header row + one row per theme.
func (s *Screen) themePickerHeight() int {
	if s.themePicker == nil {
		return 0
	}
	return len(s.themePicker.themes) + 1
}

// renderThemePicker draws the picker above the input line.
func (s *Screen) renderThemePicker(width int) string {
	p := s.themePicker
	if p == nil {
		return ""
	}
	st := s.styles
	header := st.NewStyle().Foreground(st.Muted).Italic(true).
		Render("  pick a theme — ↑/↓ preview · enter apply · esc cancel")
	rows := []string{header}
	for i, t := range p.themes {
		rows = append(rows, renderThemePickerRow(st, t, width, i == p.idx))
	}
	return strings.Join(rows, "\n")
}

// renderThemePickerRow draws one row: theme id, display name, and a row of
// swatches painted in the theme's own colors so the user previews the palette
// without having to navigate to each option.
func renderThemePickerRow(st *theme.Styles, t theme.Theme, width int, selected bool) string {
	label := fmt.Sprintf("  %-12s  %-18s  ", t.ID, t.DisplayName)
	swatches := renderSwatches(st, t)
	body := label + swatches
	if selected {
		// Pad to width so the highlight spans the full row.
		bodyW := lipgloss.Width(body)
		if bodyW < width {
			body += strings.Repeat(" ", width-bodyW)
		}
		return st.NewStyle().
			Background(st.Accent).
			Foreground(st.Bg).
			Bold(true).
			Render(body)
	}
	return body
}

// renderSwatches draws five colored blocks using the previewed theme's colors,
// not the active theme's — so each row shows what you'd be switching to.
func renderSwatches(st *theme.Styles, t theme.Theme) string {
	colors := []lipgloss.Color{t.Accent, t.Accent2, t.OK, t.Warn, t.Like}
	var b strings.Builder
	for _, c := range colors {
		b.WriteString(st.NewStyle().Foreground(c).Render("██"))
	}
	return b.String()
}
