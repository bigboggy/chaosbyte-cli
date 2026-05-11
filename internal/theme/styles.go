package theme

import "github.com/charmbracelet/lipgloss"

var (
	TabActive = lipgloss.NewStyle().
			Foreground(Bg).
			Background(Accent).
			Bold(true).
			Padding(0, 2)

	TabInactive = lipgloss.NewStyle().
			Foreground(Muted).
			Padding(0, 2)

	TabMore = lipgloss.NewStyle().
		Foreground(Accent2).
		Padding(0, 2)

	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(Accent2).
		Padding(0, 1)

	Status = lipgloss.NewStyle().
		Foreground(Muted).
		Padding(0, 1)

	BranchItem = lipgloss.NewStyle().
			Foreground(Fg).
			Padding(0, 1)

	BranchItemSel = lipgloss.NewStyle().
			Foreground(Bg).
			Background(Accent).
			Bold(true).
			Padding(0, 1)

	CommitSHA = lipgloss.NewStyle().Foreground(Muted)

	CommitAuthor = lipgloss.NewStyle().Foreground(OK)

	CommitTime = lipgloss.NewStyle().
			Foreground(Muted).
			Italic(true)

	CommitMsg = lipgloss.NewStyle().Foreground(Fg)

	LikeIcon = lipgloss.NewStyle().Foreground(Like)
	Liked    = lipgloss.NewStyle().Foreground(Like).Bold(true)

	CommentCount = lipgloss.NewStyle().Foreground(Accent)

	CommentAuthor = lipgloss.NewStyle().
			Foreground(Accent2).
			Bold(true)

	CommentBody = lipgloss.NewStyle().Foreground(Fg)

	HelpKey = lipgloss.NewStyle().
		Foreground(Accent).
		Bold(true)

	HelpDesc = lipgloss.NewStyle().Foreground(Muted)
)

// Pane returns the rounded-border container used by overlays. The border color
// shifts when focused to draw the eye.
func Pane(focused bool) lipgloss.Style {
	border := BorderLo
	if focused {
		border = BorderHi
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(0, 1)
}
