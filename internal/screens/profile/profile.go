// Package profile renders a user's GitHub-flavored MySpace profile:
// cached GitHub data (bio, repos, stars, contribution graph) on top, then
// vibespace-native sections (posts, friends, guestbook) below.
//
// The screen is read-only — it never mutates state. Friend requests, posts,
// and guestbook signing all happen via slash commands in the lobby. This
// keeps the screen simple (no input handling, no command palette) and the
// state machine for those flows lives in one place.
//
// Layout: sections are stacked as centered cards capped at maxCardWidth. The
// focused card gets heavy borders + accent color + a one-line drop shadow,
// giving a subtle 3D "lifted" feel. Tab / j/k cycle focus; the viewport
// auto-scrolls to keep the focused card visible.
package profile

import (
	"fmt"
	"strings"
	"time"

	"github.com/bigboggy/vibespace/internal/screens"
	"github.com/bigboggy/vibespace/internal/store"
	"github.com/bigboggy/vibespace/internal/theme"
	"github.com/bigboggy/vibespace/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	maxCardWidth = 88
	minCardWidth = 48

	repoTileWidth  = 20
	repoTileHeight = 4
	repoTileGap    = 1
	repoFetchLimit = 8
)

// Screen is the profile view. Target is the user being viewed; Viewer is the
// session's current user (may be empty for unauthenticated guests).
type Screen struct {
	styles *theme.Styles
	data   *store.Store

	target string
	viewer string

	focused int // index into sections()
	scroll  int // line offset into the rendered stack

	// Repo grid state. repoCols is cached from the last render so Update can
	// compute row jumps without knowing the current width.
	repoCursor int
	repoCount  int
	repoCols   int
}

// New returns an empty profile screen. The target is set via SetTarget before
// the screen is activated.
func New(styles *theme.Styles, data *store.Store) *Screen {
	return &Screen{styles: styles, data: data}
}

// SetTarget configures which profile to view and from whose perspective.
// Called by the app router before switching to this screen.
func (s *Screen) SetTarget(target, viewer string) {
	s.target = strings.TrimPrefix(target, "@")
	s.viewer = strings.TrimPrefix(viewer, "@")
	s.focused = 0
	s.scroll = 0
	s.repoCursor = 0
}

// Target returns the gh_login of the currently-viewed profile.
func (s *Screen) Target() string { return s.target }

// Viewer returns the gh_login of the current session user.
func (s *Screen) Viewer() string { return s.viewer }

// Screen interface ──────────────────────────────────────────────────────────

func (s *Screen) Init() tea.Cmd         { return nil }
func (s *Screen) Name() string          { return screens.ProfileID }
func (s *Screen) Title() string         { return "profile" }
func (s *Screen) InputFocused() bool    { return false }

func (s *Screen) HeaderContext() string {
	if s.target == "" {
		return ""
	}
	secs := s.sections()
	if len(secs) == 0 {
		return "@" + s.target
	}
	idx := s.focused
	if idx >= len(secs) {
		idx = len(secs) - 1
	}
	return fmt.Sprintf("@%s · %s (%d/%d)", s.target, strings.ToLower(secs[idx].title), idx+1, len(secs))
}

func (s *Screen) Footer() []screens.KeyHint {
	if s.inRepoGrid() {
		return []screens.KeyHint{
			{Key: "←↑→↓", Desc: "select repo"},
			{Key: "tab", Desc: "next section"},
			{Key: "esc", Desc: "lobby"},
		}
	}
	return []screens.KeyHint{
		{Key: "tab/j/k", Desc: "section"},
		{Key: "g/G", Desc: "first/last"},
		{Key: "esc", Desc: "lobby"},
	}
}

// inRepoGrid is true when the REPOS section is the focused one and there are
// repos to navigate. Used to switch input semantics: arrow keys move within
// the tile grid instead of switching sections.
func (s *Screen) inRepoGrid() bool {
	secs := s.sections()
	if s.focused >= len(secs) {
		return false
	}
	return secs[s.focused].id == "repos" && s.repoCount > 0
}

