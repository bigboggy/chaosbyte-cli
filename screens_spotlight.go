package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Spotlight struct {
	Project     string
	Author      string
	RepoURL     string
	Description string
	Stars       int
	Language    string
	Highlights  []string
}

const spotlightRotateSecs = 300

func seedSpotlights() []Spotlight {
	return []Spotlight{
		{
			Project:     "lazygit",
			Author:      "@jesseduffield",
			RepoURL:     "https://github.com/jesseduffield/lazygit",
			Description: "simple terminal UI for git commands. the thing your senior engineer secretly uses.",
			Stars:       54820,
			Language:    "Go",
			Highlights: []string{
				"fully keyboard driven git, no mouse no problem",
				"stage, commit, push, pull, rebase, cherry-pick from one screen",
				"the answer to 'why am i typing all this git out by hand'",
			},
		},
		{
			Project:     "atuin",
			Author:      "@ellie",
			RepoURL:     "https://github.com/atuinsh/atuin",
			Description: "magical shell history. you will weep when you realize what you've been missing.",
			Stars:       21102,
			Language:    "Rust",
			Highlights: []string{
				"searchable, syncable, encrypted shell history across machines",
				"fzf-style fuzzy search over every command you've ever run",
				"finally, a reason to be proud of your bash history",
			},
		},
		{
			Project:     "zellij",
			Author:      "@aram",
			RepoURL:     "https://github.com/zellij-org/zellij",
			Description: "a terminal workspace with batteries included. tmux had a kid, the kid is opinionated.",
			Stars:       24410,
			Language:    "Rust",
			Highlights: []string{
				"layouts, panes, tabs, plugins — all configurable in kdl",
				"the UI tells you the keybinds, so you won't print a cheat sheet",
				"floating windows in your terminal. yes really.",
			},
		},
		{
			Project:     "helix",
			Author:      "@helix-editor",
			RepoURL:     "https://github.com/helix-editor/helix",
			Description: "post-modern modal text editor. it's vim if vim had therapy.",
			Stars:       38201,
			Language:    "Rust",
			Highlights: []string{
				"selection → action grammar, opposite of vim's verb → motion",
				"multi-cursor first class, no plugin gymnastics",
				"LSP and tree-sitter built in, you do nothing to get smart features",
			},
		},
		{
			Project:     "uv",
			Author:      "@astral-sh",
			RepoURL:     "https://github.com/astral-sh/uv",
			Description: "the python package manager you wanted in 2014, finally arrived in 2025.",
			Stars:       34102,
			Language:    "Rust",
			Highlights: []string{
				"10-100x faster than pip, written in rust because of course",
				"replaces pip, pip-tools, pipx, poetry, pyenv, virtualenv — in one tool",
				"your python environment is now actually reproducible, somehow",
			},
		},
	}
}

func seedSpotlightChat() []ChatMessage {
	now := time.Now()
	h := func(d time.Duration) time.Time { return now.Add(-d) }
	return []ChatMessage{
		{"@yamlhater", "wait this is what i've been needing the whole time", h(4 * time.Minute)},
		{"@nullpointer", "the demo gif alone sold me, send help i'm installing it now", h(3 * time.Minute)},
		{"@vibe_master", "imagine using this for one (1) week and writing a medium post about it", h(150 * time.Second)},
		{"@devops_bard", "i tried this, then i tried to uninstall it. could not.", h(70 * time.Second)},
		{"@junior_dev", "is this the one where you press a key and it just works? or is this the OTHER one", h(40 * time.Second)},
		{"@standup_ghost", "they're all the one where you press a key and it just works", h(20 * time.Second)},
	}
}

// spotlightRotation returns the current spotlight index and seconds remaining
// until the next rotation. It is a pure function of the current time so it
// stays consistent across renders.
func (m model) spotlightRotation() (int, int) {
	if len(m.spotlights) == 0 {
		return 0, spotlightRotateSecs
	}
	t := m.now
	if t.IsZero() {
		t = time.Now()
	}
	secs := t.Unix()
	idx := int((secs / spotlightRotateSecs) % int64(len(m.spotlights)))
	remaining := spotlightRotateSecs - int(secs%spotlightRotateSecs)
	return idx, remaining
}

func mmss(secs int) string {
	if secs < 0 {
		secs = 0
	}
	return fmt.Sprintf("%d:%02d", secs/60, secs%60)
}

