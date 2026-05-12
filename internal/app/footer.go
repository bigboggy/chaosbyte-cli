package app

import (
	"strings"

	"github.com/bchayka/gitstatus/internal/theme"
	"github.com/charmbracelet/lipgloss"
)

// renderFooter builds the bottom status bar: key hints from the active
// screen, plus any active flash message in green.
func (a *App) renderFooter() string {
	hints := a.activeScreen().Footer()
	var parts []string
	for _, k := range hints {
		parts = append(parts, theme.HelpKey.Render(k.Key)+" "+theme.HelpDesc.Render(k.Desc))
	}
	help := strings.Join(parts, "  ·  ")

	flash := ""
	if a.flash != "" {
		flash = lipgloss.NewStyle().Foreground(theme.OK).Render("  " + a.flash)
	}
	inner := theme.Status.Render(help) + flash
	return lipgloss.PlaceHorizontal(a.width, lipgloss.Left, inner)
}
