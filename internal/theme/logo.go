package theme

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// LogoLines is the CHAOSBYTE studio splash (ANSI Shadow style). 6 rows, 75 cols.
// chaosbyte is the maker; vibespace is the chatroom product the splash leads
// into. The logo therefore stays on the studio brand, not the product brand.
var LogoLines = []string{
	" ██████╗██╗  ██╗ █████╗  ██████╗ ███████╗██████╗ ██╗   ██╗████████╗███████╗",
	"██╔════╝██║  ██║██╔══██╗██╔═══██╗██╔════╝██╔══██╗╚██╗ ██╔╝╚══██╔══╝██╔════╝",
	"██║     ███████║███████║██║   ██║███████╗██████╔╝ ╚████╔╝    ██║   █████╗  ",
	"██║     ██╔══██║██╔══██║██║   ██║╚════██║██╔══██╗  ╚██╔╝     ██║   ██╔══╝  ",
	"╚██████╗██║  ██║██║  ██║╚██████╔╝███████║██████╔╝   ██║      ██║   ███████╗",
	" ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═════╝    ╚═╝      ╚═╝   ╚══════╝",
}

// LogoGradient cycles the three accent colors top-to-bottom for the logo.
var LogoGradient = []lipgloss.Color{
	Accent, Accent,
	Accent2, Accent2,
	Like, Like,
}

// RenderLogo paints LogoLines using LogoGradient.
func RenderLogo() string {
	var out []string
	for i, line := range LogoLines {
		out = append(out, lipgloss.NewStyle().
			Foreground(LogoGradient[i%len(LogoGradient)]).
			Bold(true).
			Render(line))
	}
	return strings.Join(out, "\n")
}
