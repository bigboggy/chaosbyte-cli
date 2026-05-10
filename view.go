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

	const paneChrome = 4 // border (2) + horizontal padding (2)

	header := m.renderHeader()
	footer := m.renderFooter()

	headerH := lipgloss.Height(header)
	footerH := lipgloss.Height(footer)
	bodyH := m.height - headerH - footerH
	if bodyH < 6 {
		bodyH = 6
	}

	var body string
	if m.mode == modeCompose || m.mode == modeComment {
		body = m.renderPopup(m.width, bodyH)
	} else {
		leftWidth := m.width - (branchPaneWidth + paneChrome)
		if leftWidth < 30 {
			leftWidth = 30
		}
		branchContentH := bodyH - 2
		if branchContentH < 3 {
			branchContentH = 3
		}
		feedContentH := bodyH - 2
		if feedContentH < 3 {
			feedContentH = 3
		}

		bp := paneStyle(m.focus == focusBranches).
			Width(branchPaneWidth).
			Height(branchContentH)
		branches := bp.Render(m.renderBranches(branchContentH))

		fp := paneStyle(m.focus == focusFeed).
			Width(leftWidth - paneChrome).
			Height(feedContentH)
		feedContentW := leftWidth - paneChrome - 2
		feed := fp.Render(m.renderFeed(feedContentW, feedContentH))

		body = lipgloss.JoinHorizontal(lipgloss.Top, feed, branches)
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
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
	branchName := ""
	if b := pointerToBranch(&m); b != nil {
		branchName = "on " + b.Name
	}
	totalCommits, totalLikes, totalComments := repoTotals(m.branches)

	title := lipgloss.NewStyle().Foreground(colorAccent2).Bold(true).Render("gitstatus")
	sep := lipgloss.NewStyle().Foreground(colorMuted).Render(" · ")

	left := title + sep +
		lipgloss.NewStyle().Foreground(colorOk).Render(branchName) + sep +
		lipgloss.NewStyle().Foreground(colorMuted).Render(fmt.Sprintf("%d commits", totalCommits)) + sep +
		likeStyle.Render(fmt.Sprintf("♥ %d", totalLikes)) + sep +
		commentCountStyle.Render(fmt.Sprintf("💬 %d", totalComments))

	right := lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
		Render(time.Now().Format("Mon 15:04:05"))

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}

	return " " + left + strings.Repeat(" ", gap) + right + " "
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
		return statusStyle.Render("no commits yet — press 'n' to compose one")
	}

	title := titleStyle.Render("feed: " + b.Name)

	var blocks []string
	var startLines []int
	line := 0
	for i, c := range b.Commits {
		focused := i == m.commitIdx && m.focus == focusFeed
		block := m.renderCommit(c, focused, width)
		startLines = append(startLines, line)
		blocks = append(blocks, block)
		line += lipgloss.Height(block)
		if i < len(b.Commits)-1 {
			line++ // divider
		}
	}

	var bodyParts []string
	for i, block := range blocks {
		bodyParts = append(bodyParts, block)
		if i < len(blocks)-1 {
			bodyParts = append(bodyParts, dividerLine(width))
		}
	}
	body := strings.Join(bodyParts, "\n")
	bodyLines := strings.Split(body, "\n")

	visibleH := height - 2 // -2 for title + blank
	if visibleH < 1 {
		visibleH = 1
	}

	selStart := 0
	selBlockH := 1
	if m.commitIdx < len(startLines) {
		selStart = startLines[m.commitIdx]
		selBlockH = lipgloss.Height(blocks[m.commitIdx])
	}
	selEnd := selStart + selBlockH

	scroll := 0
	if selEnd > visibleH {
		scroll = selEnd - visibleH
	}
	if selStart < scroll {
		scroll = selStart
	}
	if scroll < 0 {
		scroll = 0
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

	scrollIndicator := ""
	if len(bodyLines) > visibleH {
		pct := 100
		if len(bodyLines)-visibleH > 0 {
			pct = scroll * 100 / (len(bodyLines) - visibleH)
		}
		scrollIndicator = lipgloss.NewStyle().Foreground(colorMuted).Render(fmt.Sprintf("  %d%%", pct))
	}

	return strings.Join([]string{title + scrollIndicator, "", visible}, "\n")
}

