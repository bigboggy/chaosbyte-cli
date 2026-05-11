package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

const (
	minWidth  = 80
	minHeight = 22
)

func (m model) View() string {
	if m.width < minWidth || m.height < minHeight {
		return lipgloss.NewStyle().
			Foreground(colorWarn).
			Render(fmt.Sprintf("terminal too small (%dx%d), need at least %dx%d", m.width, m.height, minWidth, minHeight))
	}

	header := m.renderHeader()
	footer := m.renderFooter()

	headerH := lipgloss.Height(header)
	footerH := lipgloss.Height(footer)
	bodyH := m.height - headerH - footerH
	if bodyH < 6 {
		bodyH = 6
	}

	var body string
	switch m.screen {
	case screenHome:
		body = m.renderHome(m.width, bodyH)
	case screenChat:
		body = m.renderChat(m.width, bodyH)
	case screenNews:
		body = m.renderNews(m.width, bodyH)
	case screenResources:
		body = m.renderResources(m.width, bodyH)
	case screenSpotlight:
		body = m.renderSpotlight(m.width, bodyH)
	case screenGames:
		body = m.renderGames(m.width, bodyH)
	case screenDiscussions:
		body = m.renderDiscussionsBody(m.width, bodyH)
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

// feedShellWidth is the width budget for the centered feed area.
func feedShellWidth(termW int) int {
	w := termW * 80 / 100
	if w > 100 {
		w = 100
	}
	if w < 60 {
		w = 60
	}
	if w > termW-2 {
		w = termW - 2
	}
	return w
}

// ============================================================================
// Header
// ============================================================================

func (m model) renderHeader() string {
	title := lipgloss.NewStyle().Foreground(colorAccent2).Bold(true).Render("gitstatus hub")
	sep := lipgloss.NewStyle().Foreground(colorMuted).Render(" · ")

	screenTag := lipgloss.NewStyle().
		Foreground(colorBg).
		Background(colorAccent).
		Bold(true).
		Padding(0, 1).
		Render(m.screen.name())

	var ctx string
	switch m.screen {
	case screenHome:
		ctx = lipgloss.NewStyle().Foreground(colorMuted).Render("all-in-one for devs & vibe coders")
	case screenChat:
		if m.chatActive >= 0 && m.chatActive < len(m.channels) {
			ch := m.channels[m.chatActive]
			ctx = lipgloss.NewStyle().Foreground(colorOk).Render(ch.Name) + sep +
				lipgloss.NewStyle().Foreground(colorMuted).Render(fmt.Sprintf("%d online", ch.Online))
		} else {
			ctx = lipgloss.NewStyle().Foreground(colorMuted).Render(fmt.Sprintf("%d channels", len(m.channels)))
		}
	case screenNews:
		ctx = lipgloss.NewStyle().Foreground(colorMuted).Render(fmt.Sprintf("%d/%d", m.newsIdx+1, len(m.newsItems)))
	case screenResources:
		ctx = lipgloss.NewStyle().Foreground(colorMuted).Render(resourceTabs[m.resourcesTab])
	case screenSpotlight:
		_, secs := m.spotlightRotation()
		ctx = lipgloss.NewStyle().Foreground(colorAccent).Render("LIVE") + sep +
			lipgloss.NewStyle().Foreground(colorMuted).Render("next in "+mmss(secs))
	case screenGames:
		ctx = lipgloss.NewStyle().Foreground(colorMuted).Render(fmt.Sprintf("%d games", len(m.games)))
	case screenDiscussions:
		if b := m.currentBranchPtr(); b != nil {
			totalCommits, totalLikes, totalComments := repoTotals(m.branches)
			ctx = lipgloss.NewStyle().Foreground(colorOk).Render("on "+b.Name) + sep +
				lipgloss.NewStyle().Foreground(colorMuted).Render(fmt.Sprintf("%d commits", totalCommits)) + sep +
				likeStyle.Render(fmt.Sprintf("♥ %d", totalLikes)) + sep +
				commentCountStyle.Render(fmt.Sprintf("💬 %d", totalComments))
		}
	}

	left := title + sep + screenTag
	if ctx != "" {
		left += sep + ctx
	}

	right := lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
		Render(time.Now().Format("Mon 15:04:05"))

	shellW := feedShellWidth(m.width)
	gap := shellW - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	inner := left + strings.Repeat(" ", gap) + right
	return lipgloss.PlaceHorizontal(m.width, lipgloss.Center, inner)
}

func (m model) currentBranchPtr() *Branch {
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

// ============================================================================
// Footer (per-screen keybinds)
// ============================================================================

type keyHint struct{ k, d string }

func (m model) renderFooter() string {
	keys := m.footerKeys()
	var parts []string
	for _, k := range keys {
		parts = append(parts, helpKeyStyle.Render(k.k)+" "+helpDescStyle.Render(k.d))
	}
	help := strings.Join(parts, "  ·  ")

	flash := ""
	if m.flash != "" {
		flash = lipgloss.NewStyle().Foreground(colorOk).Render("  " + m.flash)
	}
	inner := statusStyle.Render(help) + flash
	return lipgloss.PlaceHorizontal(m.width, lipgloss.Center, inner)
}

func (m model) footerKeys() []keyHint {
	switch m.screen {
	case screenHome:
		return []keyHint{
			{"1-6", "jump"}, {"hjkl", "move"}, {"enter", "open"}, {"q", "quit"},
		}
	case screenChat:
		if m.chatInputActive {
			return []keyHint{
				{"enter", "send"}, {"ctrl+enter", "newline"}, {"esc", "cancel"},
			}
		}
		if m.chatActive >= 0 {
			return []keyHint{
				{"i", "type"}, {"j/k", "scroll"}, {"G", "newest"}, {"esc", "lobby"}, {"H", "home"},
			}
		}
		return []keyHint{
			{"j/k", "move"}, {"enter", "join"}, {"n", "new"}, {"esc", "home"}, {"q", "quit"},
		}
	case screenNews:
		return []keyHint{
			{"j/k", "move"}, {"enter", "open"}, {"y", "copy url"}, {"esc", "home"}, {"q", "quit"},
		}
	case screenResources:
		if m.resourcesQueryActive {
			return []keyHint{
				{"type", "filter"}, {"enter", "submit"}, {"esc", "back"},
			}
		}
		return []keyHint{
			{"tab", "tabs"}, {"j/k", "move"}, {"enter", "open"}, {"/", "search"}, {"esc", "home"},
		}
	case screenSpotlight:
		if m.spotlightInputActive {
			return []keyHint{
				{"enter", "send"}, {"ctrl+enter", "newline"}, {"esc", "cancel"},
			}
		}
		return []keyHint{
			{"i", "chat"}, {"j/k", "scroll"}, {"o", "open repo"}, {"esc", "home"}, {"q", "quit"},
		}
	case screenGames:
		if m.gameState == gameStateBugHunter {
			return []keyHint{
				{"0-9", "type"}, {"enter", "guess"}, {"r", "reset"}, {"esc", "back"},
			}
		}
		return []keyHint{
			{"j/k", "move"}, {"enter", "play"}, {"esc", "home"}, {"q", "quit"},
		}
	case screenDiscussions:
		switch m.mode {
		case modeCompose:
			return []keyHint{
				{"enter", "push"}, {"ctrl+enter", "newline"}, {"esc", "cancel"},
			}
		case modeDetails:
			if m.commentInput.Focused() {
				return []keyHint{
					{"enter", "post"}, {"ctrl+enter", "newline"}, {"esc", "back to thread"},
				}
			}
			return []keyHint{
				{"j/k", "select"}, {"l", "like"}, {"r", "reply"}, {"esc", "close"},
			}
		case modeBranchPicker:
			return []keyHint{
				{"j/k", "move"}, {"enter", "checkout"}, {"esc", "cancel"},
			}
		}
		return []keyHint{
			{"n", "new"}, {"enter", "open"}, {"l", "like"}, {"j/k", "move"},
			{"tab", "next branch"}, {"b", "branches"}, {"esc", "home"},
		}
	}
	return nil
}

// ============================================================================
// Discussions screen rendering
// ============================================================================

func (m model) renderDiscussionsBody(termW, bodyH int) string {
	switch m.mode {
	case modeCompose, modeDetails:
		return m.renderPopup(termW, bodyH)
	case modeBranchPicker:
		return m.renderBranchPickerOverlay(termW, bodyH)
	}
	return m.renderDiscussionsMain(termW, bodyH)
}

func (m model) renderDiscussionsMain(termW, bodyH int) string {
	shellW := feedShellWidth(termW)
	contentW := shellW - 2

	tabs := m.renderTabs(contentW)
	tabsH := lipgloss.Height(tabs)

	feedH := bodyH - tabsH - 1
	if feedH < 4 {
		feedH = 4
	}
	feed := m.renderFeed(contentW, feedH)

	col := lipgloss.JoinVertical(lipgloss.Left, tabs, "", feed)
	return lipgloss.Place(termW, bodyH, lipgloss.Center, lipgloss.Top, col)
}

func (m model) renderTabs(width int) string {
	visible := m.visibleTabBranches()
	var tabs []string
	for i, idx := range visible {
		label := fmt.Sprintf("%d %s", i+1, m.branches[idx].Name)
		if idx == m.branchIdx {
			tabs = append(tabs, tabActiveStyle.Render(label))
		} else {
			tabs = append(tabs, tabInactiveStyle.Render(label))
		}
	}
	more := tabMoreStyle.Render(fmt.Sprintf("b  more (%d)", len(m.branches)))
	tabs = append(tabs, more)

	row := strings.Join(tabs, "  ")
	rowW := lipgloss.Width(row)
	if rowW < width {
		row = lipgloss.PlaceHorizontal(width, lipgloss.Left, row)
	}
	return row
}

func (m model) renderBranchPickerOverlay(width, height int) string {
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

	title := lipgloss.NewStyle().Foreground(colorAccent2).Bold(true).Render("checkout a branch")
	rule := lipgloss.NewStyle().Foreground(colorBorderLo).Render(strings.Repeat("─", contentW))

	var rows []string
	for i, b := range m.branches {
		marker := "  "
		if i == m.branchPickerIdx {
			marker = "▸ "
		}
		label := fmt.Sprintf("%s%-30s %d", marker, truncate(b.Name, 30), len(b.Commits))
		if i == m.branchPickerIdx {
			rows = append(rows, branchItemSelStyle.Width(contentW).Render(label))
		} else if i == m.branchIdx {
			rows = append(rows, lipgloss.NewStyle().Foreground(colorOk).Render(label))
		} else {
			rows = append(rows, branchItemStyle.Render(label))
		}
	}
	hint := helpDescStyle.Render("j/k move   ·   enter checkout   ·   esc cancel")

	inner := lipgloss.JoinVertical(lipgloss.Left,
		title,
		rule,
		strings.Join(rows, "\n"),
		rule,
		hint,
	)
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccent).
		Padding(1, 2).
		Render(inner)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

var logoLines = []string{
	" ██████╗██╗  ██╗ █████╗  ██████╗ ███████╗██████╗ ██╗   ██╗████████╗███████╗",
	"██╔════╝██║  ██║██╔══██╗██╔═══██╗██╔════╝██╔══██╗╚██╗ ██╔╝╚══██╔══╝██╔════╝",
	"██║     ███████║███████║██║   ██║███████╗██████╔╝ ╚████╔╝    ██║   █████╗  ",
	"██║     ██╔══██║██╔══██║██║   ██║╚════██║██╔══██╗  ╚██╔╝     ██║   ██╔══╝  ",
	"╚██████╗██║  ██║██║  ██║╚██████╔╝███████║██████╔╝   ██║      ██║   ███████╗",
	" ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═════╝    ╚═╝      ╚═╝   ╚══════╝",
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

func (m model) renderFeed(width, height int) string {
	b := m.currentBranchPtr()
	if b == nil || len(b.Commits) == 0 {
		return padToHeight(statusStyle.Render("no commits yet — press 'n' to compose one"), height)
	}

	title := titleStyle.Render("feed: " + b.Name)

	var blocks []string
	var startLines []int
	line := 0
	for i, c := range b.Commits {
		focused := i == m.commitIdx
		block := m.renderCommit(c, focused, width)
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
			bodyParts = append(bodyParts, dividerLine(width))
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

	indicator := strings.Repeat(" ", width)
	if len(bodyLines) > visibleH {
		pct := 100
		if len(bodyLines)-visibleH > 0 {
			pct = scroll * 100 / (len(bodyLines) - visibleH)
		}
		indicator = lipgloss.NewStyle().
			Foreground(colorMuted).
			Width(width).
			Align(lipgloss.Right).
			Render(fmt.Sprintf("scroll %d%%", pct))
	}
	parts := []string{title, indicator, "", visible}
	return padToHeight(strings.Join(parts, "\n"), height)
}

func dividerLine(width int) string {
	return lipgloss.NewStyle().Foreground(colorBorderLo).Render(strings.Repeat("─", width))
}

func padToHeight(s string, h int) string {
	if h <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	if len(lines) >= h {
		return strings.Join(lines[:h], "\n")
	}
	return s + strings.Repeat("\n", h-len(lines))
}

func (m model) peekCommit() *Commit {
	b := m.currentBranchPtr()
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
	const chrome = 4
	bodyW := width - chrome
	if bodyW < 8 {
		bodyW = 8
	}

	header := fmt.Sprintf("%s  %s  %s",
		commitAuthorStyle.Render(c.Author),
		commitTimeStyle.Render(humanizeTime(c.At)),
		commitSHAStyle.Render(c.SHA),
	)

	wrapped := wrap(c.Message, bodyW)
	msg := commitMsgStyle.Render(wrapped)

	likeMark := likeStyle.Render("♥")
	if c.Liked {
		likeMark = likedStyle.Render("♥")
	}
	stats := fmt.Sprintf("%s %d   %s %d",
		likeMark, c.Likes,
		commentCountStyle.Render("💬"), len(c.Comments),
	)

	content := strings.Join([]string{header, "", msg, "", stats}, "\n")

	box := lipgloss.NewStyle().Padding(0, 1).Width(bodyW)
	if selected {
		box = box.Border(lipgloss.RoundedBorder()).BorderForeground(colorAccent)
	} else {
		box = box.Border(lipgloss.HiddenBorder())
	}
	return box.Render(content)
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
	return m.renderDetailsPopup(width, height)
}

func (m model) renderComposeScreen(width, height int) string {
	logo := m.renderLogo()
	logoH := lipgloss.Height(logo)

	popupW, _ := popupSize(width, height)

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
	if b := m.currentBranchPtr(); b != nil {
		bn = b.Name
	}
	contentW := popupW - 6
	if contentW < 30 {
		contentW = 30
	}
	title := lipgloss.NewStyle().Foreground(colorAccent2).Bold(true).Render("commit on " + bn)
	m.commitInput.SetWidth(contentW)
	m.commitInput.SetHeight(taH)
	ta := m.commitInput.View()
	hint := helpDescStyle.Render("enter push   ·   ctrl+enter newline   ·   esc cancel")

	rule := lipgloss.NewStyle().Foreground(colorBorderLo).Render(strings.Repeat("─", contentW))

	inner := strings.Join([]string{title, rule, ta, rule, hint}, "\n")
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccent).
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

func (m model) renderDetailsPopup(width, height int) string {
	c := m.peekCommit()
	if c == nil {
		return ""
	}

	popupW := width * 80 / 100
	if popupW > 100 {
		popupW = 100
	}
	if popupW < 60 {
		popupW = 60
	}
	contentW := popupW - 6
	if contentW < 30 {
		contentW = 30
	}

	rule := lipgloss.NewStyle().Foreground(colorBorderLo).Render(strings.Repeat("─", contentW))

	postSelected := m.detailsSelIdx == -1
	post := renderPostBlock(c, contentW, postSelected)

	flat := flattenComments(c.Comments, 0)

	target := "the post"
	if postSelected {
		target = c.Author
	} else if m.detailsSelIdx >= 0 && m.detailsSelIdx < len(flat) {
		target = flat[m.detailsSelIdx].c.Author
	}
	var replyHeader string
	if m.commentInput.Focused() {
		replyHeader = lipgloss.NewStyle().Foreground(colorOk).Bold(true).Render("replying to " + target)
	} else {
		replyHeader = lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render("press r to reply to " + target)
	}

	m.commentInput.SetWidth(contentW)
	m.commentInput.SetHeight(3)
	inputView := m.commentInput.View()
	var hint string
	if m.commentInput.Focused() {
		hint = helpDescStyle.Render("enter post   ·   ctrl+enter newline   ·   esc back to thread")
	} else {
		hint = helpDescStyle.Render("j/k select   ·   l like   ·   r reply   ·   esc close")
	}

	top := lipgloss.JoinVertical(lipgloss.Left, post, "", rule)
	bottom := lipgloss.JoinVertical(lipgloss.Left, rule, "", replyHeader, inputView, "", hint)

	const popupBoxOverhead = 4
	availInner := height - popupBoxOverhead
	if availInner < 10 {
		availInner = 10
	}

	topH := lipgloss.Height(top)
	bottomH := lipgloss.Height(bottom)
	middleH := availInner - topH - bottomH
	if middleH < 4 {
		middleH = 4
	}

	var blocks []string
	var startLines []int
	line := 0
	for i, f := range flat {
		selected := i == m.detailsSelIdx
		block := renderCommentBlock(*f.c, f.depth, contentW, selected)
		startLines = append(startLines, line)
		blocks = append(blocks, block)
		line += lipgloss.Height(block)
	}

	var middleStr string
	if len(flat) == 0 {
		middleStr = lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render("no replies yet — be the first.")
	} else {
		middleStr = strings.Join(blocks, "\n")
	}
	middleLines := strings.Split(middleStr, "\n")

	scroll := 0
	if m.detailsSelIdx >= 0 && m.detailsSelIdx < len(blocks) {
		selStart := startLines[m.detailsSelIdx]
		selBlockH := lipgloss.Height(blocks[m.detailsSelIdx])
		selEnd := selStart + selBlockH
		if selEnd > middleH {
			scroll = selEnd - middleH
		}
		if selStart < scroll {
			scroll = selStart
		}
	}
	if scroll < 0 {
		scroll = 0
	}
	if scroll+middleH > len(middleLines) {
		scroll = len(middleLines) - middleH
	}
	if scroll < 0 {
		scroll = 0
	}

	end := scroll + middleH
	if end > len(middleLines) {
		end = len(middleLines)
	}

	var visible string
	if scroll < len(middleLines) {
		visible = strings.Join(middleLines[scroll:end], "\n")
	}
	actualH := lipgloss.Height(visible)
	if actualH < middleH {
		visible += strings.Repeat("\n", middleH-actualH)
	}

	if len(middleLines) > middleH {
		pct := 100
		if denom := len(middleLines) - middleH; denom > 0 {
			pct = scroll * 100 / denom
		}
		indicator := lipgloss.NewStyle().
			Foreground(colorMuted).
			Width(contentW).
			Align(lipgloss.Right).
			Render(fmt.Sprintf("scroll %d%%", pct))
		top = lipgloss.JoinVertical(lipgloss.Left, post, "", rule, indicator)
		topH = lipgloss.Height(top)
		middleH = availInner - topH - bottomH
		if middleH < 4 {
			middleH = 4
		}
		if scroll+middleH > len(middleLines) {
			scroll = len(middleLines) - middleH
		}
		if scroll < 0 {
			scroll = 0
		}
		end = scroll + middleH
		if end > len(middleLines) {
			end = len(middleLines)
		}
		visible = strings.Join(middleLines[scroll:end], "\n")
		actualH = lipgloss.Height(visible)
		if actualH < middleH {
			visible += strings.Repeat("\n", middleH-actualH)
		}
	}

	inner := lipgloss.JoinVertical(lipgloss.Left, top, visible, bottom)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccent).
		Padding(1, 2).
		Render(inner)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

func renderPostBlock(c *Commit, width int, selected bool) string {
	header := fmt.Sprintf("%s  %s  %s",
		commitAuthorStyle.Bold(true).Render(c.Author),
		commitTimeStyle.Render(humanizeTime(c.At)),
		commitSHAStyle.Render(c.SHA),
	)
	body := commitMsgStyle.Render(wrap(c.Message, width-4))
	likeMark := likeStyle.Render("♥")
	if c.Liked {
		likeMark = likedStyle.Render("♥")
	}
	stats := fmt.Sprintf("%s %d   %s %d",
		likeMark, c.Likes,
		commentCountStyle.Render("💬"), len(c.Comments),
	)
	content := strings.Join([]string{header, "", body, "", stats}, "\n")

	box := lipgloss.NewStyle().Padding(0, 1).Width(width - 4)
	if selected {
		return box.Border(lipgloss.RoundedBorder()).BorderForeground(colorAccent).Render(content)
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
		commentAuthorStyle.Render(cm.Author),
		commitTimeStyle.Render(humanizeTime(cm.At)),
	)
	body := commentBodyStyle.Render(wrap(cm.Body, innerW))
	likeMark := likeStyle.Render("♥")
	if cm.Liked {
		likeMark = likedStyle.Render("♥")
	}
	stats := fmt.Sprintf("%s %d", likeMark, cm.Likes)
	if len(cm.Comments) > 0 {
		stats += "   " + commentCountStyle.Render("↳") + lipgloss.NewStyle().Foreground(colorMuted).Render(fmt.Sprintf(" %d", len(cm.Comments)))
	}
	content := strings.Join([]string{header, body, stats}, "\n")

	box := lipgloss.NewStyle().Padding(0, 1).Width(innerW)
	if selected {
		box = box.Border(lipgloss.RoundedBorder()).BorderForeground(colorAccent)
	} else {
		box = box.Border(lipgloss.HiddenBorder())
	}
	return lipgloss.NewStyle().PaddingLeft(len(indent)).Render(box.Render(content))
}

// ============================================================================
// Helpers
// ============================================================================

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
