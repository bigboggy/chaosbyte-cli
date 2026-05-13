// Package spotlight features one project at a time. The engine cycles items
// through presenting → transition → opt-in → next, with the title rendered
// by the field engine so it cascades on entry and on cursor hover.
package spotlight

import (
	"fmt"
	"strings"
	"time"

	"github.com/bchayka/gitstatus/internal/field"
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

	backdrop *field.Backdrop
	engine   *Engine
	fgIdx    int // index whose title is currently registered as foreground; -1 sentinel
}

func New() *Screen {
	ta := textarea.New()
	ta.Placeholder = "join the discussion... (enter to send)"
	ta.Prompt = ""
	ta.ShowLineNumbers = false
	ta.CharLimit = 0
	ta.SetHeight(3)
	items := seedSpotlights()
	return &Screen{
		items:    items,
		chat:     seedChat(),
		input:    ta,
		backdrop: field.NewBackdrop(),
		engine:   NewEngine(len(items)),
		fgIdx:    -1,
	}
}

func (s *Screen) Init() tea.Cmd { return field.TickCmd() }

// OnEnter is the router's field-driven entry hook: fire a cascade for the
// current spotlight so the project name flap-spins on entry, then decays.
func (s *Screen) OnEnter() {
	s.fgIdx = -1
	s.cascadeCurrent()
}

// cascadeCurrent fires the title cascade for the active item. Called on
// entry and on engine rotation; the cascade auto-decays so the field
// returns to quiet after the moment lands.
func (s *Screen) cascadeCurrent() {
	if s.engine.Index() >= len(s.items) {
		return
	}
	sp := s.items[s.engine.Index()]
	s.backdrop.AddCascade(field.CascadeLine{
		Row:   0,
		Text:  "spotlight · " + sp.Project,
		Decay: 4 * time.Second,
	})
}

func (s *Screen) Name() string  { return screens.SpotlightID }
func (s *Screen) Title() string { return "spotlight" }

func (s *Screen) HeaderContext() string {
	sep := lipgloss.NewStyle().Foreground(theme.Muted).Render(" · ")
	stateLabel := "LIVE"
	if s.engine.IsOptIn() {
		stateLabel = "QUEUED"
	} else if s.engine.IsTransition() {
		stateLabel = "—"
	}
	remaining := int(s.engine.Remaining().Seconds())
	return lipgloss.NewStyle().Foreground(theme.Accent).Render(stateLabel) + sep +
		lipgloss.NewStyle().Foreground(theme.Muted).Render("next in "+mmss(remaining))
}

func (s *Screen) Footer() []screens.KeyHint {
	if s.inputActive {
		return []screens.KeyHint{
			{Key: "enter", Desc: "send"}, {Key: "ctrl+enter", Desc: "newline"}, {Key: "esc", Desc: "cancel"},
		}
	}
	if s.engine.IsOptIn() {
		return []screens.KeyHint{
			{Key: "enter", Desc: "go now"}, {Key: "n", Desc: "skip"}, {Key: "i", Desc: "post"}, {Key: "esc", Desc: "lobby"},
		}
	}
	return []screens.KeyHint{
		{Key: "o", Desc: "open repo"}, {Key: "n", Desc: "next"}, {Key: "i", Desc: "post"}, {Key: "j/k", Desc: "scroll"}, {Key: "esc", Desc: "lobby"},
	}
}

func (s *Screen) InputFocused() bool { return s.inputActive }

func (s *Screen) Update(msg tea.Msg) (screens.Screen, tea.Cmd) {
	switch m := msg.(type) {
	case field.TickMsg:
		now := time.Time(m)
		s.backdrop.Tick(now)
		if entered := s.engine.Tick(now); entered {
			s.backdrop.Pulse(0.8)
		}
		s.syncForegroundTitle()
		// Spotlight is cascade-driven: the room is quiet unless a rotation
		// just fired, in which case AddCascade has already pulsed ot and
		// the engine will animate until it decays.
		s.backdrop.SetTier(0)
		return s, field.TickCmd()
	case tea.MouseMsg:
		s.backdrop.SetCursor(float64(m.X), float64(m.Y))
		return s, nil
	case tea.KeyMsg:
		s.backdrop.Pulse(0.04)
		if s.inputActive {
			return s.updateCompose(m)
		}
		return s.updateNormal(m)
	}
	return s, nil
}

