package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const meUser = "@boggy"

// lobbyCommands is the canonical list of slash commands surfaced by
// autocomplete. Aliases (e.g. /commits → /discussions) still work in
// handleSlash but are deliberately omitted here to keep the suggestion strip
// short.
var lobbyCommands = []struct{ name, desc string }{
	{"/news", "open news feed"},
	{"/spotlight", "open featured project"},
	{"/resources", "open skills & github repos"},
	{"/games", "open mini-games"},
	{"/discussions", "open commit feed"},
	{"/join", "join or switch channel"},
	{"/leave", "return to #lobby"},
	{"/list", "list channels"},
	{"/who", "list users"},
	{"/topic", "view or set topic"},
	{"/me", "third-person action"},
	{"/clear", "clear scrollback"},
	{"/help", "show all commands"},
	{"/quit", "exit chaosbyte"},
}

// matchCommands returns commands whose names start with prefix. Empty if the
// prefix isn't a slash command at all.
func matchCommands(prefix string) []string {
	if !strings.HasPrefix(prefix, "/") {
		return nil
	}
	var out []string
	for _, c := range lobbyCommands {
		if strings.HasPrefix(c.name, prefix) {
			out = append(out, c.name)
		}
	}
	return out
}

func commandDesc(name string) string {
	for _, c := range lobbyCommands {
		if c.name == name {
			return c.desc
		}
	}
	return ""
}

func newLobbyInput() textinput.Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.CharLimit = 0
	ti.Placeholder = "message #lobby or type /help"
	ti.Focus()
	return ti
}

// lobbyJoin posts the join system message to the current channel.
func (m *model) lobbyJoin() {
	if m.chatActive < 0 || m.chatActive >= len(m.channels) {
		m.chatActive = 0
	}
	ch := &m.channels[m.chatActive]
	if m.joinPosted {
		return
	}
	ch.Messages = append(ch.Messages, ChatMessage{
		Author: meUser, Body: "entered the chat", At: time.Now(), Kind: ChatJoin,
	})
	m.joinPosted = true
	m.chatScroll = 0
}

func (m model) updateLobby(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "enter":
		m.resetCompletion()
		return m.lobbySubmit()
	case "tab":
		return m.cycleCompletion(1), nil
	case "shift+tab":
		return m.cycleCompletion(-1), nil
	case "up":
		// history recall — newer entry
		if len(m.lobbyHistory) == 0 {
			return m, nil
		}
		if m.historyIdx > 0 {
			m.historyIdx--
		}
		if m.historyIdx < len(m.lobbyHistory) {
			m.lobbyInput.SetValue(m.lobbyHistory[m.historyIdx])
			m.lobbyInput.CursorEnd()
		}
		return m, nil
	case "down":
		if len(m.lobbyHistory) == 0 {
			return m, nil
		}
		if m.historyIdx < len(m.lobbyHistory) {
			m.historyIdx++
		}
		if m.historyIdx >= len(m.lobbyHistory) {
			m.lobbyInput.SetValue("")
		} else {
			m.lobbyInput.SetValue(m.lobbyHistory[m.historyIdx])
			m.lobbyInput.CursorEnd()
		}
		return m, nil
	case "pgup":
		m.chatScroll += 5
		return m, nil
	case "pgdown":
		m.chatScroll -= 5
		if m.chatScroll < 0 {
			m.chatScroll = 0
		}
		return m, nil
	case "esc":
		m.lobbyInput.SetValue("")
		m.resetCompletion()
		return m, nil
	}
	// any other key edits the input → invalidate the completion cycle so the
	// next Tab restarts from the new stem
	m.resetCompletion()
	var cmd tea.Cmd
	m.lobbyInput, cmd = m.lobbyInput.Update(msg)
	return m, cmd
}

func (m *model) resetCompletion() {
	m.completionStem = ""
	m.completionIdx = -1
}