func dividerLine(width int) string {
	return lipgloss.NewStyle().Foreground(colorBorderLo).Render(strings.Repeat("─", width))
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

	const indentPrefix = "    "
	bodyW := width - len(indentPrefix)
	if bodyW < 8 {
		bodyW = 8
	}
	wrapped := wrap(c.Message, bodyW)
	indented := indent(wrapped, indentPrefix)
	msg := commitMsgStyle.Render(indented)

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
			cmBody := indent(wrap(cm.Body, bodyW-2), "      ")
			parts = append(parts, fmt.Sprintf("    %s %s",
				commentAuthorStyle.Render(cm.Author),
				commitTimeStyle.Render(humanizeTime(cm.At)),
			))
			parts = append(parts, commentBodyStyle.Render(cmBody))
		}
	}

	return strings.Join(parts, "\n")
}

func popupSize(termW, termH int) (w, h int) {
	w = termW * 80 / 100
	if w > 90 {
		w = 90
	}
	if w < 50 {
		w = 50
	}
	h = 14
	if h > termH-2 {
		h = termH - 2
	}
	if h < 8 {
		h = 8
	}
	return
}

func (m model) renderPopup(width, height int) string {
	if m.mode == modeCompose {
		return m.renderComposeScreen(width, height)
	}
	return m.renderCommentPopup(width, height)
}

func (m model) renderComposeScreen(width, height int) string {
	logo := m.renderLogo()
	logoH := lipgloss.Height(logo)

	popupW, _ := popupSize(width, height)

	const topBlanks = 3
	const midBlank = 1
	const boxOverhead = 8 // border(2) + vert padding(2) + title(1) + 2 rules + hint(1)

	avail := height - topBlanks - logoH - midBlank
	taH := avail - boxOverhead
	if taH < 2 {
		taH = 2
	}
	if taH > 10 {
		taH = 10
	}

	bn := ""
	if b := pointerToBranch(&m); b != nil {
		bn = b.Name
	}
	title := lipgloss.NewStyle().Foreground(colorAccent2).Bold(true).Render("commit on " + bn)
	m.commitInput.SetWidth(popupW - 4)
	m.commitInput.SetHeight(taH)
	ta := m.commitInput.View()
	hint := helpDescStyle.Render("ctrl+s push   ·   enter newline   ·   esc cancel")

	rule := lipgloss.NewStyle().Foreground(colorBorderLo).Render(strings.Repeat("─", popupW-4))

	inner := strings.Join([]string{title, rule, ta, rule, hint}, "\n")
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccent).
		Padding(1, 2).
		Width(popupW - 4).
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

func (m model) renderCommentPopup(width, height int) string {
	popupW, _ := popupSize(width, height)

	title := lipgloss.NewStyle().Foreground(colorAccent2).Bold(true).Render("reply to " + m.currentCommitSHA())
	m.commentInput.SetWidth(popupW - 4)
	ta := m.commentInput.View()
	hint := helpDescStyle.Render("ctrl+s post   ·   enter newline   ·   esc cancel")

	rule := lipgloss.NewStyle().Foreground(colorBorderLo).Render(strings.Repeat("─", popupW-4))

	inner := strings.Join([]string{title, rule, ta, rule, hint}, "\n")
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccent).
		Padding(1, 2).
		Width(popupW - 4).
		Render(inner)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

func (m model) renderFooter() string {
	var keys []struct{ k, d string }
	switch m.mode {
	case modeCompose:
		keys = []struct{ k, d string }{
			{"ctrl+s", "push"},
			{"enter", "newline"},
			{"esc", "cancel"},
		}
	case modeComment:
		keys = []struct{ k, d string }{
			{"ctrl+s", "post"},
			{"enter", "newline"},
			{"esc", "cancel"},
		}
	default:
		keys = []struct{ k, d string }{
			{"n", "new commit"},
			{"c", "comment"},
			{"l", "like"},
			{"j/k", "move"},
			{"tab", "switch pane"},
			{"q", "quit"},
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

func wrap(s string, width int) string {
	if width < 4 {
		width = 4
	}
	var out []string
	for _, paragraph := range strings.Split(s, "\n") {
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		var line string
		for _, w := range words {
			if line == "" {
				line = w
				continue
			}
			if lipgloss.Width(line)+1+lipgloss.Width(w) > width {
				out = append(out, line)
				line = w
			} else {
				line += " " + w
			}
		}
		if line != "" {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

func indent(s, prefix string) string {
	if s == "" {
		return prefix
	}
	return prefix + strings.ReplaceAll(s, "\n", "\n"+prefix)
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
