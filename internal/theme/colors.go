// Package theme holds the Catppuccin Mocha palette and shared lipgloss styles.
// Screens import this package rather than redefining colors locally.
package theme

import "github.com/charmbracelet/lipgloss"

var (
	Bg       = lipgloss.Color("#1a1b26")
	Fg       = lipgloss.Color("#c0caf5")
	Muted    = lipgloss.Color("#565f89")
	Accent   = lipgloss.Color("#7aa2f7")
	Accent2  = lipgloss.Color("#bb9af7")
	OK       = lipgloss.Color("#9ece6a")
	Warn     = lipgloss.Color("#e0af68")
	Like     = lipgloss.Color("#f7768e")
	BorderHi = lipgloss.Color("#7aa2f7")
	BorderLo = lipgloss.Color("#3b4261")
)
