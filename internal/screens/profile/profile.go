// Package profile renders a user's GitHub-flavored MySpace profile:
// cached GitHub data (bio, repos, stars, contribution graph) on top, then
// vibespace-native sections (posts, friends, guestbook) below.
//
// The screen is read-only — it never mutates state. Friend requests, posts,
// and guestbook signing all happen via slash commands in the lobby. This
// keeps the screen simple (no input handling, no command palette) and the
// state machine for those flows lives in one place.
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

// Screen is the profile view. Target is the user being viewed; Viewer is the
// session's current user (may be empty for unauthenticated guests).
type Screen struct {
	styles *theme.Styles
	data   *store.Store

	target string
	viewer string

	scroll int // line offset into the rendered body
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
	s.scroll = 0
}

// Target returns the gh_login of the currently-viewed profile.
func (s *Screen) Target() string { return s.target }

// Viewer returns the gh_login of the current session user.
func (s *Screen) Viewer() string { return s.viewer }

// Screen interface ──────────────────────────────────────────────────────────

func (s *Screen) Init() tea.Cmd                       { return nil }
func (s *Screen) Name() string                        { return screens.ProfileID }
func (s *Screen) Title() string                       { return "profile" }
func (s *Screen) HeaderContext() string               { return "@" + s.target }
func (s *Screen) InputFocused() bool                  { return false }
func (s *Screen) Footer() []screens.KeyHint {
	return []screens.KeyHint{
		{Key: "j/k", Desc: "scroll"},
		{Key: "esc", Desc: "back to lobby"},
		{Key: "q", Desc: "back"},
	}
}