// syncForegroundTitle fires a cascade whenever the active spotlight index
// changes — engine rotation, accept, skip. The cascade auto-decays so the
// title doesn't sit on the field forever; the persistent screen label
// stays as a lipgloss element above the card.
func (s *Screen) syncForegroundTitle() {
	idx := s.engine.Index()
	if idx == s.fgIdx || idx >= len(s.items) {
		return
	}
	s.fgIdx = idx
	s.cascadeCurrent()
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
	case "enter":
		if s.engine.Accept() {
			s.backdrop.Pulse(0.8)
			return s, nil
		}
		if idx := s.engine.Index(); idx < len(s.items) {
			return s, screens.OpenURL(s.items[idx].RepoURL)
		}
	case "o":
		if idx := s.engine.Index(); idx < len(s.items) {
			return s, screens.OpenURL(s.items[idx].RepoURL)
		}
	case "n":
		s.engine.Skip()
		return s, screens.Flash("skipped")
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

	idx := s.engine.Index()
	if idx >= len(s.items) {
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center,
			theme.Status.Render("no spotlight scheduled"))
	}
	sp := s.items[idx]

	// Row 0 is the persistent screen title (lipgloss). The cascade fired on
	// rotation also lives at row 0 of the field and shows through when it's
	// active; once it decays the static title still sits above the field.
	titleLine := theme.Title.Render("spotlight · " + sp.Project)
	statusLine := s.renderStatusLine(sp, contentW)

	var card string
	cardH := 0
	if !s.engine.IsTransition() {
		card = renderCard(sp, contentW)
		cardH = lipgloss.Height(card)
	}

	var optIn string
	optInH := 0
	if s.engine.IsOptIn() {
		optIn = renderOptInPanel(sp, s.engine.OptInProgress(), contentW)
		optInH = lipgloss.Height(optIn) + 1
	}

	inputH := 3
	if s.inputActive {
		inputH = 5
	}
	chatH := height - cardH - inputH - optInH - 6
	if chatH < 4 {
		chatH = 4
	}

	chat := s.renderChat(contentW, chatH)

	var input string
	if s.inputActive {
		s.input.SetWidth(contentW - 2)
		s.input.SetHeight(3)
		input = s.input.View()
	} else if s.engine.IsOptIn() {
		input = lipgloss.NewStyle().Foreground(theme.Muted).Italic(true).
			Render("enter — start this spotlight now · n — skip to next · i — post in this thread")
	} else {
		input = lipgloss.NewStyle().Foreground(theme.Muted).Italic(true).
			Render("o — open repo · n — next spotlight · i — post · j/k — scroll")
	}

	parts := []string{
		titleLine,
		statusLine,
		"",
	}
	if card != "" {
		parts = append(parts, card)
	}
	parts = append(parts,
		ui.Divider(contentW),
		chat,
		ui.Divider(contentW),
	)
	if optIn != "" {
		parts = append(parts, optIn)
	}
	parts = append(parts, input)

	content := lipgloss.JoinVertical(lipgloss.Left, parts...)
	contentRows := strings.Split(content, "\n")
	if len(contentRows) < height {
		pad := make([]string, height-len(contentRows))
		contentRows = append(contentRows, pad...)
	} else if len(contentRows) > height {
		contentRows = contentRows[:height]
	}

	fieldRows := strings.Split(s.backdrop.Render(contentW, height), "\n")
	composed := field.Composite(contentRows, fieldRows, height)
	return lipgloss.Place(width, height, lipgloss.Left, lipgloss.Top, composed)
}

func (s *Screen) renderStatusLine(sp Spotlight, width int) string {
	count := len(s.items)
	if count == 0 {
		count = 1
	}
	pos := fmt.Sprintf("%d/%d", s.engine.Index()+1, count)
	remaining := mmss(int(s.engine.Remaining().Seconds()))

	switch {
	case s.engine.IsOptIn():
		return lipgloss.NewStyle().Foreground(theme.Accent).Bold(true).
			Render(fmt.Sprintf("queued · %s · 15s window · %s", sp.Project, pos))
	case s.engine.IsTransition():
		return lipgloss.NewStyle().Foreground(theme.Muted).
			Render("…")
	}
	return lipgloss.NewStyle().Foreground(theme.Accent).Bold(true).
		Render(fmt.Sprintf("on stage · %s remaining · %s", remaining, pos))
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

// renderOptInPanel is the per-frame "next up · 15s countdown" block shown
// during the opt-in window. progress is 0..1 across the window.
func renderOptInPanel(sp Spotlight, progress float64, width int) string {
	innerW := width - 4
	barW := innerW - 8
	if barW < 8 {
		barW = 8
	}
	filled := int(float64(barW) * progress)
	if filled < 0 {
		filled = 0
	}
	if filled > barW {
		filled = barW
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barW-filled)
	barStyled := lipgloss.NewStyle().Foreground(theme.Accent).Render(bar)

	head := lipgloss.NewStyle().Foreground(theme.Warn).Bold(true).Render("next up")
	name := lipgloss.NewStyle().Foreground(theme.Accent2).Bold(true).Render(sp.Project)
	by := lipgloss.NewStyle().Foreground(theme.Muted).Render("by " + sp.Author)
	tag := lipgloss.NewStyle().Foreground(theme.Accent).Render(fmt.Sprintf("[%s]", sp.Language))

	row1 := fmt.Sprintf("%s  %s  %s  %s", head, name, tag, by)
	row2 := lipgloss.NewStyle().Foreground(theme.Fg).Render(ui.Truncate(sp.Description, innerW))

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Warn).
		Padding(0, 2).
		Width(innerW).
		Render(lipgloss.JoinVertical(lipgloss.Left, row1, row2, "", barStyled))
}

func (s *Screen) renderChat(width, height int) string {
	title := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true).Render("live discussion")
	bodyH := height - 1
	if bodyH < 1 {
		bodyH = 1
	}

	var lines []string
	for _, msg := range s.chat {
		lines = append(lines, ui.RenderChatLine(msg, width)...)
	}
	maxScroll := len(lines) - bodyH
	if maxScroll < 0 {
		maxScroll = 0
	}
	if s.chatScroll > maxScroll {
		s.chatScroll = maxScroll
	}
	end := len(lines) - s.chatScroll
	start := end - bodyH
	if start < 0 {
		start = 0
	}
	if end > len(lines) {
		end = len(lines)
	}
	if end < start {
		end = start
	}
	chatRows := lines[start:end]
	if len(chatRows) < bodyH {
		pad := make([]string, bodyH-len(chatRows))
		chatRows = append(pad, chatRows...)
	}
	return lipgloss.JoinVertical(lipgloss.Left, title, strings.Join(chatRows, "\n"))
}
