package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

const (
	branchPaneWidth = 28
	minWidth        = 80
	minHeight       = 22
)

func (m model) View() string {
	if m.width < minWidth || m.height < minHeight {
		return lipgloss.NewStyle().
			Foreground(colorWarn).
			Render(fmt.Sprintf("terminal too small (%dx%d), need at least %dx%d", m.width, m.height, minWidth, minHeight))
	}

	header := m.renderHeader()
	footer := m.renderFooter()
	input := m.renderInput()

	headerH := lipgloss.Height(header)
	footerH := lipgloss.Height(footer)
	inputH := lipgloss.Height(input)
	const bodyBorder = 2
	bodyH := m.height - headerH - footerH - inputH - bodyBorder
	if bodyH < 3 {
		bodyH = 3
	}

	bp := paneStyle(m.focus == focusBranches).
		Width(branchPaneWidth).
		Height(bodyH)
	fp := paneStyle(m.focus == focusFeed).
		Width(m.width - branchPaneWidth - 2).
		Height(bodyH)

	branches := bp.Render(m.renderBranches(bodyH))
	feed := fp.Render(m.renderFeed(m.width-branchPaneWidth-6, bodyH))

	body := lipgloss.JoinHorizontal(lipgloss.Top, branches, feed)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		body,
		input,
		footer,
	)
}

var logoLines = []string{
	" ██████╗ ██╗████████╗███████╗████████╗ █████╗ ████████╗██╗   ██╗███████╗",
	"██╔════╝ ██║╚══██╔══╝██╔════╝╚══██╔══╝██╔══██╗╚══██╔══╝██║   ██║██╔════╝",
	"██║  ███╗██║   ██║   ███████╗   ██║   ███████║   ██║   ██║   ██║███████╗",
	"██║   ██║██║   ██║   ╚════██║   ██║   ██╔══██║   ██║   ██║   ██║╚════██║",
	"╚██████╔╝██║   ██║   ███████║   ██║   ██║  ██║   ██║   ╚██████╔╝███████║",
	" ╚═════╝ ╚═╝   ╚═╝   ╚══════╝   ╚═╝   ╚═╝  ╚═╝   ╚═╝    ╚═════╝ ╚══════╝",
}

func (m model) renderLogo() string {
	gradient := []lipgloss.Color{
		colorAccent, colorAccent,
		colorAccent2, colorAccent2,
		colorLike, colorLike,
	}
	var rendered []string
	for i, line := range logoLines {
		s := lipgloss.NewStyle().
			Foreground(gradient[i%len(gradient)]).
			Bold(true).
			Render(line)
		rendered = append(rendered, s)
	}
	return strings.Join(rendered, "\n")
}

func (m model) renderHeader() string {
	logo := m.renderLogo()
	logoW := lipgloss.Width(logo)

	branchName := ""
	if b := pointerToBranch(&m); b != nil {
		branchName = "on " + b.Name
	}

	totalCommits, totalLikes, totalComments := repoTotals(m.branches)
	rightLines := []string{
		lipgloss.NewStyle().Foreground(colorAccent2).Bold(true).Render(branchName),
		fmt.Sprintf(
			"%s %d   %s %d   %s %d",
			lipgloss.NewStyle().Foreground(colorMuted).Render("commits"), totalCommits,
			likeStyle.Render("♥"), totalLikes,
			commentCountStyle.Render("💬"), totalComments,
		),
		lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
			Render(time.Now().Format("Mon Jan 2  15:04:05")),
	}
	right := lipgloss.JoinVertical(lipgloss.Left, rightLines...)
	rightW := lipgloss.Width(right)

	if logoW+rightW+4 > m.width {
		return logo
	}

	gap := m.width - logoW - rightW
	if gap < 4 {
		gap = 4
	}
	spacer := lipgloss.NewStyle().Width(gap).Render("")

	return lipgloss.JoinHorizontal(lipgloss.Top, logo, spacer, right)
}

func pointerToBranch(m *model) *Branch {
	if len(m.branches) == 0 {
		return nil
	}
	return &m.branches[m.branchIdx]
}

func repoTotals(branches []Branch) (commits, likes, comments int) {
	for _, b := range branches {
		commits += len(b.Commits)
		for _, c := range b.Commits {
			likes += c.Likes
			comments += len(c.Comments)
		}
	}
	return
}

