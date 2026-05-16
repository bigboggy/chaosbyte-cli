// Package leaderboard renders the full /leaderboard screen: a filterable
// table of users ranked by total tokens spent across all wired AI CLIs
// (Claude Code, OpenCode, Codex).
//
// The top-right header widget is a 5-row glance into the past 7 days; this
// screen is the deep dive — same data, but the user picks the time window
// (today, week, month, year) and sees every user, not just the top five.
package leaderboard

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

// timeRange names the filter buckets shown as tabs at the top of the screen.
// The integer values double as the index into rangeOrder for nav.
type timeRange int

const (
	rangeDaily timeRange = iota
	rangeWeekly
	rangeMonthly
	rangeYearly
)

var rangeOrder = []timeRange{rangeDaily, rangeWeekly, rangeMonthly, rangeYearly}

func (r timeRange) label() string {
	switch r {
	case rangeDaily:
		return "DAILY"
	case rangeWeekly:
		return "WEEKLY"
	case rangeMonthly:
		return "MONTHLY"
	case rangeYearly:
		return "YEARLY"
	}
	return ""
}

// since returns the inclusive lower bound for the filter. Today rolls over at
// 00:00 UTC since the underlying date column is a UTC YYYY-MM-DD string.
func (r timeRange) since(now time.Time) time.Time {
	n := now.UTC()
	switch r {
	case rangeDaily:
		return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
	case rangeWeekly:
		return n.AddDate(0, 0, -7)
	case rangeMonthly:
		return n.AddDate(0, -1, 0)
	case rangeYearly:
		return n.AddDate(-1, 0, 0)
	}
	return time.Time{}
}

// Screen is the full leaderboard view.
type Screen struct {
	styles *theme.Styles
	data   *store.Store

	filter timeRange
	scroll int // row offset into the table for vertical paging

	// joinVisible toggles the "join the leaderboard" install dialog overlay.
	// While visible the screen acts as an input owner so esc closes the
	// dialog instead of bouncing back to the lobby.
	joinVisible bool
}

// New returns a leaderboard screen wired to the shared store.
func New(styles *theme.Styles, data *store.Store) *Screen {
	return &Screen{styles: styles, data: data, filter: rangeWeekly}
}

// ShowJoin opens the install/join dialog. Called by the app router when the
// lobby emits OpenLeaderboardJoinMsg, so the dialog is already visible on
// the first frame after navigation.
func (s *Screen) ShowJoin() { s.joinVisible = true }

// ── Screen interface ────────────────────────────────────────────────────────

func (s *Screen) Init() tea.Cmd      { return nil }
func (s *Screen) Name() string       { return screens.LeaderboardID }
func (s *Screen) Title() string      { return "leaderboard" }

// InputFocused goes true while the join dialog is up so the router gives us
// every keystroke — otherwise esc would close the screen entirely instead of
// just dismissing the modal.
func (s *Screen) InputFocused() bool { return s.joinVisible }

func (s *Screen) HeaderContext() string {
	return strings.ToLower(s.filter.label())
}

func (s *Screen) Footer() []screens.KeyHint {
	if s.joinVisible {
		return []screens.KeyHint{
			{Key: "esc", Desc: "close"},
		}
	}
	return []screens.KeyHint{
		{Key: "h/l", Desc: "filter"},
		{Key: "1-4", Desc: "jump"},
		{Key: "j/k", Desc: "scroll"},
		{Key: "esc", Desc: "lobby"},
	}
}

