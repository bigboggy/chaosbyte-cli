package lobby

import (
	"fmt"
	"strings"
	"time"

	"github.com/bchayka/gitstatus/internal/screens"
	"github.com/bchayka/gitstatus/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

// command is one entry in the slash-command catalog. handlers live in the
// dispatch switch rather than as function pointers on this struct, which lets
// handlers reference `builtins` without creating a package-level init cycle.
type command struct {
	name string
	desc string
}

// builtins is the canonical list of slash commands. Order here is the order
// shown in autocomplete. Aliases are wired in `aliases` and don't appear here
// to keep the suggestion strip tidy.
var builtins = []command{
	{"/news", "open news feed"},
	{"/spotlight", "open featured project"},
	{"/resources", "open skills & github repos"},
	{"/games", "open mini-games"},
	{"/discussions", "open commit feed"},
	{"/ambient", "open ambient field"},
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

// aliases maps an alternate spelling to its canonical command name. Aliases
// don't show in autocomplete.
var aliases = map[string]string{
	"/skills":   "/resources",
	"/commits":  "/discussions",
	"/feed":     "/discussions",
	"/exit":     "/quit",
	"/bye":      "/quit",
	"/part":     "/leave",
	"/channels": "/list",
	"/users":    "/who",
	"/?":        "/help",
}

// canonicalName resolves aliases to their primary command name.
func canonicalName(name string) string {
	if a, ok := aliases[name]; ok {
		return a
	}
	return name
}

// handleSlash parses and dispatches a typed slash command. Unknown commands
// post a system message back to chat.
func (s *Screen) handleSlash(text string) (*Screen, tea.Cmd) {
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return s, nil
	}
	name := canonicalName(parts[0])
	args := parts[1:]

	switch name {
	case "/news":
		return s, screens.Navigate(screens.NewsID)
	case "/spotlight":
		return s, screens.Navigate(screens.SpotlightID)
	case "/resources":
		return s, screens.Navigate(screens.ResourcesID)
	case "/games":
		return s, screens.Navigate(screens.GamesID)
	case "/discussions":
		return s, screens.Navigate(screens.DiscussionsID)
	case "/ambient":
		return s, screens.Navigate(screens.AmbientID)
	case "/help":
		return s.cmdHelp()
	case "/clear":
		return s.cmdClear()
	case "/quit":
		return s, screens.Quit()
	case "/me":
		return s.cmdMe(args)
	case "/join":
		return s.cmdJoin(args)
	case "/leave":
		return s.cmdLeave()
	case "/list":
		return s.cmdList()
	case "/who":
		return s.cmdWho()
	case "/topic":
		return s.cmdTopic(args)
	}
	s.postSystem(fmt.Sprintf("unknown command %q — try /help", parts[0]))
	return s, nil
}

// ---------------------------------------------------------------------------
// Per-command handlers, kept as methods on *Screen so they have direct access
// to the channel list and posting helpers.
// ---------------------------------------------------------------------------

func (s *Screen) cmdHelp() (*Screen, tea.Cmd) {
	lines := []string{"available commands:"}
	for _, c := range builtins {
		lines = append(lines, fmt.Sprintf("  %-13s %s", c.name, c.desc))
	}
	lines = append(lines, "navigation: esc returns to lobby from any screen · ctrl+c quits")
	s.postSystem(strings.Join(lines, "\n"))
	return s, nil
}

func (s *Screen) cmdClear() (*Screen, tea.Cmd) {
	if ch := s.activeChannel(); ch != nil {
		ch.Messages = nil
	}
	return s, nil
}

func (s *Screen) cmdMe(args []string) (*Screen, tea.Cmd) {
	body := strings.Join(args, " ")
	if body == "" {
		return s, nil
	}
	if ch := s.activeChannel(); ch != nil {
		ch.Messages = append(ch.Messages, ui.ChatMessage{
			Author: s.nick, Body: body, At: time.Now(), Kind: ui.ChatAction,
		})
		s.chatScroll = 0
	}
	return s, nil
}

func (s *Screen) cmdJoin(args []string) (*Screen, tea.Cmd) {
	if len(args) == 0 {
		s.postSystem("usage: /join #channel")
		return s, nil
	}
	name := args[0]
	if !strings.HasPrefix(name, "#") {
		name = "#" + name
	}
	idx := -1
	for i, ch := range s.channels {
		if ch.Name == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		s.channels = append(s.channels, Channel{
			Name: name, Topic: "(freshly minted) — claim a topic with /topic",
			Members: 1, Online: 1,
		})
		idx = len(s.channels) - 1
	}
	s.chatActive = idx
	s.chatScroll = 0
	s.channels[idx].Messages = append(s.channels[idx].Messages, ui.ChatMessage{
		Author: s.nick, Body: "joined " + name, At: time.Now(), Kind: ui.ChatJoin,
	})
	return s, nil
}

func (s *Screen) cmdLeave() (*Screen, tea.Cmd) {
	if s.chatActive > 0 {
		leaving := s.channels[s.chatActive].Name
		s.chatActive = 0
		s.chatScroll = 0
		s.postSystem(s.nick + " left " + leaving)
		return s, nil
	}
	s.postSystem("you're already in #lobby — try /quit to exit")
	return s, nil
}

func (s *Screen) cmdList() (*Screen, tea.Cmd) {
	lines := []string{"channels:"}
	for _, ch := range s.channels {
		lines = append(lines, fmt.Sprintf("  %-20s  %4d online · %s",
			ch.Name, ch.Online, truncate(ch.Topic, 36)))
	}
	s.postSystem(strings.Join(lines, "\n"))
	return s, nil
}

func (s *Screen) cmdWho() (*Screen, tea.Cmd) {
	ch := s.activeChannel()
	if ch == nil {
		return s, nil
	}
	s.postSystem("active in " + ch.Name + ": @yamlhater @nullpointer @devops_bard @junior_dev @standup_ghost @vibe_master @ai_grifter @senior_intern @recovering_pm @borrow_checker @corporate_villain @boggy")
	return s, nil
}

func (s *Screen) cmdTopic(args []string) (*Screen, tea.Cmd) {
	ch := s.activeChannel()
	if ch == nil {
		return s, nil
	}
	body := strings.Join(args, " ")
	if body == "" {
		s.postSystem("topic: " + ch.Topic)
		return s, nil
	}
	ch.Topic = body
	s.postSystem(s.nick + " set topic: " + body)
	return s, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}