func (s *Screen) Update(msg tea.Msg) (screens.Screen, tea.Cmd) {
	switch m := msg.(type) {
	case tea.KeyMsg:
		n := len(s.sections())
		if n == 0 {
			return s, nil
		}
		inGrid := s.inRepoGrid()
		switch m.String() {
		case "tab":
			s.focused = (s.focused + 1) % n
		case "j", "down":
			if !inGrid || !s.gridDown() {
				s.focused = (s.focused + 1) % n
			}
		case "shift+tab":
			s.focused = (s.focused - 1 + n) % n
		case "k", "up":
			if !inGrid || !s.gridUp() {
				s.focused = (s.focused - 1 + n) % n
			}
		case "h", "left":
			if inGrid {
				s.gridHoriz(-1)
			}
		case "l", "right":
			if inGrid {
				s.gridHoriz(1)
			}
		case "g", "home":
			if inGrid {
				s.repoCursor = 0
			} else {
				s.focused = 0
			}
		case "G", "end":
			if inGrid {
				s.repoCursor = s.repoCount - 1
			} else {
				s.focused = n - 1
			}
		case "pgdown":
			s.scroll += 10
		case "pgup":
			s.scroll -= 10
			if s.scroll < 0 {
				s.scroll = 0
			}
		}
	}
	return s, nil
}

// gridHoriz shifts the repo cursor by one within the grid, clamping at the
// ends. Horizontal motion never escapes the section — that's what tab is for.
func (s *Screen) gridHoriz(delta int) {
	if s.repoCount == 0 {
		return
	}
	next := s.repoCursor + delta
	if next < 0 || next >= s.repoCount {
		return
	}
	s.repoCursor = next
}

// gridUp moves the cursor up one row. Returns false when already on the top
// row — the caller treats that as a signal to escape to the previous section,
// so up-arrow doesn't trap the user inside the grid.
func (s *Screen) gridUp() bool {
	if s.repoCount == 0 || s.repoCols == 0 {
		return false
	}
	if s.repoCursor < s.repoCols {
		return false
	}
	s.repoCursor -= s.repoCols
	return true
}

// gridDown moves the cursor down one row. Returns false when already on the
// bottom row so the caller escapes to the next section. When the target row
// has fewer tiles than the current column, snap to the last tile in that row.
func (s *Screen) gridDown() bool {
	if s.repoCount == 0 || s.repoCols == 0 {
		return false
	}
	lastRow := (s.repoCount - 1) / s.repoCols
	if s.repoCursor/s.repoCols >= lastRow {
		return false
	}
	next := s.repoCursor + s.repoCols
	if next >= s.repoCount {
		next = s.repoCount - 1
	}
	s.repoCursor = next
	return true
}

// View renders the profile body within (width, height).
func (s *Screen) View(width, height int) string {
	if s.target == "" {
		return s.placeholder(width, height, "no profile selected")
	}

	cardW := width - 4
	if cardW > maxCardWidth {
		cardW = maxCardWidth
	}
	if cardW < minCardWidth {
		cardW = minCardWidth
	}
	innerW := cardW - 4 // borders + padding

	secs := s.sections()
	if s.focused >= len(secs) {
		s.focused = len(secs) - 1
	}

	// Render each section as a card and track vertical offsets so we can
	// auto-scroll to keep the focused card visible.
	cards := make([]string, len(secs))
	offsets := make([]int, len(secs))
	heights := make([]int, len(secs))
	const gap = 1 // blank line between cards

	cursor := 1 // 1-row top padding so focused shadows don't clip the chrome
	for i, sec := range secs {
		body := sec.body(innerW)
		card := s.renderCard(sec.title, body, cardW, i == s.focused)
		cards[i] = card
		offsets[i] = cursor
		h := lipgloss.Height(card)
		heights[i] = h
		cursor += h + gap
	}

	// Auto-scroll: keep the focused card fully visible. When the card is
	// taller than the viewport we align its top.
	s.scroll = clampScroll(s.scroll, offsets[s.focused], heights[s.focused], cursor, height)

	// Compose the stack with leading top padding, then crop to the scroll
	// window and pad to exact height.
	stack := strings.Repeat("\n", 1) + strings.Join(cards, strings.Repeat("\n", gap+1))
	cropped := cropLines(stack, s.scroll, height)

	// Indent each line to center the cards within `width`.
	leftPad := (width - cardW) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	cropped = indent(cropped, leftPad)
	return ui.PadToHeight(cropped, height)
}

// ── Sections ────────────────────────────────────────────────────────────────

type section struct {
	id    string
	title string
	body  func(innerW int) string
}