func (s *Screen) Update(msg tea.Msg) (screens.Screen, tea.Cmd) {
	switch m := msg.(type) {
	case tea.KeyMsg:
		// Join-dialog has its own tiny key vocabulary — everything else is
		// suppressed so the modal feels modal. The dialog itself is opened
		// from the lobby via /leaderboard-join, not by an in-screen keybind.
		if s.joinVisible {
			switch m.String() {
			case "esc", "q", "enter":
				s.joinVisible = false
			}
			return s, nil
		}
		switch m.String() {
		case "h", "left", "shift+tab":
			i := int(s.filter)
			if i > 0 {
				s.filter = rangeOrder[i-1]
				s.scroll = 0
			}
		case "l", "right", "tab":
			i := int(s.filter)
			if i < len(rangeOrder)-1 {
				s.filter = rangeOrder[i+1]
				s.scroll = 0
			}
		case "1":
			s.filter = rangeDaily
			s.scroll = 0
		case "2":
			s.filter = rangeWeekly
			s.scroll = 0
		case "3":
			s.filter = rangeMonthly
			s.scroll = 0
		case "4":
			s.filter = rangeYearly
			s.scroll = 0
		case "j", "down":
			s.scroll++
		case "k", "up":
			s.scroll--
			if s.scroll < 0 {
				s.scroll = 0
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

// View renders the tab strip + ranked table within (width, height).
func (s *Screen) View(width, height int) string {
	st := s.styles

	// Cap the content width so the table reads well even on ultra-wide
	// terminals. The top-right widget overlay still fits at common sizes.
	contentW := width - 4
	if contentW > 84 {
		contentW = 84
	}
	if contentW < 40 {
		contentW = 40
	}

	since := s.filter.since(time.Now())
	entries, _ := s.data.Leaderboard(since, 1000) // soft cap

	tabs := s.renderTabs(contentW)
	subtitle := st.NewStyle().Foreground(st.Muted).Italic(true).
		Render(fmt.Sprintf("%d users · since %s UTC",
			len(entries), since.Format("2006-01-02")))

	tableHead, tableRows := s.renderTable(entries, contentW)

	// Vertical paging across the table rows only — tabs and headers stay
	// pinned at the top of the screen.
	chrome := []string{tabs, "", subtitle, "", tableHead}
	chromeH := 0
	for _, c := range chrome {
		chromeH += lipgloss.Height(c)
	}
	tableH := height - chromeH - 2 // bottom slack
	if tableH < 3 {
		tableH = 3
	}
	if s.scroll > len(tableRows) {
		s.scroll = max(0, len(tableRows)-1)
	}
	end := s.scroll + tableH
	if end > len(tableRows) {
		end = len(tableRows)
	}
	visibleRows := tableRows
	if s.scroll < len(tableRows) {
		visibleRows = tableRows[s.scroll:end]
	} else {
		visibleRows = nil
	}

	body := strings.Join(append(chrome, visibleRows...), "\n")
	body = ui.PadToHeight(body, height)

	// Center the table horizontally within the body width.
	leftPad := (width - contentW) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	out := indent(body, leftPad)

	if s.joinVisible {
		out = st.Place(width, height, lipgloss.Center, lipgloss.Center,
			s.renderJoinDialog())
	}
	return out
}

// ── Join dialog ─────────────────────────────────────────────────────────────

// installOneLiner is the curl pipe users run on their own machine to install
// the binary + the per-minute tracker. Defined as a constant rather than
// wrapped per-render so the displayed text is stable and trivially
// copy-pasteable.
const installOneLiner = "curl -fsSL https://raw.githubusercontent.com/bigboggy/vibespace/main/scripts/install.sh | bash"

// renderJoinDialog builds the centered modal shown over the leaderboard
// when /leaderboard-join is invoked. It explains in three short beats what
// installing does (binary + per-minute tracker), gives the curl one-liner
// the user copies, and reminds them that /auth is still the gating step.
func (s *Screen) renderJoinDialog() string {
	st := s.styles

	title := st.NewStyle().Foreground(st.Accent2).Bold(true).
		Render("🏆  JOIN THE LEADERBOARD")
	sub := st.NewStyle().Foreground(st.Muted).Italic(true).
		Render("install the local tracker — runs every minute, uploads token totals")

	curlLine := st.NewStyle().
		Foreground(st.OK).
		Background(st.Bg).
		Padding(0, 2).
		Render(installOneLiner)

	steps := []string{
		st.NewStyle().Foreground(st.Accent).Bold(true).Render("1.") + "  paste the line above into a shell on your laptop",
		st.NewStyle().Foreground(st.Accent).Bold(true).Render("2.") + "  back here, type " +
			st.NewStyle().Foreground(st.Accent).Bold(true).Render("/auth") +
			" to link your SSH key to your GitHub login",
		st.NewStyle().Foreground(st.Accent).Bold(true).Render("3.") + "  within a minute (or run `vibespace report` now) and you're on the board",
	}
	stepsBlock := strings.Join(steps, "\n")

	hint := st.NewStyle().Foreground(st.Muted).Italic(true).
		Render("press esc to close")

	inner := lipgloss.JoinVertical(lipgloss.Left,
		title,
		sub,
		"",
		curlLine,
		"",
		stepsBlock,
		"",
		hint,
	)

	return st.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(st.Accent).
		Padding(1, 3).
		Render(inner)
}

// ── Tabs ────────────────────────────────────────────────────────────────────

func (s *Screen) renderTabs(width int) string {
	st := s.styles
	var parts []string
	for _, r := range rangeOrder {
		label := r.label()
		if r == s.filter {
			parts = append(parts, st.NewStyle().
				Foreground(st.Bg).Background(st.Accent).
				Bold(true).Padding(0, 1).Render(label))
		} else {
			parts = append(parts, st.NewStyle().
				Foreground(st.Muted).Padding(0, 1).Render(label))
		}
	}
	strip := lipgloss.JoinHorizontal(lipgloss.Top, parts...)
	rule := st.NewStyle().Foreground(st.BorderLo).
		Render(strings.Repeat("─", width))
	return strip + "\n" + rule
}

// ── Table ───────────────────────────────────────────────────────────────────

// renderTable returns (header row, body rows). Splitting keeps the header
// pinned when the body scrolls.
func (s *Screen) renderTable(entries []store.LeaderboardEntry, width int) (string, []string) {
	st := s.styles
	col := func(text string, w int, align lipgloss.Position, fg lipgloss.Color, bold bool) string {
		style := st.NewStyle().Foreground(fg).Width(w).Align(align)
		if bold {
			style = style.Bold(true)
		}
		return style.Render(text)
	}

	// Column widths — sum should equal width. Tight on narrow terminals.
	rankW := 4
	totalW := 10
	srcW := 10
	loginW := width - rankW - totalW - srcW*3 - 4 // 4 = inter-column gaps
	if loginW < 10 {
		loginW = 10
	}

	head := col("#", rankW, lipgloss.Left, st.Muted, true) + " " +
		col("USER", loginW, lipgloss.Left, st.Muted, true) + " " +
		col("TOTAL", totalW, lipgloss.Right, st.Muted, true) + " " +
		col("CLAUDE", srcW, lipgloss.Right, st.Muted, true) + " " +
		col("OPENCODE", srcW, lipgloss.Right, st.Muted, true) + " " +
		col("CODEX", srcW, lipgloss.Right, st.Muted, true)
	rule := st.NewStyle().Foreground(st.BorderLo).
		Render(strings.Repeat("─", width))
	header := head + "\n" + rule

	if len(entries) == 0 {
		emptyMsg := st.NewStyle().Foreground(st.Muted).Italic(true).
			Render("  no usage tracked yet — run vibespace report on your machine to sync")
		return header, []string{"", emptyMsg}
	}

	var rows []string
	for i, e := range entries {
		rank := col(fmt.Sprintf("%d.", i+1), rankW, lipgloss.Left, st.Muted, false)
		login := col("@"+ui.Truncate(e.Login, loginW-1), loginW, lipgloss.Left, st.Accent, true)
		total := col(formatTokens(e.Total), totalW, lipgloss.Right, st.Like, true)
		claude := col(sourceCell(e.PerSource[store.SourceClaude]), srcW, lipgloss.Right, st.Fg, false)
		opencode := col(sourceCell(e.PerSource[store.SourceOpenCode]), srcW, lipgloss.Right, st.Fg, false)
		codex := col(sourceCell(e.PerSource[store.SourceCodex]), srcW, lipgloss.Right, st.Fg, false)
		rows = append(rows, rank+" "+login+" "+total+" "+claude+" "+opencode+" "+codex)
	}
	return header, rows
}

// sourceCell returns "—" for zero so the table doesn't read as a wall of 0s
// for sources a user hasn't touched.
func sourceCell(n int64) string {
	if n == 0 {
		return "—"
	}
	return formatTokens(n)
}

// formatTokens condenses a token count to fit in ~6 cells: 1.2k, 42.3M, 1.5B.
func formatTokens(n int64) string {
	switch {
	case n >= 1_000_000_000:
		return fmt.Sprintf("%.1fB", float64(n)/1_000_000_000)
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
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
