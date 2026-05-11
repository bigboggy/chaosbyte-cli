package discussions

import (
	"strings"
	"time"

	"github.com/bchayka/gitstatus/internal/screens"
	"github.com/bchayka/gitstatus/internal/theme"
	"github.com/bchayka/gitstatus/internal/ui"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// compose mode lets the user push a new commit to the active branch.

func (s *Screen) enterCompose() tea.Cmd {
	s.mode = modeCompose
	s.commitInput.SetValue("")
	s.commitInput.Focus()
	return textarea.Blink
}

func (s *Screen) updateCompose(km tea.KeyMsg) (screens.Screen, tea.Cmd) {
	switch km.String() {
	case "ctrl+enter", "alt+enter", "ctrl+j":
		s.commitInput.InsertString("\n")
		return s, nil
	case "enter", "ctrl+s", "ctrl+d":
		text := strings.TrimRight(strings.TrimSpace(s.commitInput.Value()), "\n")
		var cmd tea.Cmd
		if text != "" {
			if b := s.currentBranch(); b != nil {
				b.Commits = append([]Commit{{
					SHA:     ui.FakeSHA(),
					Author:  meUser,
					Message: text,
					At:      time.Now(),
				}}, b.Commits...)
				s.commitIdx = 0
				cmd = screens.Flash("pushed to " + b.Name)
			}
		}
		s.commitInput.SetValue("")
		s.commitInput.Blur()
		s.mode = modeNormal
		return s, cmd
	case "esc":
		s.commitInput.SetValue("")
		s.commitInput.Blur()
		s.mode = modeNormal
		return s, nil
	}
	var cmd tea.Cmd
	s.commitInput, cmd = s.commitInput.Update(km)
	return s, cmd
}

func (s *Screen) renderComposePopup(width, height int) string {
	logo := theme.RenderLogo()
	logoH := lipgloss.Height(logo)

	popupW, _ := ui.PopupSize(width, height)

	const topBlanks = 3
	const midBlank = 1
	const boxOverhead = 8

	avail := height - topBlanks - logoH - midBlank
	taH := avail - boxOverhead
	if taH < 2 {
		taH = 2
	}
	if taH > 10 {
		taH = 10
	}

	bn := ""
	if b := s.currentBranch(); b != nil {
		bn = b.Name
	}
	contentW := popupW - 6
	if contentW < 30 {
		contentW = 30
	}
	title := lipgloss.NewStyle().Foreground(theme.Accent2).Bold(true).Render("commit on " + bn)
	s.commitInput.SetWidth(contentW)
	s.commitInput.SetHeight(taH)
	ta := s.commitInput.View()
	hint := theme.HelpDesc.Render("enter push   ·   ctrl+enter newline   ·   esc cancel")

	rule := lipgloss.NewStyle().Foreground(theme.BorderLo).Render(strings.Repeat("─", contentW))

	inner := strings.Join([]string{title, rule, ta, rule, hint}, "\n")
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Accent).
		Padding(1, 2).
		Render(inner)

	logoCentered := lipgloss.PlaceHorizontal(width, lipgloss.Center, logo)
	boxCentered := lipgloss.PlaceHorizontal(width, lipgloss.Center, box)

	parts := make([]string, 0, topBlanks+1+midBlank+1)
	for i := 0; i < topBlanks; i++ {
		parts = append(parts, "")
	}
	parts = append(parts, logoCentered)
	for i := 0; i < midBlank; i++ {
		parts = append(parts, "")
	}
	parts = append(parts, boxCentered)

	content := strings.Join(parts, "\n")
	used := lipgloss.Height(content)
	if pad := height - used; pad > 0 {
		content += strings.Repeat("\n", pad)
	}
	return content
}
