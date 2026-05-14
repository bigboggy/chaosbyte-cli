// Package lobby is the chat screen. It renders channels and messages from a
// shared *hub.Hub, owning only session-local UI state (input, history, scroll,
// active channel, identity).
//
// Files in this package:
//   - lobby.go     — Screen type, Init/Update/View, hub subscription
//   - commands.go  — slash command registry + per-command handlers
//   - completion.go — Tab autocomplete
package lobby

import (
	"fmt"
	"strings"

	"github.com/bchayka/gitstatus/internal/auth"
	"github.com/bchayka/gitstatus/internal/hub"
	"github.com/bchayka/gitstatus/internal/screens"
	"github.com/bchayka/gitstatus/internal/theme"
	"github.com/bchayka/gitstatus/internal/ui"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Screen is the lobby's session-local state. Channels and messages live in the
// hub; the lobby reads from it during View and on every hub Event.
type Screen struct {
	hub    *hub.Hub
	subID  uint64
	events <-chan hub.Event

	auth        *auth.Service // nil disables /auth
	fingerprint string        // SSH pubkey fingerprint; "" if no key

	meUser     string
	activeName string // currently-viewed channel name; defaults to "#lobby"
	chatScroll int

	input      textinput.Model
	history    []string
	historyIdx int
	paletteIdx int

	joinPosted bool
	authFlow   *authFlowState
}

// New constructs a lobby bound to hub. meUser is the participant's display
// handle (e.g. "@boggy"). fingerprint is the SSH pubkey fingerprint used to
// associate this session with a stored identity (may be ""). authSvc may be
// nil to disable /auth. The session subscribes to the hub immediately; call
// Cleanup when the session ends.
func New(meUser, fingerprint string, h *hub.Hub, authSvc *auth.Service) *Screen {
	id, events := h.Subscribe()
	h.SetViewing(id, "#lobby")
	return &Screen{
		hub:         h,
		subID:       id,
		events:      events,
		auth:        authSvc,
		fingerprint: fingerprint,
		meUser:      meUser,
		activeName:  "#lobby",
		input:       newInput(),
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

// waitForEvent returns a Cmd that blocks until the next hub event lands. The
// Cmd must be re-issued after each event so the session keeps listening.
func waitForEvent(ch <-chan hub.Event) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return nil
		}
		return ev
	}
}

func (s *Screen) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, waitForEvent(s.events))
}

func (s *Screen) Name() string  { return screens.LobbyID }
func (s *Screen) Title() string { return "lobby" }