// cycleCompletion replaces the lobby input with the next (or previous) match
// of the active completion stem. The stem is captured the first time Tab is
// pressed since the input last changed, so successive Tabs walk through all
// matches without narrowing as the input changes underneath.
func (m model) cycleCompletion(delta int) tea.Model {
	cur := m.lobbyInput.Value()
	if m.completionStem == "" || m.completionIdx < 0 {
		m.completionStem = cur
	}
	matches := matchCommands(m.completionStem)
	if len(matches) == 0 {
		return m
	}
	if m.completionIdx < 0 {
		if delta > 0 {
			m.completionIdx = 0
		} else {
			m.completionIdx = len(matches) - 1
		}
	} else {
		m.completionIdx = (m.completionIdx + delta + len(matches)) % len(matches)
	}
	m.lobbyInput.SetValue(matches[m.completionIdx])
	m.lobbyInput.CursorEnd()
	return m
}

func (m model) lobbySubmit() (tea.Model, tea.Cmd) {
	text := strings.TrimSpace(m.lobbyInput.Value())
	m.lobbyInput.SetValue("")
	if text == "" {
		return m, nil
	}
	m.lobbyHistory = append(m.lobbyHistory, text)
	m.historyIdx = len(m.lobbyHistory)

	if strings.HasPrefix(text, "/") {
		return m.handleSlash(text)
	}
	m.postUser(text)
	return m, nil
}

func (m *model) postUser(body string) {
	if m.chatActive < 0 || m.chatActive >= len(m.channels) {
		return
	}
	m.channels[m.chatActive].Messages = append(m.channels[m.chatActive].Messages, ChatMessage{
		Author: meUser, Body: body, At: time.Now(),
	})
	m.chatScroll = 0
}

func (m *model) postSystem(body string) {
	if m.chatActive < 0 || m.chatActive >= len(m.channels) {
		return
	}
	for _, line := range strings.Split(body, "\n") {
		m.channels[m.chatActive].Messages = append(m.channels[m.chatActive].Messages, ChatMessage{
			Author: "*", Body: line, At: time.Now(), Kind: ChatSystem,
		})
	}
	m.chatScroll = 0
}

func (m model) handleSlash(text string) (tea.Model, tea.Cmd) {
	parts := strings.Fields(text)
	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "/news":
		m.screen = screenNews
	case "/spotlight":
		m.screen = screenSpotlight
	case "/resources", "/skills":
		m.screen = screenResources
	case "/games":
		m.screen = screenGames
	case "/discussions", "/commits", "/feed":
		m.screen = screenDiscussions
	case "/help", "/?":
		m.postSystem(slashHelpText())
	case "/clear":
		if m.chatActive >= 0 && m.chatActive < len(m.channels) {
			m.channels[m.chatActive].Messages = nil
		}
	case "/quit", "/exit", "/bye":
		return m, tea.Quit
	case "/me":
		body := strings.Join(args, " ")
		if body == "" {
			return m, nil
		}
		if m.chatActive >= 0 && m.chatActive < len(m.channels) {
			m.channels[m.chatActive].Messages = append(
				m.channels[m.chatActive].Messages,
				ChatMessage{Author: meUser, Body: body, At: time.Now(), Kind: ChatAction},
			)
			m.chatScroll = 0
		}
	case "/join":
		if len(args) == 0 {
			m.postSystem("usage: /join #channel")
			break
		}
		name := args[0]
		if !strings.HasPrefix(name, "#") {
			name = "#" + name
		}
		idx := -1
		for i, ch := range m.channels {
			if ch.Name == name {
				idx = i
				break
			}
		}
		if idx < 0 {
			m.channels = append(m.channels, Channel{
				Name: name, Topic: "(freshly minted) — claim a topic with /topic",
				Members: 1, Online: 1,
			})
			idx = len(m.channels) - 1
		}
		m.chatActive = idx
		m.chatScroll = 0
		m.channels[idx].Messages = append(m.channels[idx].Messages, ChatMessage{
			Author: meUser, Body: "joined " + name, At: time.Now(), Kind: ChatJoin,
		})
	case "/leave", "/part":
		if m.chatActive > 0 {
			leaving := m.channels[m.chatActive].Name
			m.chatActive = 0
			m.chatScroll = 0
			m.postSystem(meUser + " left " + leaving)
		} else {
			m.postSystem("you're already in #lobby — try /quit to exit")
		}
	case "/list", "/channels":
		var lines []string
		lines = append(lines, "channels:")
		for _, ch := range m.channels {
			lines = append(lines, fmt.Sprintf("  %-20s  %4d online · %s", ch.Name, ch.Online, truncate(ch.Topic, 36)))
		}
		m.postSystem(strings.Join(lines, "\n"))
	case "/who", "/users":
		m.postSystem("active in " + m.channels[m.chatActive].Name + ": @yamlhater @nullpointer @devops_bard @junior_dev @standup_ghost @vibe_master @ai_grifter @senior_intern @recovering_pm @borrow_checker @corporate_villain @boggy")
	case "/topic":
		body := strings.Join(args, " ")
		if body == "" {
			m.postSystem("topic: " + m.channels[m.chatActive].Topic)
			break
		}
		m.channels[m.chatActive].Topic = body
		m.postSystem(meUser + " set topic: " + body)
	case "/nick":
		m.postSystem("nick is fixed at " + meUser + " in this build")
	default:
		m.postSystem(fmt.Sprintf("unknown command %q — try /help", cmd))
	}
	return m, nil
}