func (s *Screen) Update(msg tea.Msg) (screens.Screen, tea.Cmd) {
	switch m := msg.(type) {
	case tea.KeyMsg:
		switch m.String() {
		case "j", "down":
			s.scroll++
		case "k", "up":
			if s.scroll > 0 {
				s.scroll--
			}
		case "g", "home":
			s.scroll = 0
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

// View renders the profile body within (width, height).
func (s *Screen) View(width, height int) string {
	if s.target == "" {
		return s.placeholder(width, height, "no profile selected")
	}
	body := s.renderBody(width)
	return s.applyScroll(body, height)
}

// applyScroll trims rendered lines to the scroll window.
func (s *Screen) applyScroll(body string, height int) string {
	lines := strings.Split(body, "\n")
	if s.scroll >= len(lines) {
		s.scroll = max(0, len(lines)-1)
	}
	lines = lines[s.scroll:]
	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

func (s *Screen) placeholder(width, height int, msg string) string {
	st := s.styles
	return st.Place(width, height, lipgloss.Center, lipgloss.Center,
		st.NewStyle().Foreground(st.Muted).Render(msg))
}

// renderBody composes every section top-to-bottom. width includes the body
// gutter; sections render to width directly.
func (s *Screen) renderBody(width int) string {
	w := width - 2
	if w < 40 {
		w = 40
	}
	var sections []string
	sections = append(sections, s.renderHeader(w))
	sections = append(sections, s.renderContributions(w))
	sections = append(sections, s.renderPosts(w))
	sections = append(sections, s.renderRepos(w))
	sections = append(sections, s.renderStars(w))
	sections = append(sections, s.renderFriends(w))
	sections = append(sections, s.renderGuestbook(w))
	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// sectionTitle renders the "── SECTION ───" rule used between sections.
func (s *Screen) sectionTitle(title string, width int) string {
	st := s.styles
	label := st.NewStyle().Foreground(st.Accent2).Bold(true).
		Render(strings.ToUpper(title))
	rule := strings.Repeat("─", max(2, width-lipgloss.Width(label)-3))
	ruleStyled := st.NewStyle().Foreground(st.BorderLo).Render(rule)
	return label + " " + ruleStyled
}

// ── Header ──────────────────────────────────────────────────────────────────

func (s *Screen) renderHeader(width int) string {
	st := s.styles
	u, ok, _ := s.data.User(s.target)

	name := "@" + s.target
	nameStyled := st.NewStyle().Foreground(st.Accent).Bold(true).Render(name)
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

	// Stat strip: followers · following · repos · stars · friends
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

	// Friend status pill between viewer and target.
	pill := s.renderFriendPill()

	header := nameStyled + realName + bio + metaLine + "\n\n" + statLine
	if pill != "" {
		header += "\n\n" + pill
	}
	return header + "\n"
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
		return st.NewStyle().Foreground(st.Accent).Render("+ /friend @" + s.target +
			st.NewStyle().Foreground(st.Muted).Render("   send a friend request"))
	}
}

// ── Contributions ───────────────────────────────────────────────────────────

func (s *Screen) renderContributions(width int) string {
	st := s.styles
	title := s.sectionTitle("contributions", width)

	to := time.Now()
	from := to.AddDate(-1, 0, 0)
	days, _ := s.data.Contributions(s.target, from, to)
	if len(days) == 0 {
		return title + "\n" + st.NewStyle().Foreground(st.Muted).
			Render("  no contribution data yet — /auth syncs from GitHub") + "\n"
	}

	// Bucket: index by date → count.
	byDate := make(map[string]int, len(days))
	total := 0
	for _, d := range days {
		byDate[d.Date.Format("2006-01-02")] = d.Count
		total += d.Count
	}

	// Build a 7-row × N-col grid, with weeks as columns starting from the
	// Sunday at-or-before `from`. Width caps the column count.
	start := from
	for start.Weekday() != time.Sunday {
		start = start.AddDate(0, 0, -1)
	}
	maxCols := width - 4
	if maxCols < 10 {
		maxCols = 10
	}
	totalWeeks := int(to.Sub(start).Hours()/24)/7 + 1
	if totalWeeks > maxCols {
		// Show most-recent weeks only.
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
			count := byDate[day.Format("2006-01-02")]
			b.WriteString(contribCell(st, count))
		}
		rows[d] = b.String()
	}

	footer := st.NewStyle().Foreground(st.Muted).
		Render(fmt.Sprintf("  %d contributions in the last year", total))
	return title + "\n" + strings.Join(rows, "\n") + "\n" + footer + "\n"
}

// contribCell returns a single-cell colored block sized to the contribution count.
func contribCell(st *theme.Styles, n int) string {
	// Five buckets, GitHub-ish: empty, low, mid, high, very high.
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

// ── Posts ───────────────────────────────────────────────────────────────────

func (s *Screen) renderPosts(width int) string {
	st := s.styles
	title := s.sectionTitle("posts", width)
	posts, _ := s.data.PostsByAuthor(s.target, 5)
	if len(posts) == 0 {
		hint := "  nothing posted yet"
		if s.viewer == s.target {
			hint += " — type /post <message> to write your first one"
		}
		return title + "\n" + st.NewStyle().Foreground(st.Muted).Render(hint) + "\n"
	}
	var blocks []string
	for _, p := range posts {
		when := st.NewStyle().Foreground(st.Muted).Italic(true).
			Render(ui.HumanizeTime(p.CreatedAt))
		body := ui.Wrap(p.Body, width-2)
		blocks = append(blocks, "  "+when+"\n  "+
			strings.ReplaceAll(body, "\n", "\n  "))
	}
	return title + "\n" + strings.Join(blocks, "\n\n") + "\n"
}

// ── Repos ───────────────────────────────────────────────────────────────────

func (s *Screen) renderRepos(width int) string {
	st := s.styles
	title := s.sectionTitle("repositories", width)
	repos, _ := s.data.TopRepos(s.target, 6)
	if len(repos) == 0 {
		return title + "\n" + st.NewStyle().Foreground(st.Muted).
			Render("  no repos cached yet — /auth syncs from GitHub") + "\n"
	}
	var rows []string
	for _, r := range repos {
		name := st.NewStyle().Foreground(st.Accent).Bold(true).Render(r.Name)
		stars := st.NewStyle().Foreground(st.Like).Render(fmt.Sprintf("★ %d", r.Stars))
		lang := ""
		if r.Language != "" {
			lang = st.NewStyle().Foreground(st.Muted).Render("  " + r.Language)
		}
		head := name + "  " + stars + lang
		desc := ""
		if r.Description != "" {
			desc = "\n  " + st.NewStyle().Foreground(st.Muted).
				Render(ui.Truncate(r.Description, width-4))
		}
		rows = append(rows, "  "+head+desc)
	}
	return title + "\n" + strings.Join(rows, "\n\n") + "\n"
}

// ── Stars ───────────────────────────────────────────────────────────────────

func (s *Screen) renderStars(width int) string {
	st := s.styles
	title := s.sectionTitle("starred", width)
	stars, _ := s.data.TopStars(s.target, 5)
	if len(stars) == 0 {
		return title + "\n" + st.NewStyle().Foreground(st.Muted).
			Render("  no stars cached yet") + "\n"
	}
	var rows []string
	for _, r := range stars {
		name := st.NewStyle().Foreground(st.Accent).Render(r.FullName)
		count := st.NewStyle().Foreground(st.Like).Render(fmt.Sprintf("★ %d", r.Stars))
		rows = append(rows, "  "+name+"  "+count)
	}
	return title + "\n" + strings.Join(rows, "\n") + "\n"
}

// ── Friends ─────────────────────────────────────────────────────────────────

func (s *Screen) renderFriends(width int) string {
	st := s.styles
	title := s.sectionTitle("friends", width)
	friends, _ := s.data.Friends(s.target)
	if len(friends) == 0 {
		return title + "\n" + st.NewStyle().Foreground(st.Muted).
			Render("  no friends yet — be the first to /friend @"+s.target) + "\n"
	}
	limit := 12
	if len(friends) < limit {
		limit = len(friends)
	}
	var chips []string
	for _, f := range friends[:limit] {
		chips = append(chips, st.NewStyle().Foreground(st.Accent2).Render("@"+f.Login))
	}
	body := "  " + strings.Join(chips, "  ")
	if len(friends) > limit {
		body += st.NewStyle().Foreground(st.Muted).
			Render(fmt.Sprintf("   +%d more", len(friends)-limit))
	}
	return title + "\n" + body + "\n"
}

// ── Guestbook ───────────────────────────────────────────────────────────────

func (s *Screen) renderGuestbook(width int) string {
	st := s.styles
	title := s.sectionTitle("guestbook (friends only)", width)
	entries, _ := s.data.Guestbook(s.target, 8)
	if len(entries) == 0 {
		hint := "  empty — friends can /sign @" + s.target + " <message>"
		return title + "\n" + st.NewStyle().Foreground(st.Muted).Render(hint) + "\n"
	}
	var blocks []string
	for _, e := range entries {
		who := st.NewStyle().Foreground(st.Accent2).Render("@" + e.Author)
		when := st.NewStyle().Foreground(st.Muted).Italic(true).
			Render("  " + ui.HumanizeTime(e.CreatedAt))
		body := ui.Wrap(e.Body, width-4)
		blocks = append(blocks, "  "+who+when+"\n    "+
			strings.ReplaceAll(body, "\n", "\n    "))
	}
	return title + "\n" + strings.Join(blocks, "\n\n") + "\n"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
