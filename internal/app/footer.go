package app

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderFooter builds the bottom status bar: key hints from the active screen.
func (a *App) renderFooter() string {
	st := a.styles
	hints := a.activeScreen().Footer()
	var parts []string
	for _, k := range hints {
		parts = append(parts, st.HelpKey().Render(k.Key)+" "+st.HelpDesc().Render(k.Desc))
	}
	help := strings.Join(parts, "  ·  ")
	return st.PlaceHorizontal(a.width, lipgloss.Left, st.Status().Render(help))
}
