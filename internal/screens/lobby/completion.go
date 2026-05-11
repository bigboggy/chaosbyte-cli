package lobby

import (
	"fmt"
	"strings"

	"github.com/bchayka/gitstatus/internal/theme"
	"github.com/charmbracelet/lipgloss"
)

// matchCommands returns canonical commands whose names start with prefix.
// Aliases are excluded so the suggestion strip doesn't get noisy.
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

// resetCompletion clears the active completion cycle. Call after any input
// edit so the next Tab starts from the new stem.
func (s *Screen) resetCompletion() {
	s.completionStem = ""
	s.completionIdx = -1
}

// cycleCompletion replaces the input with the next (delta>0) or previous
// match of the completion stem. The stem is captured the first time Tab is
// pressed since the input last changed, so successive Tabs walk through all
// matches even though the input itself keeps changing under them.
func (s *Screen) cycleCompletion(delta int) {
	cur := s.input.Value()
	if s.completionStem == "" || s.completionIdx < 0 {
		s.completionStem = cur
	}
	matches := matchCommands(s.completionStem)
	if len(matches) == 0 {
		return
	}
	if s.completionIdx < 0 {
		if delta > 0 {
			s.completionIdx = 0
		} else {
			s.completionIdx = len(matches) - 1
		}
	} else {
		s.completionIdx = (s.completionIdx + delta + len(matches)) % len(matches)
	}
	s.input.SetValue(matches[s.completionIdx])
	s.input.CursorEnd()
}

// renderCompletionStrip shows matching slash commands above the input. It hides
// itself when there's nothing useful to suggest (no slash prefix, no matches,
// or the input already equals the only match).
func (s *Screen) renderCompletionStrip(width int) string {
	cur := s.input.Value()
	matches := matchCommands(cur)
	if len(matches) == 0 {
		return ""
	}
	if len(matches) == 1 && matches[0] == cur {
		return ""
	}

	label := lipgloss.NewStyle().Foreground(theme.Muted).Italic(true).Render("tab ")

	if len(matches) == 1 {
		name := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true).Render(matches[0])
		desc := lipgloss.NewStyle().Foreground(theme.Muted).Render("  " + commandDesc(matches[0]))
		return label + name + desc
	}

	const inlineCap = 10
	var chips []string
	for i, name := range matches {
		if i >= inlineCap {
			chips = append(chips, lipgloss.NewStyle().Foreground(theme.Muted).
				Render(fmt.Sprintf("+%d more", len(matches)-inlineCap)))
			break
		}
		style := lipgloss.NewStyle().Foreground(theme.Accent)
		if i == s.completionIdx {
			style = style.Bold(true).Underline(true)
		}
		chips = append(chips, style.Render(name))
	}
	return label + strings.Join(chips, "  ")
}