func (m model) updateSpotlight(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.spotlightInputActive {
		return m.updateSpotlightCompose(msg)
	}
	switch msg.String() {
	case "j", "down":
		m.spotlightChatScroll++
	case "k", "up":
		if m.spotlightChatScroll > 0 {
			m.spotlightChatScroll--
		}
	case "g":
		m.spotlightChatScroll = 0
	case "G":
		m.spotlightChatScroll = 9999
	case "i", "c":
		m.spotlightInputActive = true
		m.spotlightInput.Focus()
		return m, textarea.Blink
	case "o", "enter":
		idx, _ := m.spotlightRotation()
		if idx < len(m.spotlights) {
			m.setFlash("opening: " + m.spotlights[idx].RepoURL)
		}
	}
	return m, nil
}

func (m model) updateSpotlightCompose(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if isNewlineKey(msg.String()) {
		m.spotlightInput.InsertString("\n")
		return m, nil
	}
	if isSubmitKey(msg.String()) {
		body := strings.TrimRight(strings.TrimSpace(m.spotlightInput.Value()), "\n")
		if body != "" {
			m.spotlightChat = append(m.spotlightChat, ChatMessage{
				Author: "@you", Body: body, At: time.Now(),
			})
			m.setFlash("posted to spotlight chat")
		}
		m.spotlightInput.SetValue("")
		m.spotlightInput.Blur()
		m.spotlightInputActive = false
		return m, nil
	}
	if msg.String() == "esc" {
		m.spotlightInput.SetValue("")
		m.spotlightInput.Blur()
		m.spotlightInputActive = false
		return m, nil
	}
	var cmd tea.Cmd
	m.spotlightInput, cmd = m.spotlightInput.Update(msg)
	return m, cmd
}

func (m model) renderSpotlight(width, height int) string {
	w := feedShellWidth(width)
	contentW := w - 2

	idx, secs := m.spotlightRotation()
	if idx >= len(m.spotlights) {
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center,
			statusStyle.Render("no spotlight scheduled"))
	}
	sp := m.spotlights[idx]

	title := titleStyle.Render("spotlight · " + sp.Project)
	rotateNote := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).
		Render(fmt.Sprintf("next rotation in %s · %d/%d", mmss(secs), idx+1, len(m.spotlights)))

	card := renderSpotlightCard(sp, contentW)
	cardH := lipgloss.Height(card)

	inputH := 5
	if !m.spotlightInputActive {
		inputH = 3
	}
	chatH := height - cardH - inputH - 6
	if chatH < 4 {
		chatH = 4
	}

	chat := m.renderSpotlightChat(contentW, chatH)

	var input string
	if m.spotlightInputActive {
		m.spotlightInput.SetWidth(contentW - 2)
		m.spotlightInput.SetHeight(3)
		input = m.spotlightInput.View()
	} else {
		input = lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
			Render("press i to join the discussion · j/k scroll · o open repo")
	}

	stacked := lipgloss.JoinVertical(lipgloss.Left,
		title, rotateNote, "", card,
		dividerLine(contentW),
		chat,
		dividerLine(contentW),
		input,
	)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Top, stacked)
}

func renderSpotlightCard(sp Spotlight, width int) string {
	innerW := width - 4

	name := lipgloss.NewStyle().Foreground(colorAccent2).Bold(true).
		Render(sp.Project)
	lang := lipgloss.NewStyle().Foreground(colorAccent).Render(fmt.Sprintf("[%s]", sp.Language))
	author := lipgloss.NewStyle().Foreground(colorOk).Render(sp.Author)
	stars := lipgloss.NewStyle().Foreground(colorWarn).Render(fmt.Sprintf("★ %d", sp.Stars))

	header := fmt.Sprintf("%s  %s  %s  %s", name, lang, stars, author)

	desc := lipgloss.NewStyle().Foreground(colorFg).Render(wrap(sp.Description, innerW))
	url := lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render(sp.RepoURL)

	var highlights []string
	for _, h := range sp.Highlights {
		highlights = append(highlights,
			lipgloss.NewStyle().Foreground(colorAccent).Render("  ▸ ")+
				lipgloss.NewStyle().Foreground(colorFg).Render(h))
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		header, "", desc, "", strings.Join(highlights, "\n"), "", url,
	)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccent2).
		Padding(0, 2).
		Width(innerW).
		Render(content)
}

func (m model) renderSpotlightChat(width, height int) string {
	title := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("live discussion")
	var lines []string
	for _, msg := range m.spotlightChat {
		lines = append(lines, renderChatLine(msg, width)...)
	}
	maxScroll := len(lines) - (height - 1)
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.spotlightChatScroll > maxScroll {
		m.spotlightChatScroll = maxScroll
	}
	end := len(lines) - m.spotlightChatScroll
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
	visible = padToHeight(visible, height-1)
	return lipgloss.JoinVertical(lipgloss.Left, title, visible)
}
