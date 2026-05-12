// Package lobby is the chat-style entry point that doubles as the app's home
// screen. It owns the channel list, manages an always-focused input, and
// routes slash commands to other screens via screens.Navigate.
//
// Files in this package:
//   - lobby.go     — Screen type, Init/Update/View, message posting
//   - commands.go  — slash command registry + per-command handlers
//   - completion.go — Tab autocomplete
//   - seed.go      — fake channels + the @boggy username
package lobby

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/bchayka/gitstatus/internal/field"
	"github.com/bchayka/gitstatus/internal/screens"
	"github.com/bchayka/gitstatus/internal/theme"
	"github.com/bchayka/gitstatus/internal/ui"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TickMsg fires at ~60fps while the lobby is active. Drives the backdrop
// field engine.
type TickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(16*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// Screen is the lobby's own state. The chat input is always focused, so this
// screen reports InputFocused()==true and the app's global key handlers stay
// out of the way.
type Screen struct {
	channels   []Channel
	chatActive int
	chatScroll int

	input      textinput.Model
	history    []string
	historyIdx int
	paletteIdx int // selection inside the command palette popup

	joinPosted bool

	// engine drives the field backdrop behind the chat scrollback. Each
	// keystroke pulses its motion accumulator so typing produces palette
	// drift the same way mouse motion does on the ertdfgcvb site.
	engine *field.Engine
}

// New constructs a fresh lobby with seeded channels and a focused input.
func New() *Screen {
	return &Screen{
		channels:   seedChannels(),
		chatActive: 0,
		input:      newInput(),
		engine:     field.NewEngine(),
	}
}

func newInput() textinput.Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.CharLimit = 0
	ti.Placeholder = "message #lobby or type /help"
	ti.Focus()
	return ti
}

func (s *Screen) Init() tea.Cmd { return tea.Batch(textinput.Blink, tickCmd()) }

func (s *Screen) Name() string  { return screens.LobbyID }
func (s *Screen) Title() string { return "lobby" }

func (s *Screen) HeaderContext() string {
	ch := s.activeChannel()
	if ch == nil {
		return ""
	}
	sep := lipgloss.NewStyle().Foreground(theme.Muted).Render(" · ")
	return lipgloss.NewStyle().Foreground(theme.OK).Render(ch.Name) + sep +
		lipgloss.NewStyle().Foreground(theme.Muted).Render(fmt.Sprintf("%d online", ch.Online))
}

func (s *Screen) Footer() []screens.KeyHint {
	if s.paletteVisible() {
		return []screens.KeyHint{
			{Key: "↑/↓", Desc: "navigate"},
			{Key: "tab", Desc: "fill"},
			{Key: "enter", Desc: "run"},
			{Key: "esc", Desc: "cancel"},
		}
	}
	return []screens.KeyHint{
		{Key: "enter", Desc: "send"},
		{Key: "/", Desc: "commands"},
		{Key: "↑/↓", Desc: "history"},
		{Key: "pgup/pgdn", Desc: "scroll"},
		{Key: "ctrl+c", Desc: "quit"},
	}
}

func (s *Screen) InputFocused() bool { return true }

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (s *Screen) Update(msg tea.Msg) (screens.Screen, tea.Cmd) {
	switch m := msg.(type) {
	case tea.KeyMsg:
		// Every keystroke is a small pulse of activity that drives the
		// engine's motion accumulator. Without this, the field would sit
		// gray during chat because the mouse isn't moving.
		s.engine.Pulse(0.04)
		return s.handleKey(m)
	case tea.MouseMsg:
		s.engine.SetCursor(float64(m.X), float64(m.Y))
		return s, nil
	case TickMsg:
		s.engine.Tick(time.Time(m))
		return s, tickCmd()
	}
	return s, nil
}

