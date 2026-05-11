package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ChatMessage struct {
	Author string
	Body   string
	At     time.Time
}

type Channel struct {
	Name     string
	Topic    string
	Members  int
	Online   int
	Unread   int
	Messages []ChatMessage
}

func seedChannels() []Channel {
	now := time.Now()
	h := func(d time.Duration) time.Time { return now.Add(-d) }
	return []Channel{
		{
			Name: "#general", Topic: "general dev chatter · be excellent",
			Members: 1842, Online: 312, Unread: 4,
			Messages: []ChatMessage{
				{"@yamlhater", "anyone else's CI just decided today was the day to be slow", h(38 * time.Minute)},
				{"@nullpointer", "ours is just printing fortune cookies now. devops did it.", h(36 * time.Minute)},
				{"@devops_bard", "the fortune cookies ARE the test output, you just have to interpret them", h(35 * time.Minute)},
				{"@junior_dev", "wait is that real or are we doing bits", h(32 * time.Minute)},
				{"@standup_ghost", "yes", h(30 * time.Minute)},
				{"@yamlhater", "lmao", h(29 * time.Minute)},
				{"@recovering_pm", "putting it on the roadmap", h(25 * time.Minute)},
				{"@senior_intern", "i shipped a feature today by accident. it's now load-bearing.", h(12 * time.Minute)},
				{"@ai_grifter", "every feature is an accident if you squint", h(8 * time.Minute)},
				{"@nullpointer", "philosophy hour again i see", h(4 * time.Minute)},
				{"@yamlhater", "the only philosophy is rollback", h(90 * time.Second)},
			},
		},
		{
			Name: "#vibe-coding", Topic: "for when the linter has feelings",
			Members: 904, Online: 198, Unread: 11,
			Messages: []ChatMessage{
				{"@vibe_master", "tonight's stack: bun + zod + a single prayer", h(2 * time.Hour)},
				{"@yamlhater", "you forgot the README that lies about what it does", h(95 * time.Minute)},
				{"@vibe_master", "the README is implied. the vibes ARE the README.", h(90 * time.Minute)},
				{"@ai_grifter", "i asked claude to refactor and it wrote a haiku instead. shipped.", h(70 * time.Minute)},
				{"@standup_ghost", "post the haiku", h(65 * time.Minute)},
				{"@ai_grifter", "your tests pass / the prod servers all weep / nobody knows why", h(63 * time.Minute)},
				{"@nullpointer", "okay that's actually good", h(60 * time.Minute)},
				{"@vibe_master", "deploy on a friday they said. it'll be fine they said.", h(20 * time.Minute)},
				{"@yamlhater", "what could go wrong, it's just a small change", h(18 * time.Minute)},
				{"@devops_bard", "the change was small. the blast radius was not.", h(15 * time.Minute)},
			},
		},
		{
			Name: "#rust-anonymous", Topic: "you don't have to talk about rust. but you will.",
			Members: 521, Online: 87, Unread: 0,
			Messages: []ChatMessage{
				{"@borrow_checker", "i fought the compiler. the compiler won. again.", h(4 * time.Hour)},
				{"@nullpointer", "you'll thank it in production", h(3*time.Hour + 50*time.Minute)},
				{"@borrow_checker", "i AM in production. i'm just very tired.", h(3*time.Hour + 45*time.Minute)},
				{"@yamlhater", "rewrite it in rust", h(3 * time.Hour)},
				{"@senior_intern", "rewrote it in rust. now it segfaults faster.", h(2*time.Hour + 30*time.Minute)},
			},
		},
		{
			Name: "#help", Topic: "actual help, occasionally",
			Members: 2104, Online: 411, Unread: 2,
			Messages: []ChatMessage{
				{"@junior_dev", "is it normal for `git push --force` to feel this good", h(50 * time.Minute)},
				{"@devops_bard", "yes. that's how they get you.", h(48 * time.Minute)},
				{"@yamlhater", "the rush is real and the consequences are realer", h(45 * time.Minute)},
				{"@recovering_pm", "we have an entire SOP about this. nobody reads it.", h(40 * time.Minute)},
				{"@nullpointer", "what's the SOP", h(38 * time.Minute)},
				{"@recovering_pm", "don't", h(37 * time.Minute)},
			},
		},
		{
			Name: "#side-projects", Topic: "show what you're building (or pretending to)",
			Members: 1207, Online: 256, Unread: 0,
			Messages: []ChatMessage{
				{"@vibe_master", "spent 6 hours on the landing page, 12 minutes on the product", h(3 * time.Hour)},
				{"@yamlhater", "the landing page IS the product", h(2*time.Hour + 50*time.Minute)},
				{"@junior_dev", "built a todo app. it has 14 dependencies and a CI pipeline.", h(2 * time.Hour)},
				{"@nullpointer", "perfect. you're ready for series A.", h(110 * time.Minute)},
			},
		},
		{
			Name: "#offtopic", Topic: "feelings, snacks, and bad takes",
			Members: 887, Online: 142, Unread: 1,
			Messages: []ChatMessage{
				{"@standup_ghost", "objectively the best ide is the one that doesn't crash today", h(6 * time.Hour)},
				{"@yamlhater", "that's a moving target", h(5*time.Hour + 50*time.Minute)},
				{"@devops_bard", "everything's a moving target if you don't pin it", h(5 * time.Hour)},
				{"@vibe_master", "this is the most slack-y take i've seen all week", h(4 * time.Hour)},
			},
		},
	}
}