func (m model) renderBranches(height int) string {
	var lines []string
	lines = append(lines, titleStyle.Render("branches"))
	lines = append(lines, "")
	for i, b := range m.branches {
		label := fmt.Sprintf("%-20s %d", truncate(b.Name, 20), len(b.Commits))
		if i == m.branchIdx {
			lines = append(lines, branchItemSelStyle.Render("▸ "+label))
		} else {
			lines = append(lines, branchItemStyle.Render("  "+label))
		}
	}
	return clipVertical(strings.Join(lines, "\n"), height)
}

func (m model) renderFeed(width, height int) string {
	b := pointerToBranch(&m)
	if b == nil || len(b.Commits) == 0 {
		return statusStyle.Render("no commits yet — switch to input (Tab) and push one")
	}

	var sections []string
	sections = append(sections, titleStyle.Render("feed: "+b.Name))
	sections = append(sections, "")

	for i, c := range b.Commits {
		sections = append(sections, m.renderCommit(c, i == m.commitIdx && m.focus == focusFeed, width))
		if i != len(b.Commits)-1 {
			sections = append(sections, lipgloss.NewStyle().Foreground(colorBorderLo).Render(strings.Repeat("─", width)))
		}
	}

	if m.mode == modeComment {
		sections = append(sections, "")
		sections = append(sections, lipgloss.NewStyle().
			Foreground(colorAccent2).
			Render("commenting on "+m.currentCommitSHA()+":"))
		sections = append(sections, m.commentInput.View())
	}

	return clipVertical(strings.Join(sections, "\n"), height)
}

func (m model) currentCommitSHA() string {
	c := m.peekCommit()
	if c == nil {
		return ""
	}
	return c.SHA
}

func (m model) peekCommit() *Commit {
	b := pointerToBranch(&m)
	if b == nil || len(b.Commits) == 0 {
		return nil
	}
	idx := m.commitIdx
	if idx >= len(b.Commits) {
		idx = len(b.Commits) - 1
	}
	return &b.Commits[idx]
}

func (m model) renderCommit(c Commit, selected bool, width int) string {
	marker := "  "
	if selected {
		marker = lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("▸ ")
	}

	header := fmt.Sprintf("%s%s  %s  %s",
		marker,
		commitSHAStyle.Render(c.SHA),
		commitAuthorStyle.Render(c.Author),
		commitTimeStyle.Render(humanizeTime(c.At)),
	)

	msg := commitMsgStyle.Render("    " + c.Message)

	likeMark := likeStyle.Render("♥")
	if c.Liked {
		likeMark = likedStyle.Render("♥")
	}
	stats := fmt.Sprintf("    %s %d   %s %d",
		likeMark, c.Likes,
		commentCountStyle.Render("💬"), len(c.Comments),
	)

	parts := []string{header, msg, stats}

	if selected && len(c.Comments) > 0 {
		parts = append(parts, "")
		for _, cm := range c.Comments {
			line := fmt.Sprintf("    %s %s  %s",
				commentAuthorStyle.Render(cm.Author),
				commitTimeStyle.Render(humanizeTime(cm.At)),
				commentBodyStyle.Render(cm.Body),
			)
			parts = append(parts, line)
		}
	}

	_ = width
	return strings.Join(parts, "\n")
}

func (m model) renderInput() string {
	style := paneStyle(m.focus == focusInput).Width(m.width - 2)
	hint := ""
	b := pointerToBranch(&m)
	if b != nil {
		hint = lipgloss.NewStyle().Foreground(colorMuted).Render(" → " + b.Name)
	}
	return style.Render(m.commitInput.View() + hint)
}

func (m model) renderFooter() string {
	keys := []struct{ k, d string }{
		{"tab", "switch pane"},
		{"j/k", "move"},
		{"l", "like"},
		{"c", "comment"},
		{"i", "compose"},
		{"q", "quit"},
	}
	if m.mode == modeComment {
		keys = []struct{ k, d string }{
			{"enter", "post"},
			{"esc", "cancel"},
		}
	}
	var parts []string
	for _, k := range keys {
		parts = append(parts, helpKeyStyle.Render(k.k)+" "+helpDescStyle.Render(k.d))
	}
	help := strings.Join(parts, "  ·  ")

	flash := ""
	if m.flash != "" {
		flash = lipgloss.NewStyle().Foreground(colorOk).Render("  " + m.flash)
	}
	return statusStyle.Render(help) + flash
}

func humanizeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("Jan 2")
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}

func clipVertical(s string, h int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= h {
		return s
	}
	return strings.Join(lines[:h], "\n")
}