func (s *Screen) HeaderContext() string {
	name := s.activeName
	sep := lipgloss.NewStyle().Foreground(theme.Muted).Render(" · ")
	return lipgloss.NewStyle().Foreground(theme.OK).Render(name) + sep +
		lipgloss.NewStyle().Foreground(theme.Muted).Render(fmt.Sprintf("%d online", s.hub.Online(name)))
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

// Cleanup unsubscribes from the hub and cancels any in-flight auth flow.
// Called by app.Cleanup when the session ends; safe to call more than once.
func (s *Screen) Cleanup() {
	s.cancelAuthFlow()
	if s.hub != nil {
		s.hub.Unsubscribe(s.subID)
	}
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (s *Screen) Update(msg tea.Msg) (screens.Screen, tea.Cmd) {
	switch m := msg.(type) {
	case hub.Event:
		// Re-subscribe for the next event. The screen rerenders automatically;
		// View pulls fresh data from the hub.
		return s, waitForEvent(s.events)
	case authStartedMsg:
		return s.handleAuthStarted(m)
	case authResultMsg:
		return s.handleAuthResult(m)
	case tea.KeyMsg:
		return s.handleKey(m)
	default:
		_ = m
	}
	return s, nil
}

func (s *Screen) handleKey(msg tea.KeyMsg) (screens.Screen, tea.Cmd) {
	// While the auth modal is up, swallow everything except cancel/quit.
	if s.authFlow != nil {
		switch msg.String() {
		case "ctrl+c":
			s.cancelAuthFlow()
			return s, tea.Quit
		case "esc":
			s.cancelAuthFlow()
		}
		return s, nil
	}

	switch msg.String() {
	case "ctrl+c":
		return s, tea.Quit
	case "enter":
		if cmd := s.acceptPalette(); cmd != "" {
			s.input.SetValue(cmd)
		}
		s.resetPalette()
		return s.submit()
	case "tab":
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
		return s.handleSlash(text)
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
// Posting helpers — push through the hub so all sessions see the message.
// ---------------------------------------------------------------------------

// postUser sends a normal message from this session's user into the active
// channel. Snaps scroll to bottom so our own send is visible.
func (s *Screen) postUser(body string) {
	s.hub.Post(s.activeName, s.meUser, body, ui.ChatNormal)
	s.chatScroll = 0
}

// postSystem posts a system message that only THIS session sees. Used for
// command output (e.g. /help, /list responses) — these aren't broadcast.
//
// Because the screen has no local message buffer, system messages would be
// invisible without a stash. We funnel them into the hub as a transient
// "system" message visible to everyone, which is intentional: the chat is
// public and seeing other people's /help output is part of the vibe.
func (s *Screen) postSystem(body string) {
	for _, line := range strings.Split(body, "\n") {
		s.hub.Post(s.activeName, "*", line, ui.ChatSystem)
	}
	s.chatScroll = 0
}

// EnsureJoined posts the "entered the chat" join message once.
func (s *Screen) EnsureJoined() {
	if s.joinPosted {
		return
	}
	s.hub.Post(s.activeName, s.meUser, "entered the chat", ui.ChatJoin)
	s.joinPosted = true
	s.chatScroll = 0
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (s *Screen) View(width, height int) string {
	if s.authFlow != nil {
		return s.renderAuthModal(width, height)
	}

	w := ui.FeedShellWidth(width)
	contentW := w - 2

	names := s.hub.ChannelNames()
	if !s.hub.HasChannel(s.activeName) && len(names) > 0 {
		s.activeName = names[0]
		s.hub.SetViewing(s.subID, s.activeName)
	}

	bar := topBar(s.activeName, s.hub.Online(s.activeName), contentW)
	barH := lipgloss.Height(bar)

	prompt := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true).
		Render("[" + strings.TrimPrefix(s.meUser, "@") + "]> ")
	s.input.Width = contentW - lipgloss.Width(prompt) - 1
	inputLine := prompt + s.input.View()

	palette := s.renderPalette(contentW)
	paletteH := s.paletteHeight()

	chatH := height - barH - paletteH - 4
	if chatH < 4 {
		chatH = 4
	}

	msgs, _ := s.hub.Messages(s.activeName)
	var lines []string
	for _, msg := range msgs {
		lines = append(lines, ui.RenderChatLine(msg, contentW)...)
	}
	visible := windowScrollback(lines, chatH, s.chatScroll)

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

func topBar(name string, online, width int) string {
	chName := lipgloss.NewStyle().Foreground(theme.Accent2).Bold(true).Render(name)
	onlineStr := lipgloss.NewStyle().Foreground(theme.OK).Render(fmt.Sprintf("%d online", online))
	sep := lipgloss.NewStyle().Foreground(theme.Muted).Render("  ·  ")
	left := chName + sep + onlineStr
	if lipgloss.Width(left) > width {
		left = ui.Truncate(left, width)
	}
	return left
}

// windowScrollback clamps the scroll offset and returns the visible slice
// padded to exactly height rows so the input below it stays anchored.
func windowScrollback(lines []string, height, scroll int) string {
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
	var visible string
	if start < end {
		visible = strings.Join(lines[start:end], "\n")
	}
	return ui.PadToHeight(visible, height)
}
