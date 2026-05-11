// Package spotlight features a single project, rotated every five minutes,
// with a live discussion chat below. Rotation is computed from wall-clock so
// no persistent state is needed.
package spotlight

import (
	"fmt"
	"strings"
	"time"

	"github.com/bchayka/gitstatus/internal/screens"
	"github.com/bchayka/gitstatus/internal/theme"
	"github.com/bchayka/gitstatus/internal/ui"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Screen struct {
	items      []Spotlight
	chat       []ui.ChatMessage
	chatScroll int

	input       textarea.Model
	inputActive bool
}

func New() *Screen {
	ta := textarea.New()
	ta.Placeholder = "join the discussion... (enter to send)"
	ta.Prompt = ""
	ta.ShowLineNumbers = false
	ta.CharLimit = 0
	ta.SetHeight(3)
	return &Screen{
		items: seedSpotlights(),
		chat:  seedChat(),
		input: ta,
	}
}

func (s *Screen) Init() tea.Cmd { return nil }

func (s *Screen) Name() string  { return screens.SpotlightID }
func (s *Screen) Title() string { return "spotlight" }

func (s *Screen) HeaderContext() string {
	_, secs := s.rotation()
	sep := lipgloss.NewStyle().Foreground(theme.Muted).Render(" · ")
	return lipgloss.NewStyle().Foreground(theme.Accent).Render("LIVE") + sep +
		lipgloss.NewStyle().Foreground(theme.Muted).Render("next in "+mmss(secs))
}

func (s *Screen) Footer() []screens.KeyHint {
	if s.inputActive {
		return []screens.KeyHint{
			{Key: "enter", Desc: "send"}, {Key: "ctrl+enter", Desc: "newline"}, {Key: "esc", Desc: "cancel"},
		}
	}
	return []screens.KeyHint{
		{Key: "i", Desc: "chat"}, {Key: "j/k", Desc: "scroll"}, {Key: "o", Desc: "open repo"}, {Key: "esc", Desc: "lobby"},
	}
}

func (s *Screen) InputFocused() bool { return s.inputActive }

func (s *Screen) Update(msg tea.Msg) (screens.Screen, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}
	if s.inputActive {
		return s.updateCompose(km)
	}
	return s.updateNormal(km)
}

func (s *Screen) updateNormal(km tea.KeyMsg) (screens.Screen, tea.Cmd) {
	switch km.String() {
	case "j", "down":
		s.chatScroll++
	case "k", "up":
		if s.chatScroll > 0 {
			s.chatScroll--
		}
	case "g":
		s.chatScroll = 0
	case "G":
		s.chatScroll = 9999
	case "i", "c":
		s.inputActive = true
		s.input.Focus()
		return s, textarea.Blink
	case "o", "enter":
		idx, _ := s.rotation()
		if idx < len(s.items) {
			return s, screens.Flash("opening: " + s.items[idx].RepoURL)
		}
	}
	return s, nil
}

func (s *Screen) updateCompose(km tea.KeyMsg) (screens.Screen, tea.Cmd) {
	switch km.String() {
	case "ctrl+enter", "alt+enter", "ctrl+j":
		s.input.InsertString("\n")
		return s, nil
	case "enter", "ctrl+s", "ctrl+d":
		body := strings.TrimRight(strings.TrimSpace(s.input.Value()), "\n")
		if body != "" {
			s.chat = append(s.chat, ui.ChatMessage{
				Author: "@boggy", Body: body, At: time.Now(),
			})
		}
		s.input.SetValue("")
		s.input.Blur()
		s.inputActive = false
		if body != "" {
			return s, screens.Flash("posted to spotlight chat")
		}
		return s, nil
	case "esc":
		s.input.SetValue("")
		s.input.Blur()
		s.inputActive = false
		return s, nil
	}
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(km)
	return s, cmd
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (s *Screen) View(width, height int) string {
	w := ui.FeedShellWidth(width)
	contentW := w - 2

	idx, secs := s.rotation()
	if idx >= len(s.items) {
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center,
			theme.Status.Render("no spotlight scheduled"))
	}
	sp := s.items[idx]

	title := theme.Title.Render("spotlight · " + sp.Project)
	rotateNote := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true).
		Render(fmt.Sprintf("next rotation in %s · %d/%d", mmss(secs), idx+1, len(s.items)))

	card := renderCard(sp, contentW)
	cardH := lipgloss.Height(card)

	inputH := 3
	if s.inputActive {
		inputH = 5
	}
	chatH := height - cardH - inputH - 6
	if chatH < 4 {
		chatH = 4
	}

	chat := s.renderChat(contentW, chatH)

	var input string
	if s.inputActive {
		s.input.SetWidth(contentW - 2)
		s.input.SetHeight(3)
		input = s.input.View()
	} else {
		input = lipgloss.NewStyle().Foreground(theme.Muted).Italic(true).
			Render("press i to join the discussion · j/k scroll · o open repo")
	}

	stacked := lipgloss.JoinVertical(lipgloss.Left,
		title, rotateNote, "", card,
		ui.Divider(contentW),
		chat,
		ui.Divider(contentW),
		input,
	)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Top, stacked)
}

func renderCard(sp Spotlight, width int) string {
	innerW := width - 4

	name := lipgloss.NewStyle().Foreground(theme.Accent2).Bold(true).Render(sp.Project)
	lang := lipgloss.NewStyle().Foreground(theme.Accent).Render(fmt.Sprintf("[%s]", sp.Language))
	author := lipgloss.NewStyle().Foreground(theme.OK).Render(sp.Author)
	stars := lipgloss.NewStyle().Foreground(theme.Warn).Render(fmt.Sprintf("★ %d", sp.Stars))

	header := fmt.Sprintf("%s  %s  %s  %s", name, lang, stars, author)
	desc := lipgloss.NewStyle().Foreground(theme.Fg).Render(ui.Wrap(sp.Description, innerW))
	url := lipgloss.NewStyle().Foreground(theme.Muted).Italic(true).Render(sp.RepoURL)

	var highlights []string
	for _, hi := range sp.Highlights {
		highlights = append(highlights,
			lipgloss.NewStyle().Foreground(theme.Accent).Render("  ▸ ")+
				lipgloss.NewStyle().Foreground(theme.Fg).Render(hi))
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		header, "", desc, "", strings.Join(highlights, "\n"), "", url,
	)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Accent2).
		Padding(0, 2).
		Width(innerW).
		Render(content)
}

func (s *Screen) renderChat(width, height int) string {
	title := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true).Render("live discussion")
	var lines []string
	for _, msg := range s.chat {
		lines = append(lines, ui.RenderChatLine(msg, width)...)
	}
	maxScroll := len(lines) - (height - 1)
	if maxScroll < 0 {
		maxScroll = 0
	}
	if s.chatScroll > maxScroll {
		s.chatScroll = maxScroll
	}
	end := len(lines) - s.chatScroll
	start := end - (height - 1)
	if start < 0 {
		start = 0
	}
	if end > len(lines) {
		end = len(lines)
	}
	if end < start {
		end = start
	}
	visible := strings.Join(lines[start:end], "\n")
	visible = ui.PadToHeight(visible, height-1)
	return lipgloss.JoinVertical(lipgloss.Left, title, visible)
}
