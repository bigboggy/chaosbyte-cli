// Package theme holds the Chaosbyte palette and shared lipgloss styles.
// Screens import this package rather than redefining colors locally.
//
// The palette is a near-black ground with parchment-cream body text,
// punctuated by a muted phosphor green for positive marks and a muted
// gold for the moderator's voice and for moments that need attention.
// Three text colors is the working set. Anything else is a moment.
package theme

import "github.com/charmbracelet/lipgloss"

var (
	// Ground sits beneath every surface. Deeper than Tokyo Night, deeper
	// than the popular pastel-dark themes.
	Bg = lipgloss.Color("#0a0a0c")

	// Body is the cream of paper and phosphor. Most text lives here.
	Fg = lipgloss.Color("#e6dccb")

	// Muted reads as quiet metadata, timestamps, secondary status. Counts
	// as a body-register tone rather than an accent.
	Muted = lipgloss.Color("#7d7a72")

	// Accent is a muted phosphor green. Reserved for positive marks,
	// reactions, the OK channel, and the moments that earn a colored cell.
	// The hue is warm enough to read as a CRT phosphor rather than as a
	// Matrix screensaver.
	Accent = lipgloss.Color("#7a9a6a")

	// Accent2 is the muted gold the moderator uses in the margin and that
	// carries any state that wants attention without alarm.
	Accent2 = lipgloss.Color("#b3962a")

	// OK and Like share the green register.
	OK   = Accent
	Like = Accent

	// Warn and Fault share the gold register. The distinction lives in the
	// marker glyph rather than in a second color.
	Warn  = Accent2
	Fault = Accent2

	// Borders sit deep in the dark range, present without competing.
	BorderHi = lipgloss.Color("#25252d")
	BorderLo = lipgloss.Color("#1a1a1f")
)
