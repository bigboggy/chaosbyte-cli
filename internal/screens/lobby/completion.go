package lobby

import (
	"fmt"
	"strings"

	"github.com/bchayka/gitstatus/internal/theme"
	"github.com/charmbracelet/lipgloss"
)

// palettePageSize caps how many command rows are visible at once. The selection
// scrolls inside this window so all commands remain reachable via arrow keys.
const palettePageSize = 10

// commandColWidth is the fixed left-column width for the command name. The
// description fills the remaining width. 14 fits /discussions plus padding.
const commandColWidth = 14

// matchCommands returns canonical commands whose names start with prefix.
// Aliases are excluded so the palette stays compact.
func matchCommands(prefix string) []string {
	if !strings.HasPrefix(prefix, "/") {
		return nil
	}
	var out []string
	for _, c := range builtins {
		if strings.HasPrefix(c.name, prefix) {
			out = append(out, c.name)
		}
	}
	return out
}

func commandDesc(name string) string {
	for _, c := range builtins {
		if c.name == name {
			return c.desc
		}
	}
	return ""
}

// paletteVisible reports whether the multi-row popup should be shown. It hides
// when the input isn't a slash command, has no matches, or already exactly
// equals the only match (nothing left to suggest).
func (s *Screen) paletteVisible() bool {
	matches := matchCommands(s.input.Value())
	if len(matches) == 0 {
		return false
	}
	if len(matches) == 1 && matches[0] == s.input.Value() {
		return false
	}
	return true
}

// movePalette shifts the highlighted match by delta (+1 down, -1 up), wrapping
// at both ends.
func (s *Screen) movePalette(delta int) {
	matches := matchCommands(s.input.Value())
	if len(matches) == 0 {
		return
	}
	s.paletteIdx = (s.paletteIdx + delta + len(matches)) % len(matches)
}

// resetPalette zeros the selection. Called after the input is edited so the
// filtered list starts highlighted at the top.
func (s *Screen) resetPalette() {
	s.paletteIdx = 0
}

// fillPalette replaces the input with the highlighted match, without
// submitting. Used by Tab so the user can fill in a command then type args.
func (s *Screen) fillPalette() {
	matches := matchCommands(s.input.Value())
	if len(matches) == 0 {
		return
	}
	idx := s.clampedPaletteIdx(len(matches))
	s.input.SetValue(matches[idx])
	s.input.CursorEnd()
	s.paletteIdx = 0
}

// acceptPalette returns the currently-highlighted command name, intended to be
// inserted by Enter just before submitting. Empty string means "nothing to
// accept" (palette is hidden).
func (s *Screen) acceptPalette() string {
	if !s.paletteVisible() {
		return ""
	}
	matches := matchCommands(s.input.Value())
	if len(matches) == 0 {
		return ""
	}
	return matches[s.clampedPaletteIdx(len(matches))]
}

func (s *Screen) clampedPaletteIdx(n int) int {
	idx := s.paletteIdx
	if idx < 0 || idx >= n {
		return 0
	}
	return idx
}

// paletteHeight returns the row count the popup will occupy. 0 when hidden.
func (s *Screen) paletteHeight() int {
	if !s.paletteVisible() {
		return 0
	}
	n := len(matchCommands(s.input.Value()))
	if n > palettePageSize {
		return palettePageSize
	}
	return n
}

// renderPalette draws the multi-row command popup. The selection is shown as
// a full-width highlighted row; the visible window scrolls with the selection
// so off-screen matches remain reachable.
func (s *Screen) renderPalette(width int) string {
	if !s.paletteVisible() {
		return ""
	}
	matches := matchCommands(s.input.Value())
	visible := len(matches)
	if visible > palettePageSize {
		visible = palettePageSize
	}

	sel := s.clampedPaletteIdx(len(matches))

	// Scroll the window so the selection stays inside it.
	start := 0
	if sel >= visible {
		start = sel - visible + 1
	}
	end := start + visible
	if end > len(matches) {
		end = len(matches)
		start = end - visible
		if start < 0 {
			start = 0
		}
	}

	rows := make([]string, 0, visible)
	for i := start; i < end; i++ {
		rows = append(rows, renderPaletteRow(matches[i], commandDesc(matches[i]), width, i == sel))
	}

	// "+N more" hint when the list overflows the window.
	overflow := len(matches) - visible
	if overflow > 0 {
		hint := lipgloss.NewStyle().
			Foreground(theme.Muted).
			Italic(true).
			Render(fmt.Sprintf("  +%d more — arrows to scroll", overflow))
		// Replace the last visible row with the hint? No — keep all matches
		// visible and append the hint after. Caller already sized the chat
		// area for paletteHeight(); we'd need to bump that. Skip the hint
		// for now to keep layout deterministic.
		_ = hint
	}

	return strings.Join(rows, "\n")
}

func renderPaletteRow(cmd, desc string, width int, selected bool) string {
	body := fmt.Sprintf("  %-*s  %s", commandColWidth, cmd, desc)
	if selected {
		return lipgloss.NewStyle().
			Foreground(theme.Bg).
			Background(theme.Accent).
			Bold(true).
			Width(width).
			Render(body)
	}
	cmdPart := lipgloss.NewStyle().Foreground(theme.Accent).
		Render(fmt.Sprintf("  %-*s", commandColWidth, cmd))
	descPart := lipgloss.NewStyle().Foreground(theme.Muted).Render("  " + desc)
	return cmdPart + descPart
}