func (s *Screen) sections() []section {
	return []section{
		{"about", "ABOUT", s.renderAboutBody},
		{"contrib", "CONTRIBUTIONS", s.renderContributionsBody},
		{"posts", "POSTS", s.renderPostsBody},
		{"repos", "REPOSITORIES", s.renderReposBody},
		{"stars", "STARRED", s.renderStarsBody},
		{"friends", "FRIENDS", s.renderFriendsBody},
		{"guestbook", "GUESTBOOK", s.renderGuestbookBody},
	}
}

func (s *Screen) renderAboutBody(width int) string {
	st := s.styles
	u, ok, _ := s.data.User(s.target)

	name := st.NewStyle().Foreground(st.Accent).Bold(true).Render("@" + s.target)
	var realName string
	if ok && u.Name != "" {
		realName = st.NewStyle().Foreground(st.Fg).Render("  " + u.Name)
	}

	bio := ""
	if ok && u.Bio != "" {
		bio = "\n" + ui.Wrap(u.Bio, width)
	}

	meta := []string{}
	if ok && u.Location != "" {
		meta = append(meta, "📍 "+u.Location)
	}
	if ok && u.Company != "" {
		meta = append(meta, "🏢 "+u.Company)
	}
	metaLine := ""
	if len(meta) > 0 {
		metaLine = "\n" + st.NewStyle().Foreground(st.Muted).Render(strings.Join(meta, "   "))
	}

	friends, _ := s.data.Friends(s.target)
	starCount, _ := s.data.StarsCount(s.target)
	stats := []string{
		statBlock(st, "followers", u.Followers),
		statBlock(st, "following", u.Following),
		statBlock(st, "repos", u.PublicRepos),
		statBlock(st, "stars", starCount),
		statBlock(st, "friends", len(friends)),
	}
	statLine := lipgloss.JoinHorizontal(lipgloss.Top, stats...)

	pill := s.renderFriendPill()

	out := name + realName + bio + metaLine + "\n\n" + statLine
	if pill != "" {
		out += "\n\n" + pill
	}
	return out
}

func statBlock(st *theme.Styles, label string, n int) string {
	num := st.NewStyle().Foreground(st.Fg).Bold(true).Render(fmt.Sprintf("%d", n))
	lbl := st.NewStyle().Foreground(st.Muted).Render(" " + label)
	return st.NewStyle().Padding(0, 2, 0, 0).Render(num + lbl)
}

func (s *Screen) renderFriendPill() string {
	st := s.styles
	if s.viewer == "" {
		return st.NewStyle().Foreground(st.Muted).Italic(true).
			Render("(sign in with /auth to send a friend request)")
	}
	status, _ := s.data.FriendStatusBetween(s.viewer, s.target)
	switch status {
	case store.FriendSelf:
		return st.NewStyle().Foreground(st.Muted).Italic(true).
			Render("(this is you — /post to write, /friends to see your network)")
	case store.FriendAccepted:
		return st.NewStyle().Foreground(st.OK).Bold(true).Render("● friends") +
			st.NewStyle().Foreground(st.Muted).Render("   /unfriend @"+s.target+" to remove")
	case store.FriendPendingOut:
		return st.NewStyle().Foreground(st.Warn).Bold(true).Render("○ request sent") +
			st.NewStyle().Foreground(st.Muted).Render("   waiting on @"+s.target)
	case store.FriendPendingIn:
		return st.NewStyle().Foreground(st.Like).Bold(true).Render("○ wants to be friends") +
			st.NewStyle().Foreground(st.Muted).Render("   /accept @"+s.target+" or /reject @"+s.target)
	default:
		return st.NewStyle().Foreground(st.Accent).Render("+ /friend @"+s.target) +
			st.NewStyle().Foreground(st.Muted).Render("   send a friend request")
	}
}

