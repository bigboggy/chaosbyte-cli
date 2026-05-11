package discussions

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bchayka/gitstatus/internal/screens"
	"github.com/bchayka/gitstatus/internal/theme"
	"github.com/bchayka/gitstatus/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// branch picker mode — modal list for switching the active branch.

const visibleTabs = 3

// visibleTabBranches returns the branch indices to show as tabs, ensuring the
// active branch is always one of them.
func (s *Screen) visibleTabBranches() []int {
	n := len(s.branches)
	if n == 0 {
		return nil
	}
	limit := visibleTabs
	if n < limit {
		limit = n
	}
	idxs := make([]int, 0, limit)
	for i := 0; i < limit; i++ {
		idxs = append(idxs, i)
	}
	hasActive := false
	for _, i := range idxs {
		if i == s.branchIdx {
			hasActive = true
			break
		}
	}
	if !hasActive {
		idxs[len(idxs)-1] = s.branchIdx
		sort.Ints(idxs)
	}
	return idxs
}

func (s *Screen) updateBranchPicker(km tea.KeyMsg) (screens.Screen, tea.Cmd) {
	switch km.String() {
	case "esc", "b":
		s.mode = modeNormal
		return s, nil
	case "j", "down":
		if s.branchPickerIdx < len(s.branches)-1 {
			s.branchPickerIdx++
		}
	case "k", "up":
		if s.branchPickerIdx > 0 {
			s.branchPickerIdx--
		}
	case "g":
		s.branchPickerIdx = 0
	case "G":
		s.branchPickerIdx = len(s.branches) - 1
	case "enter":
		var cmd tea.Cmd
		if s.branchPickerIdx >= 0 && s.branchPickerIdx < len(s.branches) {
			s.branchIdx = s.branchPickerIdx
			s.commitIdx = 0
			cmd = screens.Flash("checked out " + s.branches[s.branchIdx].Name)
		}
		s.mode = modeNormal
		return s, cmd
	}
	return s, nil
}

func (s *Screen) renderBranchPicker(width, height int) string {
	popupW := width * 60 / 100
	if popupW > 70 {
		popupW = 70
	}
	if popupW < 40 {
		popupW = 40
	}
	contentW := popupW - 6
	if contentW < 24 {
		contentW = 24
	}

	title := lipgloss.NewStyle().Foreground(theme.Accent2).Bold(true).
		Render("checkout a branch")
	rule := lipgloss.NewStyle().Foreground(theme.BorderLo).
		Render(strings.Repeat("─", contentW))

	var rows []string
	for i, b := range s.branches {
		marker := "  "
		if i == s.branchPickerIdx {
			marker = "▸ "
		}
		label := fmt.Sprintf("%s%-30s %d", marker, ui.Truncate(b.Name, 30), len(b.Commits))
		switch {
		case i == s.branchPickerIdx:
			rows = append(rows, theme.BranchItemSel.Width(contentW).Render(label))
		case i == s.branchIdx:
			rows = append(rows, lipgloss.NewStyle().Foreground(theme.OK).Render(label))
		default:
			rows = append(rows, theme.BranchItem.Render(label))
		}
	}
	hint := theme.HelpDesc.Render("j/k move   ·   enter checkout   ·   esc cancel")

	inner := lipgloss.JoinVertical(lipgloss.Left,
		title, rule, strings.Join(rows, "\n"), rule, hint,
	)
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Accent).
		Padding(1, 2).
		Render(inner)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
