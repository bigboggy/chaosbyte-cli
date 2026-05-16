package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/bigboggy/vibespace/internal/store"
	"github.com/bigboggy/vibespace/internal/theme"
	"github.com/bigboggy/vibespace/internal/ui"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Top-right leaderboard widget. Always queries the past 7 days so the corner
// glance is "what's happening right now" — the full /leaderboard screen is
// what you use to slice by day / week / month / year.
const (
	widgetWidth    = 28
	widgetTopN     = 5
	widgetSinceDur = 7 * 24 * time.Hour
	widgetMinTermW = 70 // hide widget on terminals narrower than this
)

// renderLeaderboardWidget builds the bordered panel that lives in the top
// right corner. Returns "" when there's no store wired (local mode without
// data, or terminal too narrow) so the caller can skip the overlay entirely.
func (a *App) renderLeaderboardWidget() string {
	if a.data == nil || a.width < widgetMinTermW {
		return ""
	}
	st := a.styles
	since := time.Now().Add(-widgetSinceDur)
	entries, _ := a.data.Leaderboard(since, widgetTopN)

	title := st.NewStyle().Foreground(st.Accent2).Bold(true).Render("🏆 TOP 5")
	rng := st.NewStyle().Foreground(st.Muted).Render("  7d")
	header := title + rng

	innerW := widgetWidth - 4 // 2 border + 2 padding
	rows := []string{header, ""}

	if len(entries) == 0 {
		rows = append(rows,
			st.NewStyle().Foreground(st.Muted).Italic(true).Render("no usage tracked"),
			st.NewStyle().Foreground(st.Muted).Italic(true).Render("yet — run"),
			st.NewStyle().Foreground(st.Accent).Render("vibespace report"),
			st.NewStyle().Foreground(st.Muted).Italic(true).Render("on your machine"),
			"",
		)
	} else {
		for i := 0; i < widgetTopN; i++ {
			if i < len(entries) {
				rows = append(rows, formatLeaderRow(st, i+1, entries[i], innerW))
			} else {
				rows = append(rows, "")
			}
		}
	}
	rows = append(rows, "")
	rows = append(rows, st.NewStyle().Foreground(st.Muted).Italic(true).
		Render("press /leaderboard"))

	return st.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(st.BorderLo).
		Padding(0, 1).
		Width(innerW).
		Render(strings.Join(rows, "\n"))
}

// formatLeaderRow renders one ranked row, right-aligning the token count so
// the column reads cleanly regardless of login length.
func formatLeaderRow(st *theme.Styles, rank int, e store.LeaderboardEntry, innerW int) string {
	rankStr := st.NewStyle().Foreground(st.Muted).Render(fmt.Sprintf("%d.", rank))
	total := st.NewStyle().Foreground(st.Like).Bold(true).Render(formatTokens(e.Total))

	loginBudget := innerW - lipgloss.Width(rankStr) - lipgloss.Width(total) - 2
	if loginBudget < 4 {
		loginBudget = 4
	}
	login := ui.Truncate("@"+e.Login, loginBudget)
	loginStyled := st.NewStyle().Foreground(st.Accent).Render(login)

	rowLeft := rankStr + " " + loginStyled
	pad := innerW - lipgloss.Width(rowLeft) - lipgloss.Width(total)
	if pad < 1 {
		pad = 1
	}
	return rowLeft + strings.Repeat(" ", pad) + total
}

// formatTokens condenses a token count to fit in ~5 cells: 1.2k, 42.3M, etc.
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

// overlayTopRight splices the widget onto the top-right of body so the
// widget's rightmost column lands at totalWidth-1. Body lines under the
// widget are truncated ANSI-aware so the existing styling on the
// surviving left portion stays intact.
//
// Returns body unchanged when widget is empty or the terminal is too narrow
// to fit both side-by-side comfortably.
func overlayTopRight(body, widget string, totalWidth int) string {
	if widget == "" {
		return body
	}
	widgetLines := strings.Split(widget, "\n")
	widgetW := 0
	for _, l := range widgetLines {
		if w := lipgloss.Width(l); w > widgetW {
			widgetW = w
		}
	}
	if widgetW == 0 || totalWidth < widgetW+20 {
		return body
	}
	const gutter = 1
	leftBudget := totalWidth - widgetW - gutter

	bodyLines := strings.Split(body, "\n")
	for i, wLine := range widgetLines {
		var bodyLine string
		if i < len(bodyLines) {
			bodyLine = bodyLines[i]
		}
		clipped := ansi.Truncate(bodyLine, leftBudget, "")
		pad := leftBudget - lipgloss.Width(clipped)
		if pad < 0 {
			pad = 0
		}
		composed := clipped + strings.Repeat(" ", pad+gutter) + wLine
		if i < len(bodyLines) {
			bodyLines[i] = composed
		} else {
			bodyLines = append(bodyLines, composed)
		}
	}
	return strings.Join(bodyLines, "\n")
}