func slashHelpText() string {
	return strings.Join([]string{
		"available commands:",
		"  /news          open news feed",
		"  /spotlight     open featured project + live chat",
		"  /resources     open skills + github repos",
		"  /games         open mini-games",
		"  /discussions   open commit feed",
		"  /join #name    join or switch channel",
		"  /leave         return to #lobby",
		"  /list          list channels",
		"  /who           list users in this channel",
		"  /topic [text]  view or set channel topic",
		"  /me <action>   third-person action",
		"  /clear         clear scrollback",
		"  /help          show this list",
		"  /quit          exit chaosbyte",
		"navigation: esc returns to lobby from any screen · ctrl+c quits",
	}, "\n")
}

// ============================================================================
// Rendering
// ============================================================================

func (m model) renderLobby(width, height int) string {
	w := feedShellWidth(width)
	contentW := w - 2

	if m.chatActive < 0 || m.chatActive >= len(m.channels) {
		m.chatActive = 0
	}
	ch := m.channels[m.chatActive]

	bar := lobbyTopBar(ch, contentW)
	barH := lipgloss.Height(bar)

	promptText := "[" + strings.TrimPrefix(meUser, "@") + "]> "
	prompt := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render(promptText)
	m.lobbyInput.Width = contentW - lipgloss.Width(prompt) - 1
	input := prompt + m.lobbyInput.View()
	inputH := 1

	completions := m.renderCompletionStrip(contentW)
	completionsH := 0
	if completions != "" {
		completionsH = 1
	}

	chatH := height - barH - inputH - 3 - completionsH
	if chatH < 4 {
		chatH = 4
	}

	var lines []string
	for _, msg := range ch.Messages {
		lines = append(lines, renderLobbyLine(msg, contentW)...)
	}
	maxScroll := len(lines) - chatH
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.chatScroll > maxScroll {
		m.chatScroll = maxScroll
	}
	end := len(lines) - m.chatScroll
	start := end - chatH
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
	visible = padToHeight(visible, chatH)

	parts := []string{
		bar,
		dividerLine(contentW),
		visible,
		dividerLine(contentW),
	}
	if completions != "" {
		parts = append(parts, completions)
	}
	parts = append(parts, input)
	stacked := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Top, stacked)
}