func (s *Screen) renderContributionsBody(width int) string {
	st := s.styles
	to := time.Now()
	from := to.AddDate(-1, 0, 0)
	days, _ := s.data.Contributions(s.target, from, to)
	if len(days) == 0 {
		return st.NewStyle().Foreground(st.Muted).
			Render("no contribution data yet — /auth syncs from GitHub")
	}

	byDate := make(map[string]int, len(days))
	total := 0
	for _, d := range days {
		byDate[d.Date.Format("2006-01-02")] = d.Count
		total += d.Count
	}

	start := from
	for start.Weekday() != time.Sunday {
		start = start.AddDate(0, 0, -1)
	}
	maxCols := width
	if maxCols < 10 {
		maxCols = 10
	}
	totalWeeks := int(to.Sub(start).Hours()/24)/7 + 1
	if totalWeeks > maxCols {
		start = start.AddDate(0, 0, (totalWeeks-maxCols)*7)
		totalWeeks = maxCols
	}

	rows := make([]string, 7)
	for d := 0; d < 7; d++ {
		var b strings.Builder
		for w := 0; w < totalWeeks; w++ {
			day := start.AddDate(0, 0, w*7+d)
			if day.After(to) {
				b.WriteString(" ")
				continue
			}
			b.WriteString(contribCell(st, byDate[day.Format("2006-01-02")]))
		}
		rows[d] = b.String()
	}

	footer := st.NewStyle().Foreground(st.Muted).
		Render(fmt.Sprintf("%d contributions in the last year", total))
	return strings.Join(rows, "\n") + "\n\n" + footer
}

func contribCell(st *theme.Styles, n int) string {
	var color lipgloss.Color
	switch {
	case n == 0:
		color = st.BorderLo
	case n < 3:
		color = lipgloss.Color("#0e4429")
	case n < 6:
		color = lipgloss.Color("#006d32")
	case n < 10:
		color = lipgloss.Color("#26a641")
	default:
		color = lipgloss.Color("#39d353")
	}
	return st.NewStyle().Foreground(color).Render("■")
}

func (s *Screen) renderPostsBody(width int) string {
	st := s.styles
	posts, _ := s.data.PostsByAuthor(s.target, 5)
	if len(posts) == 0 {
		hint := "nothing posted yet"
		if s.viewer == s.target {
			hint += " — type /post <message> to write your first one"
		}
		return st.NewStyle().Foreground(st.Muted).Render(hint)
	}
	var blocks []string
	for _, p := range posts {
		when := st.NewStyle().Foreground(st.Muted).Italic(true).
			Render(ui.HumanizeTime(p.CreatedAt))
		body := ui.Wrap(p.Body, width)
		blocks = append(blocks, when+"\n"+body)
	}
	return strings.Join(blocks, "\n\n")
}

func (s *Screen) renderReposBody(width int) string {
	st := s.styles
	repos, _ := s.data.TopRepos(s.target, repoFetchLimit)
	s.repoCount = len(repos)
	if len(repos) == 0 {
		s.repoCols = 0
		return st.NewStyle().Foreground(st.Muted).
			Render("no repos cached yet — /auth syncs from GitHub")
	}

	cols := (width + repoTileGap) / (repoTileWidth + repoTileGap)
	if cols < 1 {
		cols = 1
	}
	s.repoCols = cols
	if s.repoCursor >= len(repos) {
		s.repoCursor = len(repos) - 1
	}

	// Selection styling only when the REPOS section itself has focus —
	// otherwise the cursor is "remembered" but not visible.
	sectionFocused := s.sections()[s.focused].id == "repos"

	var rows []string
	for rowStart := 0; rowStart < len(repos); rowStart += cols {
		end := rowStart + cols
		if end > len(repos) {
			end = len(repos)
		}
		var tiles []string
		for i := rowStart; i < end; i++ {
			tiles = append(tiles, s.renderRepoTile(repos[i], sectionFocused && i == s.repoCursor))
		}
		rows = append(rows, joinTilesHoriz(tiles, repoTileGap))
	}
	grid := strings.Join(rows, "\n")

	// Detail panel: description + clickable URL (OSC8) for the selected
	// repo. Terminals that don't support OSC8 render the URL as plain text.
	sel := repos[s.repoCursor]
	url := "https://github.com/" + s.target + "/" + sel.Name
	urlText := st.NewStyle().Foreground(st.Accent).Underline(true).Render(url)
	urlLine := hyperlink(url, urlText)
	if !sectionFocused {
		urlLine = st.NewStyle().Foreground(st.Muted).Render("select to open: ") + urlLine
	} else {
		urlLine = st.NewStyle().Foreground(st.OK).Render("open ▸ ") + urlLine
	}

	detail := ""
	if sel.Description != "" {
		detail = st.NewStyle().Foreground(st.Muted).Render(ui.Wrap(sel.Description, width)) + "\n"
	}
	detail += urlLine
	return grid + "\n\n" + detail
}

