// Package theme holds the Chaosbyte palette and shared lipgloss styles.
// Screens import this package rather than redefining colors locally.
//
// The palette is a near-black ground with parchment-cream body text,
// punctuated by a single copper accent at moments that earn it. Gold is
// reserved for moderator marks. Oxidized red carries failures. The room
// lives in two colors most of the time, the cream body on the near-black
// ground. Anything else is a moment.
package theme

import "github.com/charmbracelet/lipgloss"

var (
	// Ground sits beneath every surface. Deeper than Tokyo Night, deeper
	// than the popular pastel-dark themes.
	Bg = lipgloss.Color("#0a0a0c")

	// Body is the cream of paper and phosphor. Most text lives here.
	Fg = lipgloss.Color("#e6dccb")

	// Muted reads as quiet metadata, timestamps, secondary status.
	Muted = lipgloss.Color("#7d7a72")

	// Accent is copper, reserved for moments of intentional emphasis.
	// On screen less than five percent of cells on any given frame.
	Accent = lipgloss.Color("#c46b3a")

	// Accent2 is a deeper copper for secondary marks that should sit
	// near the accent without competing with it.
	Accent2 = lipgloss.Color("#8a4a26")

	// OK and Like share the copper register because the room celebrates
	// the same way it emphasizes. Saturated green and pink do not belong
	// in this aesthetic.
	OK   = lipgloss.Color("#c46b3a")
	Like = lipgloss.Color("#c46b3a")

	// Warn carries a muted gold, the same gold the moderator uses in the
	// margin. Soft urgency, not alarm.
	Warn = lipgloss.Color("#b3962a")

	// Fault carries oxidized red for failures and breakages, dark enough
	// not to read as a saturated alert.
	Fault = lipgloss.Color("#a02d1f")

	// Borders sit deep in the dark range, present without competing.
	BorderHi = lipgloss.Color("#25252d")
	BorderLo = lipgloss.Color("#1a1a1f")
)