func (m model) updateChat(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.chatActive < 0 {
		return m.updateChatLobby(msg)
	}
	if m.chatInputActive {
		return m.updateChatCompose(msg)
	}
	return m.updateChatRoom(msg)
}

func (m model) updateChatLobby(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.chatChannelIdx < len(m.channels)-1 {
			m.chatChannelIdx++
		}
	case "k", "up":
		if m.chatChannelIdx > 0 {
			m.chatChannelIdx--
		}
	case "g":
		m.chatChannelIdx = 0
	case "G":
		m.chatChannelIdx = len(m.channels) - 1
	case "enter", "o":
		if len(m.channels) > 0 {
			m.chatActive = m.chatChannelIdx
			m.channels[m.chatActive].Unread = 0
			m.chatScroll = 0
		}
	case "n":
		// create a new channel — mocked
		name := fmt.Sprintf("#new-%d", len(m.channels)+1)
		m.channels = append(m.channels, Channel{
			Name: name, Topic: "you just created this channel — claim a topic",
			Members: 1, Online: 1,
		})
		m.chatChannelIdx = len(m.channels) - 1
		m.setFlash("created " + name)
	}
	return m, nil
}

func (m model) updateChatRoom(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		m.chatScroll++
	case "k", "up":
		if m.chatScroll > 0 {
			m.chatScroll--
		}
	case "g":
		m.chatScroll = 0
	case "G":
		m.chatScroll = 9999 // clamped on render
	case "i", "c", "/":
		m.chatInputActive = true
		m.chatInput.Focus()
		return m, textarea.Blink
	}
	return m, nil
}

func (m model) updateChatCompose(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if isNewlineKey(msg.String()) {
		m.chatInput.InsertString("\n")
		return m, nil
	}
	if isSubmitKey(msg.String()) {
		body := strings.TrimRight(strings.TrimSpace(m.chatInput.Value()), "\n")
		if body != "" && m.chatActive >= 0 && m.chatActive < len(m.channels) {
			ch := &m.channels[m.chatActive]
			ch.Messages = append(ch.Messages, ChatMessage{
				Author: "@you", Body: body, At: time.Now(),
			})
			m.setFlash("sent to " + ch.Name)
		}
		m.chatInput.SetValue("")
		m.chatInput.Blur()
		m.chatInputActive = false
		return m, nil
	}
	if msg.String() == "esc" {
		m.chatInput.SetValue("")
		m.chatInput.Blur()
		m.chatInputActive = false
		return m, nil
	}
	var cmd tea.Cmd
	m.chatInput, cmd = m.chatInput.Update(msg)
	return m, cmd
}