// renderRepoTile renders one repo as a small fixed-size card. Selected tiles
// get a thick accent border so they pop above the rest of the grid; unselected
// tiles use a thin rounded border in the low-contrast color.
func (s *Screen) renderRepoTile(r store.Repo, selected bool) string {
	st := s.styles

	border := lipgloss.RoundedBorder()
	bColor := st.BorderLo
	nameColor := st.Fg
	if selected {
		border = lipgloss.ThickBorder()
		bColor = st.Accent
		nameColor = st.Accent
	}

	innerW := repoTileWidth - 4
	name := ui.Truncate(r.Name, innerW)
	nameStyled := st.NewStyle().Foreground(nameColor).Bold(true).Render(name)

	stars := fmt.Sprintf("★ %d", r.Stars)
	starsStyled := st.NewStyle().Foreground(st.Like).Render(stars)

	line2 := starsStyled
	if r.Language != "" {
		room := innerW - lipgloss.Width(stars) - 2
		if room > 0 {
			lang := ui.Truncate(r.Language, room)
			line2 += "  " + st.NewStyle().Foreground(st.Muted).Render(lang)
		}
	}

	body := nameStyled + "\n" + line2
	return st.NewStyle().
		Border(border).
		BorderForeground(bColor).
		Padding(0, 1).
		Width(repoTileWidth - 2).
		Height(repoTileHeight - 2).
		Render(body)
}

