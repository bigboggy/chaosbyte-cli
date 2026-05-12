// Package news is the combined-source news feed (HN, Lobsters, /r/programming
// and friends). All data is mocked; "open" surfaces the URL via FlashMsg.
package news

import (
	"fmt"
	"strings"
	"time"

	"github.com/bchayka/gitstatus/internal/field"
	"github.com/bchayka/gitstatus/internal/screens"
	"github.com/bchayka/gitstatus/internal/theme"
	"github.com/bchayka/gitstatus/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Screen struct {
	items  []Item
	idx    int
	scroll int

	backdrop *field.Backdrop
}

func New() *Screen { return &Screen{items: seedItems(), backdrop: field.NewBackdrop()} }

func (s *Screen) Init() tea.Cmd { return field.TickCmd() }

func (s *Screen) Name() string  { return screens.NewsID }
func (s *Screen) Title() string { return "news" }

func (s *Screen) HeaderContext() string {
	return lipgloss.NewStyle().Foreground(theme.Muted).
		Render(fmt.Sprintf("%d/%d", s.idx+1, len(s.items)))
}

func (s *Screen) Footer() []screens.KeyHint {
	return []screens.KeyHint{
		{Key: "j/k", Desc: "move"},
		{Key: "enter", Desc: "open"},
		{Key: "y", Desc: "copy url"},
		{Key: "esc", Desc: "lobby"},
	}
}

func (s *Screen) InputFocused() bool { return false }

func (s *Screen) Update(msg tea.Msg) (screens.Screen, tea.Cmd) {
	switch m := msg.(type) {
	case field.TickMsg:
		s.backdrop.Tick(time.Time(m))
		return s, field.TickCmd()
	case tea.MouseMsg:
		s.backdrop.SetCursor(float64(m.X), float64(m.Y))
		return s, nil
	}
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}
	s.backdrop.Pulse(0.04)
	switch km.String() {
	case "j", "down":
		if s.idx < len(s.items)-1 {
			s.idx++
		}
	case "k", "up":
		if s.idx > 0 {
			s.idx--
		}
	case "g":
		s.idx = 0
	case "G":
		s.idx = len(s.items) - 1
	case "enter", "o":
		if s.idx < len(s.items) {
			return s, screens.OpenURL(s.items[s.idx].URL)
		}
	case "y":
		if s.idx < len(s.items) {
			return s, screens.Flash("url copied (in spirit): " + s.items[s.idx].URL)
		}
	}
	return s, nil
}

func (s *Screen) View(width, height int) string {
	w := ui.FeedShellWidth(width)
	contentW := w - 2

	title := theme.Title.Render("news · combined feed")
	subtitle := lipgloss.NewStyle().Foreground(theme.Muted).Italic(true).
		Render("HN · Lobsters · /r/programming · DevHQ · ArsTechnica")

	cardH := 4
	bodyH := height - 4
	if bodyH < cardH {
		bodyH = cardH
	}
	visibleCount := bodyH / (cardH + 1)
	if visibleCount < 1 {
		visibleCount = 1
	}

	if s.idx < s.scroll {
		s.scroll = s.idx
	}
	if s.idx >= s.scroll+visibleCount {
		s.scroll = s.idx - visibleCount + 1
	}
	end := s.scroll + visibleCount
	if end > len(s.items) {
		end = len(s.items)
	}

	var cards []string
	for i := s.scroll; i < end; i++ {
		cards = append(cards, renderCard(s.items[i], contentW, i == s.idx))
	}
	body := strings.Join(cards, "\n")
	bodyRows := strings.Split(body, "\n")
	bodyHActual := bodyH - 4
	if bodyHActual < 1 {
		bodyHActual = 1
	}
	if len(bodyRows) < bodyHActual {
		pad := make([]string, bodyHActual-len(bodyRows))
		bodyRows = append(bodyRows, pad...)
	}
	fieldRows := strings.Split(s.backdrop.Render(contentW, bodyHActual), "\n")
	composed := field.Composite(bodyRows, fieldRows, bodyHActual)

	indicator := ""
	if len(s.items) > visibleCount {
		pct := 100
		if denom := len(s.items) - visibleCount; denom > 0 {
			pct = s.scroll * 100 / denom
		}
		indicator = lipgloss.NewStyle().Foreground(theme.Muted).Width(contentW).Align(lipgloss.Right).
			Render(fmt.Sprintf("scroll %d%%   %d/%d", pct, s.idx+1, len(s.items)))
	}

	stacked := lipgloss.JoinVertical(lipgloss.Left,
		title, subtitle, indicator, "", composed,
	)
	return lipgloss.Place(width, height, lipgloss.Left, lipgloss.Top, stacked)
}

func renderCard(it Item, width int, focused bool) string {
	innerW := width - 4

	source := sourceTag(it.Source)
	title := lipgloss.NewStyle().Foreground(theme.Fg).Bold(true).
		Render(ui.Truncate(it.Title, innerW-len(it.Source)-3))
	header := source + "  " + title

	meta := fmt.Sprintf("%s ▲ %d   %s 💬 %d   %s   %s",
		theme.LikeIcon.Render("▲"), it.Score,
		theme.CommentCount.Render("💬"), it.Comments,
		theme.CommitAuthor.Render(it.Author),
		theme.CommitTime.Render(ui.HumanizeTime(it.At)),
	)
	url := lipgloss.NewStyle().Foreground(theme.Muted).Italic(true).Render(ui.Truncate(it.URL, innerW))

	content := lipgloss.JoinVertical(lipgloss.Left, header, meta, url)
	box := lipgloss.NewStyle().Padding(0, 1).Width(innerW)
	if focused {
		box = box.Border(lipgloss.RoundedBorder()).BorderForeground(theme.Accent)
	} else {
		box = box.Border(lipgloss.HiddenBorder())
	}
	return box.Render(content)
}

func sourceTag(s string) string {
	var bg lipgloss.Color
	switch s {
	case "HN":
		bg = theme.Warn
	case "Lobsters":
		bg = theme.OK
	case "/r/programming":
		bg = theme.Like
	case "DevHQ":
		bg = theme.Accent
	case "ArsTechnica":
		bg = theme.Accent2
	default:
		bg = theme.Muted
	}
	return lipgloss.NewStyle().Foreground(theme.Bg).Background(bg).Bold(true).Padding(0, 1).Render(s)
}