func (m model) renderChat(width, height int) string {
	if m.chatActive < 0 {
		return m.renderChatLobby(width, height)
	}
	return m.renderChatRoom(width, height)
}

func (m model) renderChatLobby(width, height int) string {
	w := feedShellWidth(width)
	title := titleStyle.Render("chat lobby")
	subtitle := lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
		Render("pick a channel · press n to make a new one")

	var rows []string
	for i, ch := range m.channels {
		marker := "  "
		if i == m.chatChannelIdx {
			marker = "▸ "
		}
		left := fmt.Sprintf("%s%-22s", marker, truncate(ch.Name, 22))
		mid := lipgloss.NewStyle().Foreground(colorMuted).Render(truncate(ch.Topic, w-50))
		unread := ""
		if ch.Unread > 0 {
			unread = lipgloss.NewStyle().Foreground(colorLike).Bold(true).Render(fmt.Sprintf("● %d", ch.Unread))
		}
		online := lipgloss.NewStyle().Foreground(colorOk).Render(fmt.Sprintf("%d online", ch.Online))
		line := fmt.Sprintf("%s  %s  %s  %s", left, mid, online, unread)
		if i == m.chatChannelIdx {
			line = branchItemSelStyle.Width(w - 4).Render(line)
		} else {
			line = branchItemStyle.Render(line)
		}
		rows = append(rows, line)
	}
	body := strings.Join(rows, "\n")

	stacked := lipgloss.JoinVertical(lipgloss.Left,
		title, subtitle, "", body,
	)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Top, stacked)
}

func (m model) renderChatRoom(width, height int) string {
	w := feedShellWidth(width)
	ch := m.channels[m.chatActive]

	header := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(ch.Name),
		lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render(ch.Topic),
		lipgloss.NewStyle().Foreground(colorOk).Render(fmt.Sprintf("%d online · %d members", ch.Online, ch.Members)),
	)

	inputH := 5
	if !m.chatInputActive {
		inputH = 3
	}
	bodyH := height - lipgloss.Height(header) - inputH - 2
	if bodyH < 4 {
		bodyH = 4
	}

	// render messages, newest at the bottom
	contentW := w - 4
	var lines []string
	for _, msg := range ch.Messages {
		lines = append(lines, renderChatLine(msg, contentW)...)
	}

	// auto-scroll to bottom unless user scrolled
	maxScroll := len(lines) - bodyH
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.chatScroll > maxScroll {
		m.chatScroll = maxScroll
	}
	end := len(lines) - m.chatScroll
	start := end - bodyH
	if start < 0 {
		start = 0
	}
	if end > len(lines) {
		end = len(lines)
	}
	visible := strings.Join(lines[start:end], "\n")
	visible = padToHeight(visible, bodyH)

	var input string
	if m.chatInputActive {
		m.chatInput.SetWidth(contentW)
		m.chatInput.SetHeight(3)
		input = m.chatInput.View()
	} else {
		input = lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
			Render("press i to type · j/k scroll · G newest")
	}

	stacked := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		visible,
		dividerLine(w),
		input,
	)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Top, stacked)
}

func renderChatLine(msg ChatMessage, width int) []string {
	author := commentAuthorStyle.Render(msg.Author)
	ts := commitTimeStyle.Render(humanizeTime(msg.At))
	prefix := fmt.Sprintf("%s  %s  ", author, ts)
	bodyW := width - 4
	wrapped := wrap(msg.Body, bodyW)
	parts := strings.Split(wrapped, "\n")
	var out []string
	out = append(out, prefix+commentBodyStyle.Render(parts[0]))
	for _, p := range parts[1:] {
		out = append(out, "    "+commentBodyStyle.Render(p))
	}
	return out
}
