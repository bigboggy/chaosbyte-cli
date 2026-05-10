package main

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

type focus int

const (
	focusBranches focus = iota
	focusFeed
)

type mode int

const (
	modeNormal mode = iota
	modeCompose
	modeComment
)

type model struct {
	branches []Branch

	branchIdx int
	commitIdx int

	focus focus
	mode  mode

	commitInput  textarea.Model
	commentInput textarea.Model

	width  int
	height int

	flash   string
	flashAt time.Time
}

type tickMsg time.Time

func newModel() model {
	ci := textarea.New()
	ci.Placeholder = `what did you ship?  (Ctrl+S to push, Enter for newline, Esc to cancel)`
	ci.Prompt = ""
	ci.ShowLineNumbers = false
	ci.CharLimit = 0
	ci.SetHeight(8)

	cm := textarea.New()
	cm.Placeholder = "your reply..."
	cm.Prompt = ""
	cm.ShowLineNumbers = false
	cm.CharLimit = 0
	cm.SetHeight(6)

	return model{
		branches:     seedBranches(),
		focus:        focusBranches,
		commitInput:  ci,
		commentInput: cm,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, tickEvery())
}

func tickEvery() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m *model) currentBranch() *Branch {
	if len(m.branches) == 0 {
		return nil
	}
	return &m.branches[m.branchIdx]
}

func (m *model) currentCommit() *Commit {
	b := m.currentBranch()
	if b == nil || len(b.Commits) == 0 {
		return nil
	}
	if m.commitIdx >= len(b.Commits) {
		m.commitIdx = len(b.Commits) - 1
	}
	return &b.Commits[m.commitIdx]
}

func (m *model) setFlash(s string) {
	m.flash = s
	m.flashAt = time.Now()
}

func (m model) popupTextareaSize() (w, h int) {
	pw, _ := popupSize(m.width, m.height)
	return pw - 4, 8
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		taW, _ := m.popupTextareaSize()
		m.commitInput.SetWidth(taW)
		m.commentInput.SetWidth(taW)
		return m, nil

	case tickMsg:
		if !m.flashAt.IsZero() && time.Since(m.flashAt) > 3*time.Second {
			m.flash = ""
			m.flashAt = time.Time{}
		}
		return m, tickEvery()

	case tea.KeyMsg:
		switch m.mode {
		case modeCompose:
			return m.updateCompose(msg)
		case modeComment:
			return m.updateCommentMode(msg)
		}
		return m.updateNormal(msg)
	}

	return m, nil
}

func (m model) updateCompose(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if isSubmitKey(msg.String()) {
		text := strings.TrimRight(strings.TrimSpace(m.commitInput.Value()), "\n")
		if text != "" {
			b := m.currentBranch()
			if b != nil {
				b.Commits = append([]Commit{{
					SHA:     fakeSHA(),
					Author:  "you",
					Message: text,
					At:      time.Now(),
				}}, b.Commits...)
				m.commitIdx = 0
				m.setFlash("pushed to " + b.Name)
			}
		}
		m.commitInput.SetValue("")
		m.commitInput.Blur()
		m.mode = modeNormal
		return m, nil
	}
	if msg.String() == "esc" {
		m.commitInput.SetValue("")
		m.commitInput.Blur()
		m.mode = modeNormal
		return m, nil
	}
	var cmd tea.Cmd
	m.commitInput, cmd = m.commitInput.Update(msg)
	return m, cmd
}

func (m model) updateCommentMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if isSubmitKey(msg.String()) {
		body := strings.TrimRight(strings.TrimSpace(m.commentInput.Value()), "\n")
		if body != "" {
			c := m.currentCommit()
			if c != nil {
				c.Comments = append(c.Comments, Comment{
					Author: "you",
					Body:   body,
					At:     time.Now(),
				})
				m.setFlash("comment posted")
			}
		}
		m.commentInput.SetValue("")
		m.commentInput.Blur()
		m.mode = modeNormal
		return m, nil
	}
	if msg.String() == "esc" {
		m.commentInput.SetValue("")
		m.commentInput.Blur()
		m.mode = modeNormal
		return m, nil
	}
	var cmd tea.Cmd
	m.commentInput, cmd = m.commentInput.Update(msg)
	return m, cmd
}

func (m model) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "tab":
		if m.focus == focusBranches {
			m.focus = focusFeed
		} else {
			m.focus = focusBranches
		}
		return m, nil

	case "shift+tab":
		if m.focus == focusFeed {
			m.focus = focusBranches
		} else {
			m.focus = focusFeed
		}
		return m, nil

	case "n", "i":
		m.mode = modeCompose
		m.commitInput.SetValue("")
		m.commitInput.Focus()
		return m, textarea.Blink

	case "j", "down":
		if m.focus == focusBranches {
			if m.branchIdx < len(m.branches)-1 {
				m.branchIdx++
				m.commitIdx = 0
			}
		} else if m.focus == focusFeed {
			b := m.currentBranch()
			if b != nil && m.commitIdx < len(b.Commits)-1 {
				m.commitIdx++
			}
		}
		return m, nil

	case "k", "up":
		if m.focus == focusBranches {
			if m.branchIdx > 0 {
				m.branchIdx--
				m.commitIdx = 0
			}
		} else if m.focus == focusFeed {
			if m.commitIdx > 0 {
				m.commitIdx--
			}
		}
		return m, nil

	case "l":
		if m.focus == focusFeed {
			c := m.currentCommit()
			if c != nil {
				if c.Liked {
					c.Liked = false
					c.Likes--
					m.setFlash("unliked")
				} else {
					c.Liked = true
					c.Likes++
					m.setFlash("liked")
				}
			}
		}
		return m, nil

	case "c":
		if m.focus == focusFeed && m.currentCommit() != nil {
			m.mode = modeComment
			m.commentInput.SetValue("")
			m.commentInput.Focus()
			return m, textarea.Blink
		}
		return m, nil
	}

	return m, nil
}

func isSubmitKey(s string) bool {
	switch s {
	case "ctrl+s", "ctrl+d", "ctrl+enter", "alt+enter":
		return true
	}
	return false
}

func fakeSHA() string {
	const hex = "0123456789abcdef"
	now := time.Now().UnixNano()
	out := make([]byte, 7)
	for i := range out {
		out[i] = hex[now&0xf]
		now >>= 4
	}
	return string(out)
}