// renderCompletionStrip shows matching slash commands above the input. It hides
// itself when there's nothing useful to suggest (no slash prefix, no matches,
// or the input already equals the only match).
func (m model) renderCompletionStrip(width int) string {
	cur := m.lobbyInput.Value()
	matches := matchCommands(cur)
	if len(matches) == 0 {
		return ""
	}
	if len(matches) == 1 && matches[0] == cur {
		return ""
	}

	label := lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render("tab ")

	// When the user has narrowed to a single match, show its description too.
	if len(matches) == 1 {
		name := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render(matches[0])
		desc := lipgloss.NewStyle().Foreground(colorMuted).Render("  " + commandDesc(matches[0]))
		return label + name + desc
	}

	const inlineCap = 10
	var chips []string
	for i, name := range matches {
		if i >= inlineCap {
			chips = append(chips, lipgloss.NewStyle().Foreground(colorMuted).
				Render(fmt.Sprintf("+%d more", len(matches)-inlineCap)))
			break
		}
		style := lipgloss.NewStyle().Foreground(colorAccent)
		if i == m.completionIdx {
			style = style.Bold(true).Underline(true)
		}
		chips = append(chips, style.Render(name))
	}
	return label + strings.Join(chips, "  ")
}

func lobbyTopBar(ch Channel, width int) string {
	chName := lipgloss.NewStyle().Foreground(colorAccent2).Bold(true).Render(ch.Name)
	online := lipgloss.NewStyle().Foreground(colorOk).Render(fmt.Sprintf("%d online", ch.Online))
	topic := lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
		Render("topic: " + ch.Topic)
	left := chName + sepDot() + online + sepDot() + topic
	// truncate if necessary
	if lipgloss.Width(left) > width {
		left = truncate(left, width)
	}
	return left
}

func sepDot() string {
	return lipgloss.NewStyle().Foreground(colorMuted).Render("  ·  ")
}

func renderLobbyLine(msg ChatMessage, width int) []string {
	ts := commitTimeStyle.Render(humanizeTime(msg.At))
	var prefix string
	var bodyStyle lipgloss.Style
	body := msg.Body

	switch msg.Kind {
	case ChatJoin:
		prefix = lipgloss.NewStyle().Foreground(colorOk).Bold(true).Render("-->")
		bodyStyle = lipgloss.NewStyle().Foreground(colorOk).Italic(true)
		body = msg.Author + " " + body
	case ChatSystem:
		prefix = lipgloss.NewStyle().Foreground(colorMuted).Render("*")
		bodyStyle = lipgloss.NewStyle().Foreground(colorMuted).Italic(true)
	case ChatAction:
		prefix = lipgloss.NewStyle().Foreground(colorAccent2).Bold(true).Render("*")
		bodyStyle = lipgloss.NewStyle().Foreground(colorAccent2).Italic(true)
		body = msg.Author + " " + body
	default:
		nick := strings.TrimPrefix(msg.Author, "@")
		nickColor := nickHash(nick)
		prefix = lipgloss.NewStyle().Foreground(nickColor).Render("<" + nick + ">")
		bodyStyle = lipgloss.NewStyle().Foreground(colorFg)
	}

	header := ts + " " + prefix + " "
	headerW := lipgloss.Width(header)
	bodyW := width - headerW - 2
	if bodyW < 12 {
		bodyW = 12
	}
	wrapped := wrap(body, bodyW)
	parts := strings.Split(wrapped, "\n")

	var out []string
	out = append(out, header+bodyStyle.Render(parts[0]))
	pad := strings.Repeat(" ", headerW)
	for _, p := range parts[1:] {
		out = append(out, pad+bodyStyle.Render(p))
	}
	return out
}

// nickHash assigns a deterministic accent color per nickname so authors are
// visually distinct in dense scrollback.
func nickHash(nick string) lipgloss.Color {
	palette := []lipgloss.Color{
		colorAccent, colorAccent2, colorOk, colorWarn, colorLike,
	}
	h := 0
	for _, c := range nick {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return palette[h%len(palette)]
}