func (s *Screen) handleKey(msg tea.KeyMsg) (screens.Screen, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return s, tea.Quit
	case "enter":
		// When the palette is open, Enter accepts the highlighted command —
		// it's inserted in the input then submitted in one keystroke, matching
		// the muscle memory of Claude Code / VS Code command palettes.
		if cmd := s.acceptPalette(); cmd != "" {
			s.input.SetValue(cmd)
		}
		s.resetPalette()
		return s.submit()
	case "tab":
		// Tab fills the input without submitting, so users can pick a command
		// like /join and then type the channel name.
		if s.paletteVisible() {
			s.fillPalette()
		}
		return s, nil
	case "shift+tab":
		if s.paletteVisible() {
			s.movePalette(-1)
		}
		return s, nil
	case "up":
		if s.paletteVisible() {
			s.movePalette(-1)
		} else {
			s.recallHistory(-1)
		}
		return s, nil
	case "down":
		if s.paletteVisible() {
			s.movePalette(+1)
		} else {
			s.recallHistory(+1)
		}
		return s, nil
	case "pgup":
		s.chatScroll += 5
		return s, nil
	case "pgdown":
		s.chatScroll -= 5
		if s.chatScroll < 0 {
			s.chatScroll = 0
		}
		return s, nil
	case "esc":
		s.input.SetValue("")
		s.resetPalette()
		return s, nil
	}
	// Any other key edits the input → the filtered match list will change, so
	// reset the highlight back to the top of the new list.
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	s.resetPalette()
	return s, cmd
}

func (s *Screen) submit() (screens.Screen, tea.Cmd) {
	text := strings.TrimSpace(s.input.Value())
	s.input.SetValue("")
	if text == "" {
		return s, nil
	}
	s.history = append(s.history, text)
	s.historyIdx = len(s.history)

	if strings.HasPrefix(text, "/") {
		ss, cmd := s.handleSlash(text)
		return ss, cmd
	}
	s.postUser(text)
	return s, nil
}

func (s *Screen) recallHistory(delta int) {
	if len(s.history) == 0 {
		return
	}
	switch {
	case delta < 0:
		if s.historyIdx > 0 {
			s.historyIdx--
		}
	case delta > 0:
		if s.historyIdx < len(s.history) {
			s.historyIdx++
		}
	}
	if s.historyIdx >= len(s.history) {
		s.input.SetValue("")
		return
	}
	s.input.SetValue(s.history[s.historyIdx])
	s.input.CursorEnd()
}

// ---------------------------------------------------------------------------
// Posting helpers — used by both regular sends and slash command handlers
// ---------------------------------------------------------------------------

func (s *Screen) activeChannel() *Channel {
	if s.chatActive < 0 || s.chatActive >= len(s.channels) {
		return nil
	}
	return &s.channels[s.chatActive]
}

func (s *Screen) postUser(body string) {
	ch := s.activeChannel()
	if ch == nil {
		return
	}
	ch.Messages = append(ch.Messages, ui.ChatMessage{
		Author: MeUser, Body: body, At: time.Now(),
	})
	s.chatScroll = 0
}

func (s *Screen) postSystem(body string) {
	ch := s.activeChannel()
	if ch == nil {
		return
	}
	for _, line := range strings.Split(body, "\n") {
		ch.Messages = append(ch.Messages, ui.ChatMessage{
			Author: "*", Body: line, At: time.Now(), Kind: ui.ChatSystem,
		})
	}
	s.chatScroll = 0
}