// joinTilesHoriz joins multi-line tiles side-by-side with a `gap`-col spacer.
// lipgloss.JoinHorizontal handles row alignment; the spacer is a single-row
// space that gets vertically extended to the tile height.
func joinTilesHoriz(tiles []string, gap int) string {
	if len(tiles) == 0 {
		return ""
	}
	spacer := strings.Repeat(" ", gap)
	parts := make([]string, 0, len(tiles)*2-1)
	for i, t := range tiles {
		if i > 0 {
			parts = append(parts, spacer)
		}
		parts = append(parts, t)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

// hyperlink wraps text with OSC8 escape codes so terminals that support
// clickable links (iTerm2, kitty, wezterm, recent gnome-terminal, etc.) make
// the text click-to-open. Others render the text plain.
func hyperlink(url, text string) string {
	return "\x1b]8;;" + url + "\x1b\\" + text + "\x1b]8;;\x1b\\"
}

func (s *Screen) renderStarsBody(width int) string {
	st := s.styles
	_ = width
	stars, _ := s.data.TopStars(s.target, 5)
	if len(stars) == 0 {
		return st.NewStyle().Foreground(st.Muted).Render("no stars cached yet")
	}
	var rows []string
	for _, r := range stars {
		name := st.NewStyle().Foreground(st.Accent).Render(r.FullName)
		count := st.NewStyle().Foreground(st.Like).Render(fmt.Sprintf("★ %d", r.Stars))
		rows = append(rows, name+"  "+count)
	}
	return strings.Join(rows, "\n")
}

func (s *Screen) renderFriendsBody(width int) string {
	st := s.styles
	_ = width
	friends, _ := s.data.Friends(s.target)
	if len(friends) == 0 {
		return st.NewStyle().Foreground(st.Muted).
			Render("no friends yet — be the first to /friend @" + s.target)
	}
	limit := 12
	if len(friends) < limit {
		limit = len(friends)
	}
	var chips []string
	for _, f := range friends[:limit] {
		chips = append(chips, st.NewStyle().Foreground(st.Accent2).Render("@"+f.Login))
	}
	body := strings.Join(chips, "  ")
	if len(friends) > limit {
		body += st.NewStyle().Foreground(st.Muted).
			Render(fmt.Sprintf("   +%d more", len(friends)-limit))
	}
	return body
}

func (s *Screen) renderGuestbookBody(width int) string {
	st := s.styles
	entries, _ := s.data.Guestbook(s.target, 8)
	if len(entries) == 0 {
		return st.NewStyle().Foreground(st.Muted).
			Render("empty — friends can /sign @" + s.target + " <message>")
	}
	var blocks []string
	for _, e := range entries {
		who := st.NewStyle().Foreground(st.Accent2).Render("@" + e.Author)
		when := st.NewStyle().Foreground(st.Muted).Italic(true).
			Render("  " + ui.HumanizeTime(e.CreatedAt))
		body := ui.Wrap(e.Body, width-2)
		blocks = append(blocks, who+when+"\n  "+
			strings.ReplaceAll(body, "\n", "\n  "))
	}
	return strings.Join(blocks, "\n\n")
}

// ── Card rendering ──────────────────────────────────────────────────────────

// renderCard wraps body content in a bordered box with the title embedded in
// the top edge. Focused cards get heavy/double-strike borders in the accent
// color plus a low-contrast drop-shadow line below, creating a subtle 3D
// lifted feel. Unfocused cards use a thin rounded border in the low-contrast
// color so they recede into the background.
func (s *Screen) renderCard(title, body string, width int, focused bool) string {
	st := s.styles

	var (
		horiz, vert            string
		tl, tr, bl, br         string
		bColor                 lipgloss.Color
		titleColor             lipgloss.Color
		leftBracket, rightBkt  string
	)
	if focused {
		horiz, vert = "━", "┃"
		tl, tr, bl, br = "┏", "┓", "┗", "┛"
		leftBracket, rightBkt = "┫ ", " ┣"
		bColor = st.BorderHi
		titleColor = st.Accent
	} else {
		horiz, vert = "─", "│"
		tl, tr, bl, br = "╭", "╮", "╰", "╯"
		leftBracket, rightBkt = "┤ ", " ├"
		bColor = st.BorderLo
		titleColor = st.Muted
	}
	border := st.NewStyle().Foreground(bColor)
	titleStyled := st.NewStyle().Foreground(titleColor).Bold(true).Render(title)
	chip := border.Render(leftBracket) + titleStyled + border.Render(rightBkt)
	chipW := lipgloss.Width(chip)

	leadDash := 2
	trailDash := width - 2 - leadDash - chipW
	if trailDash < 1 {
		trailDash = 1
	}
	top := border.Render(tl+strings.Repeat(horiz, leadDash)) +
		chip +
		border.Render(strings.Repeat(horiz, trailDash)+tr)

	// Body rows: empty padding row on top + content + empty padding row on
	// bottom. Each content line is right-padded to innerW so the right
	// border lines up.
	innerW := width - 4 // 2 for borders + 2 for padding (1 each side)
	rows := []string{padRow(border.Render(vert), innerW)}
	for _, line := range strings.Split(body, "\n") {
		w := lipgloss.Width(line)
		padR := innerW - w
		if padR < 0 {
			padR = 0
		}
		rows = append(rows, border.Render(vert)+" "+line+strings.Repeat(" ", padR)+" "+border.Render(vert))
	}
	rows = append(rows, padRow(border.Render(vert), innerW))

	bottom := border.Render(bl + strings.Repeat(horiz, width-2) + br)

	out := top + "\n" + strings.Join(rows, "\n") + "\n" + bottom
	if focused {
		shadow := st.NewStyle().Foreground(st.BorderLo).Render(strings.Repeat("▀", width))
		out += "\n " + shadow
	}
	return out
}

func padRow(vert string, innerW int) string {
	return vert + strings.Repeat(" ", innerW+2) + vert
}

// ── helpers ────────────────────────────────────────────────────────────────

func (s *Screen) placeholder(width, height int, msg string) string {
	st := s.styles
	return st.Place(width, height, lipgloss.Center, lipgloss.Center,
		st.NewStyle().Foreground(st.Muted).Render(msg))
}

// clampScroll keeps the focused card visible inside the viewport. If the card
// is taller than the viewport we align to its top so the title chip is always
// in view.
func clampScroll(scroll, focusTop, focusHeight, total, height int) int {
	if height <= 0 {
		return 0
	}
	if focusHeight >= height || scroll > focusTop {
		scroll = focusTop
	}
	if focusTop+focusHeight > scroll+height {
		scroll = focusTop + focusHeight - height
	}
	if scroll < 0 {
		scroll = 0
	}
	maxScroll := total - height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}
	return scroll
}

// cropLines trims `body` to the [scroll, scroll+height) row window.
func cropLines(body string, scroll, height int) string {
	lines := strings.Split(body, "\n")
	if scroll >= len(lines) {
		scroll = max(0, len(lines)-1)
	}
	lines = lines[scroll:]
	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

func indent(body string, leftPad int) string {
	if leftPad <= 0 {
		return body
	}
	pad := strings.Repeat(" ", leftPad)
	lines := strings.Split(body, "\n")
	for i, l := range lines {
		lines[i] = pad + l
	}
	return strings.Join(lines, "\n")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
