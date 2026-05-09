package main

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type focus int

const (
	focusBranches focus = iota
	focusFeed
	focusInput
)

type mode int

const (
	modeNormal mode = iota
	modeComment
)

type model struct {
	branches []Branch

	branchIdx int
	commitIdx int

	focus focus
	mode  mode

	commitInput  textinput.Model
	commentInput textinput.Model

	width  int
	height int

	flash   string
	flashAt time.Time
}

type tickMsg time.Time

func newModel() model {
	ci := textinput.New()
	ci.Placeholder = `commit -m "your message" (Enter to push)`
	ci.Prompt = "$ "
	ci.CharLimit = 200

	cm := textinput.New()
	cm.Placeholder = "write a comment, Enter to post, Esc to cancel"
	cm.Prompt = "> "
	cm.CharLimit = 200

	return model{
		branches:     seedBranches(),
		focus:        focusBranches,
		commitInput:  ci,
		commentInput: cm,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tickEvery())
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

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		if !m.flashAt.IsZero() && time.Since(m.flashAt) > 3*time.Second {
			m.flash = ""
			m.flashAt = time.Time{}
		}
		return m, tickEvery()

	case tea.KeyMsg:
		if m.mode == modeComment {
			return m.updateCommentMode(msg)
		}
		return m.updateNormal(msg)
	}

	return m, nil
}

func (m model) updateCommentMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeNormal
		m.commentInput.Blur()
		m.commentInput.SetValue("")
		return m, nil
	case "enter":
		body := strings.TrimSpace(m.commentInput.Value())
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
	var cmd tea.Cmd
	m.commentInput, cmd = m.commentInput.Update(msg)
	return m, cmd
}

func (m model) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.focus == focusInput {
		switch msg.String() {
		case "esc":
			m.focus = focusFeed
			m.commitInput.Blur()
			return m, nil
		case "tab":
			m.focus = focusBranches
			m.commitInput.Blur()
			return m, nil
		case "enter":
			text := strings.TrimSpace(m.commitInput.Value())
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
				m.commitInput.SetValue("")
			}
			return m, nil
		}
		var cmd tea.Cmd
		m.commitInput, cmd = m.commitInput.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "tab":
		switch m.focus {
		case focusBranches:
			m.focus = focusFeed
		case focusFeed:
			m.focus = focusInput
			m.commitInput.Focus()
		}
		return m, nil

	case "shift+tab":
		switch m.focus {
		case focusFeed:
			m.focus = focusBranches
		case focusInput:
			m.focus = focusFeed
			m.commitInput.Blur()
		}
		return m, nil

	case "i":
		m.focus = focusInput
		m.commitInput.Focus()
		return m, textinput.Blink

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
			m.commentInput.Focus()
			return m, textinput.Blink
		}
		return m, nil
	}

	return m, nil
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
