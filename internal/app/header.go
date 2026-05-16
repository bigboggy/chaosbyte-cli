package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/bigboggy/vibespace/internal/screens"
	"github.com/bigboggy/vibespace/internal/theme"
	"github.com/bigboggy/vibespace/internal/ui"
	"github.com/charmbracelet/lipgloss"
)

// renderHeader builds the top bar: brand · screen chip · screen-specific
// context, with a clock right-aligned.
func (a *App) renderHeader() string {
	scr := a.activeScreen()
	st := a.styles

	title := st.NewStyle().Foreground(st.Accent2).Bold(true).Render("vibespace")
	sep := st.NewStyle().Foreground(st.Muted).Render(" · ")

	chip := st.NewStyle().
		Foreground(st.Bg).
		Background(st.Accent).
		Bold(true).
		Padding(0, 1).
		Render(scr.Title())

	left := title + sep + chip
	if ctx := scr.HeaderContext(); ctx != "" {
		left += sep + ctx
	}

	right := st.NewStyle().Foreground(st.Muted).Italic(true).
		Render(time.Now().Format("Mon 15:04:05"))

	shellW := ui.FeedShellWidth(a.width)
	gap := shellW - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	inner := left + strings.Repeat(" ", gap) + right
	return st.PlaceHorizontal(a.width, lipgloss.Left, inner)
}

// tooSmall is the error screen shown when the terminal is below the minimum
// size that screens are designed for.
func tooSmall(st *theme.Styles, w, h int) string {
	return st.NewStyle().Foreground(st.Warn).
		Render(fmt.Sprintf("terminal too small (%dx%d), need at least %dx%d",
			w, h, ui.MinWidth, ui.MinHeight))
}

// renderFrame assembles header + body + footer for the current frame.
//
// On non-intro screens the leaderboard widget is overlaid on the top-right
// of the body. We accept that the body's top-right corner gets clipped — chat
// scrollback pushes old lines up off the visible window anyway, and profile
// cards stay centered with horizontal slack at common terminal widths.
func (a *App) renderFrame() string {
	header := a.renderHeader()
	footer := a.renderFooter()
	bodyH := a.height - lipgloss.Height(header) - lipgloss.Height(footer)
	if bodyH < 6 {
		bodyH = 6
	}

	// Intro is fullscreen — no chrome around it.
	if a.current == screens.IntroID {
		return a.activeScreen().View(a.width, a.height)
	}

	body := a.activeScreen().View(a.width, bodyH)
	body = overlayTopRight(body, a.renderLeaderboardWidget(), a.width)
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}
