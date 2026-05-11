package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/bchayka/gitstatus/internal/screens"
	"github.com/bchayka/gitstatus/internal/theme"
	"github.com/bchayka/gitstatus/internal/ui"
	"github.com/charmbracelet/lipgloss"
)

// renderHeader builds the top bar: brand · screen chip · screen-specific
// context, with a clock right-aligned.
func (a *App) renderHeader() string {
	scr := a.activeScreen()

	title := lipgloss.NewStyle().Foreground(theme.Accent2).Bold(true).Render("chaosbyte")
	sep := lipgloss.NewStyle().Foreground(theme.Muted).Render(" · ")

	chip := lipgloss.NewStyle().
		Foreground(theme.Bg).
		Background(theme.Accent).
		Bold(true).
		Padding(0, 1).
		Render(scr.Title())

	left := title + sep + chip
	if ctx := scr.HeaderContext(); ctx != "" {
		left += sep + ctx
	}

	right := lipgloss.NewStyle().Foreground(theme.Muted).Italic(true).
		Render(time.Now().Format("Mon 15:04:05"))

	shellW := ui.FeedShellWidth(a.width)
	gap := shellW - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	inner := left + strings.Repeat(" ", gap) + right
	return lipgloss.PlaceHorizontal(a.width, lipgloss.Center, inner)
}

// tooSmall is the error screen shown when the terminal is below the minimum
// size that screens are designed for.
func tooSmall(w, h int) string {
	return lipgloss.NewStyle().Foreground(theme.Warn).
		Render(fmt.Sprintf("terminal too small (%dx%d), need at least %dx%d",
			w, h, ui.MinWidth, ui.MinHeight))
}

// renderFrame assembles header + body + footer for the current frame.
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
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}