// EnsureJoined posts the "entered the chat" join message once, then no-ops on
// subsequent calls. Called by the router when transitioning from intro.
func (s *Screen) EnsureJoined() {
	if s.joinPosted {
		return
	}
	if ch := s.activeChannel(); ch != nil {
		ch.Messages = append(ch.Messages, ui.ChatMessage{
			Author: MeUser, Body: "entered the chat", At: time.Now(), Kind: ui.ChatJoin,
		})
	}
	s.joinPosted = true
	s.chatScroll = 0
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (s *Screen) View(width, height int) string {
	w := ui.FeedShellWidth(width)
	contentW := w - 2

	if s.chatActive < 0 || s.chatActive >= len(s.channels) {
		s.chatActive = 0
	}
	ch := s.channels[s.chatActive]

	bar := topBar(ch, contentW)
	barH := lipgloss.Height(bar)

	prompt := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true).
		Render("[" + strings.TrimPrefix(MeUser, "@") + "]> ")
	s.input.Width = contentW - lipgloss.Width(prompt) - 1
	inputLine := prompt + s.input.View()

	palette := s.renderPalette(contentW)
	paletteH := s.paletteHeight()

	// Layout (bottom-anchored input):
	//   bar (1) · divider (1) · scrollback (chatH) · divider (1) · palette (paletteH) · input (1)
	// Total fixed chrome is 4 rows; scrollback flexes around it.
	chatH := height - barH - paletteH - 4
	if chatH < 4 {
		chatH = 4
	}

	var lines []string
	for _, msg := range ch.Messages {
		lines = append(lines, ui.RenderChatLine(msg, contentW)...)
	}
	chatRows := scrollbackRows(lines, chatH, s.chatScroll)

	// Render the field engine at the scrollback's exact size, then composite
	// the chat over it row-by-row. Empty rows (no chat there yet) show the
	// field; rows with chat content win out.
	s.engine.Resize(contentW, chatH)
	fieldRender := s.engine.Render()
	fieldRows := strings.Split(fieldRender, "\n")
	visible := compositeChatOverField(chatRows, fieldRows, chatH)

	parts := []string{
		bar,
		ui.Divider(contentW),
		visible,
		ui.Divider(contentW),
	}
	if palette != "" {
		parts = append(parts, palette)
	}
	parts = append(parts, inputLine)
	stacked := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return lipgloss.Place(width, height, lipgloss.Left, lipgloss.Top, stacked)
}

func topBar(ch Channel, width int) string {
	chName := lipgloss.NewStyle().Foreground(theme.Accent2).Bold(true).Render(ch.Name)
	online := lipgloss.NewStyle().Foreground(theme.OK).Render(fmt.Sprintf("%d online", ch.Online))
	topic := lipgloss.NewStyle().Foreground(theme.Muted).Italic(true).
		Render("topic: " + ch.Topic)
	sep := lipgloss.NewStyle().Foreground(theme.Muted).Render("  ·  ")
	left := chName + sep + online + sep + topic
	if lipgloss.Width(left) > width {
		left = ui.Truncate(left, width)
	}
	return left
}

// scrollbackRows clamps the scroll offset and returns the visible slice as a
// height-length string slice. Empty positions are empty strings so the
// compositor can show the field behind them.
func scrollbackRows(lines []string, height, scroll int) []string {
	maxScroll := len(lines) - height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}
	end := len(lines) - scroll
	start := end - height
	if start < 0 {
		start = 0
	}
	if end < start {
		end = start
	}
	if end > len(lines) {
		end = len(lines)
	}
	visible := lines[start:end]
	// Bottom-align: pad the TOP with empty rows so chat hugs the bottom of
	// the scrollback area. Those empty rows are where the field shows.
	if len(visible) < height {
		pad := make([]string, height-len(visible))
		visible = append(pad, visible...)
	}
	return visible
}

// ansiRegex strips ANSI SGR sequences for the emptiness check. Matching a
// row as "empty" means "no visible chars" — pure whitespace or only escape
// codes.
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func rowIsEmpty(row string) bool {
	stripped := ansiRegex.ReplaceAllString(row, "")
	return strings.TrimSpace(stripped) == ""
}

// compositeChatOverField returns a single string of `height` rows where each
// row is either the chat row (if it has visible content) or the field row.
// Per-row composition: simpler than per-cell and good enough for v1 because
// chat lines are typically content-filled or fully empty, not partial.
func compositeChatOverField(chatRows, fieldRows []string, height int) string {
	out := make([]string, height)
	for i := 0; i < height; i++ {
		var chat, fld string
		if i < len(chatRows) {
			chat = chatRows[i]
		}
		if i < len(fieldRows) {
			fld = fieldRows[i]
		}
		if rowIsEmpty(chat) {
			out[i] = fld
		} else {
			out[i] = chat
		}
	}
	return strings.Join(out, "\n")
}
