package lobby

import (
	"fmt"
	"strings"

	"github.com/bchayka/gitstatus/internal/screens"
	"github.com/bchayka/gitstatus/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

type command struct {
	name string
	desc string
}

// builtins is the canonical list of slash commands. Order here is the order
// shown in autocomplete. Aliases are wired in `aliases` and don't appear here
// to keep the suggestion strip tidy.
var builtins = []command{
	{"/join", "join or switch channel"},
	{"/leave", "return to #lobby"},
	{"/list", "list channels"},
	{"/who", "list users in this channel"},
	{"/me", "third-person action"},
	{"/auth", "link a GitHub account (/auth github)"},
	{"/clear", "clear scrollback"},
	{"/help", "show all commands"},
	{"/quit", "exit vibespace"},
}

var aliases = map[string]string{
	"/exit":     "/quit",
	"/bye":      "/quit",
	"/part":     "/leave",
	"/channels": "/list",
	"/users":    "/who",
	"/?":        "/help",
}

func canonicalName(name string) string {
	if a, ok := aliases[name]; ok {
		return a
	}
	return name
}

func (s *Screen) handleSlash(text string) (*Screen, tea.Cmd) {
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return s, nil
	}
	name := canonicalName(parts[0])
	args := parts[1:]

	switch name {
	case "/help":
		return s.cmdHelp()
	case "/clear":
		return s.cmdClear()
	case "/quit":
		s.Cleanup()
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
	case "/auth":
		return s.cmdAuth(args)
	}
	s.postSystem(fmt.Sprintf("unknown command %q — try /help", parts[0]))
	return s, nil
}

// ---------------------------------------------------------------------------
// Per-command handlers
// ---------------------------------------------------------------------------

func (s *Screen) cmdHelp() (*Screen, tea.Cmd) {
	lines := []string{"available commands:"}
	for _, c := range builtins {
		lines = append(lines, fmt.Sprintf("  %-13s %s", c.name, c.desc))
	}
	lines = append(lines, "ctrl+c quits")
	s.postSystem(strings.Join(lines, "\n"))
	return s, nil
}

// cmdClear is intentionally local-only: it just hides scrollback for this
// session by jumping scroll past the end. The hub still holds the history.
func (s *Screen) cmdClear() (*Screen, tea.Cmd) {
	msgs, _ := s.hub.Messages(s.activeName)
	s.chatScroll = len(msgs)
	return s, nil
}

func (s *Screen) cmdMe(args []string) (*Screen, tea.Cmd) {
	body := strings.Join(args, " ")
	if body == "" {
		return s, nil
	}
	s.hub.Post(s.activeName, s.meUser, body, ui.ChatAction)
	s.chatScroll = 0
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
	if !s.hub.HasChannel(name) {
		s.hub.CreateChannel(name)
	}
	prev := s.activeName
	s.activeName = name
	s.hub.SetViewing(s.subID, name)
	s.chatScroll = 0
	if prev != name {
		s.hub.Post(name, s.meUser, "joined "+name, ui.ChatJoin)
	}
	return s, nil
}

func (s *Screen) cmdLeave() (*Screen, tea.Cmd) {
	if s.activeName == "#lobby" {
		s.postSystem("you're already in #lobby — try /quit to exit")
		return s, nil
	}
	leaving := s.activeName
	s.hub.Post(leaving, s.meUser, "left "+leaving, ui.ChatJoin)
	s.activeName = "#lobby"
	s.hub.SetViewing(s.subID, "#lobby")
	s.chatScroll = 0
	return s, nil
}

func (s *Screen) cmdList() (*Screen, tea.Cmd) {
	names := s.hub.ChannelNames()
	lines := []string{"channels:"}
	for _, n := range names {
		lines = append(lines, fmt.Sprintf("  %-20s  %4d online", n, s.hub.Online(n)))
	}
	s.postSystem(strings.Join(lines, "\n"))
	return s, nil
}

func (s *Screen) cmdWho() (*Screen, tea.Cmd) {
	n := s.hub.Online(s.activeName)
	s.postSystem(fmt.Sprintf("%d connected in %s", n, s.activeName))
	return s, nil
}

func (s *Screen) cmdAuth(args []string) (*Screen, tea.Cmd) {
	if len(args) == 0 {
		s.postSystem("usage: /auth github")
		return s, nil
	}
	switch strings.ToLower(args[0]) {
	case "github", "gh":
		return s.cmdAuthGithub()
	default:
		s.postSystem("unknown provider — only `github` is supported")
		return s, nil
	}
}
