package discussions

import (
	"fmt"
	"strings"

	"github.com/bchayka/gitstatus/internal/theme"
	"github.com/bchayka/gitstatus/internal/ui"
	"github.com/charmbracelet/lipgloss"
)

// renderTabs renders the branch tab row. The active branch's tab is highlighted;
// branches beyond the window are summarized by the "b ... more" affordance.
func (s *Screen) renderTabs(width int) string {
	visible := s.visibleTabBranches()
	var tabs []string
	for i, idx := range visible {
		label := fmt.Sprintf("%d %s", i+1, s.branches[idx].Name)
		if idx == s.branchIdx {
			tabs = append(tabs, theme.TabActive.Render(label))
		} else {
			tabs = append(tabs, theme.TabInactive.Render(label))
		}
	}
	more := theme.TabMore.Render(fmt.Sprintf("b  more (%d)", len(s.branches)))
	tabs = append(tabs, more)

	row := strings.Join(tabs, "  ")
	if lipgloss.Width(row) < width {
		row = lipgloss.PlaceHorizontal(width, lipgloss.Left, row)
	}
	return row
}

// renderFeed renders the centered commit feed for the active branch.
func (s *Screen) renderFeed(width, height int) string {
	b := s.currentBranch()
	if b == nil || len(b.Commits) == 0 {
		return ui.PadToHeight(theme.Status.Render("no commits yet — press 'n' to compose one"), height)
	}

	title := theme.Title.Render("feed: " + b.Name)

	var blocks []string
	var startLines []int
	line := 0
	for i, c := range b.Commits {
		focused := i == s.commitIdx
		block := renderCommit(c, focused, width)
		startLines = append(startLines, line)
		blocks = append(blocks, block)
		line += lipgloss.Height(block)
		if i < len(b.Commits)-1 {
			line++
		}
	}

	var bodyParts []string
	for i, block := range blocks {
		bodyParts = append(bodyParts, block)
		if i < len(blocks)-1 {
			bodyParts = append(bodyParts, ui.Divider(width))
		}
	}
	body := strings.Join(bodyParts, "\n")
	bodyLines := strings.Split(body, "\n")

	visibleH := height - 3
	if visibleH < 1 {
		visibleH = 1
	}

	selStart := 0
	selBlockH := 1
	if s.commitIdx < len(startLines) {
		selStart = startLines[s.commitIdx]
		selBlockH = lipgloss.Height(blocks[s.commitIdx])
	}
	selEnd := selStart + selBlockH

	scroll := 0
	if selEnd > visibleH {
		scroll = selEnd - visibleH
	}
	if selStart < scroll {
		scroll = selStart
	}
	if scroll+visibleH > len(bodyLines) {
		scroll = len(bodyLines) - visibleH
	}
	if scroll < 0 {
		scroll = 0
	}

	end := scroll + visibleH
	if end > len(bodyLines) {
		end = len(bodyLines)
	}
	visible := strings.Join(bodyLines[scroll:end], "\n")

	indicator := strings.Repeat(" ", width)
	if len(bodyLines) > visibleH {
		pct := 100
		if len(bodyLines)-visibleH > 0 {
			pct = scroll * 100 / (len(bodyLines) - visibleH)
		}
		indicator = lipgloss.NewStyle().
			Foreground(theme.Muted).
			Width(width).
			Align(lipgloss.Right).
			Render(fmt.Sprintf("scroll %d%%", pct))
	}
	parts := []string{title, indicator, "", visible}
	return ui.PadToHeight(strings.Join(parts, "\n"), height)
}

func renderCommit(c Commit, selected bool, width int) string {
	const chrome = 4
	bodyW := width - chrome
	if bodyW < 8 {
		bodyW = 8
	}

	header := fmt.Sprintf("%s  %s  %s",
		theme.CommitAuthor.Render(c.Author),
		theme.CommitTime.Render(ui.HumanizeTime(c.At)),
		theme.CommitSHA.Render(c.SHA),
	)
	msg := theme.CommitMsg.Render(ui.Wrap(c.Message, bodyW))

	likeMark := theme.LikeIcon.Render("♥")
	if c.Liked {
		likeMark = theme.Liked.Render("♥")
	}
	stats := fmt.Sprintf("%s %d   %s %d",
		likeMark, c.Likes,
		theme.CommentCount.Render("💬"), len(c.Comments),
	)

	content := strings.Join([]string{header, "", msg, "", stats}, "\n")

	box := lipgloss.NewStyle().Padding(0, 1).Width(bodyW)
	if selected {
		box = box.Border(lipgloss.RoundedBorder()).BorderForeground(theme.Accent)
	} else {
		box = box.Border(lipgloss.HiddenBorder())
	}
	return box.Render(content)
}

// renderPostBlock renders the top "post" card inside the details popup.
func renderPostBlock(c *Commit, width int, selected bool) string {
	header := fmt.Sprintf("%s  %s  %s",
		theme.CommitAuthor.Bold(true).Render(c.Author),
		theme.CommitTime.Render(ui.HumanizeTime(c.At)),
		theme.CommitSHA.Render(c.SHA),
	)
	body := theme.CommitMsg.Render(ui.Wrap(c.Message, width-4))
	likeMark := theme.LikeIcon.Render("♥")
	if c.Liked {
		likeMark = theme.Liked.Render("♥")
	}
	stats := fmt.Sprintf("%s %d   %s %d",
		likeMark, c.Likes,
		theme.CommentCount.Render("💬"), len(c.Comments),
	)
	content := strings.Join([]string{header, "", body, "", stats}, "\n")

	box := lipgloss.NewStyle().Padding(0, 1).Width(width - 4)
	if selected {
		return box.Border(lipgloss.RoundedBorder()).BorderForeground(theme.Accent).Render(content)
	}
	return box.Border(lipgloss.HiddenBorder()).Render(content)
}

func renderCommentBlock(cm Comment, depth, width int, selected bool) string {
	indent := strings.Repeat("  ", depth)
	innerW := width - len(indent) - 4
	if innerW < 12 {
		innerW = 12
	}

	header := fmt.Sprintf("%s  %s",
		theme.CommentAuthor.Render(cm.Author),
		theme.CommitTime.Render(ui.HumanizeTime(cm.At)),
	)
	body := theme.CommentBody.Render(ui.Wrap(cm.Body, innerW))
	likeMark := theme.LikeIcon.Render("♥")
	if cm.Liked {
		likeMark = theme.Liked.Render("♥")
	}
	stats := fmt.Sprintf("%s %d", likeMark, cm.Likes)
	if len(cm.Comments) > 0 {
		stats += "   " + theme.CommentCount.Render("↳") +
			lipgloss.NewStyle().Foreground(theme.Muted).Render(fmt.Sprintf(" %d", len(cm.Comments)))
	}
	content := strings.Join([]string{header, body, stats}, "\n")

	box := lipgloss.NewStyle().Padding(0, 1).Width(innerW)
	if selected {
		box = box.Border(lipgloss.RoundedBorder()).BorderForeground(theme.Accent)
	} else {
		box = box.Border(lipgloss.HiddenBorder())
	}
	return lipgloss.NewStyle().PaddingLeft(len(indent)).Render(box.Render(content))
}
