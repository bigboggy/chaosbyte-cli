// Package discussions is the commit feed screen — a Reddit-style threaded
// view scoped per branch. It's the original feature; everything else was
// added around it.
//
// Files:
//   - discussions.go — Screen, state machine, top-level update/view
//   - seed.go        — Branch/Commit/Comment types + fake data
//   - render.go      — feed/tab/commit rendering
//   - compose.go     — modeCompose handlers + popup
//   - details.go     — modeDetails handlers + popup
//   - branches.go    — modeBranchPicker handlers + popup
//   - comments.go    — comment tree flattening + like toggling
package discussions

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

// mode is the discussions screen's internal state machine. Modal popups
// (compose, details, branch picker) suspend normal navigation.
type mode int

const (
	modeNormal mode = iota
	modeCompose
	modeDetails
	modeBranchPicker
)

const meUser = "@boggy"

// Screen owns the branch list, current selection, mode, and the two textareas
// for composing commits and replies.
type Screen struct {
	branches []Branch

	branchIdx int
	commitIdx int

	mode mode

	commitInput  textarea.Model
	commentInput textarea.Model

	detailsSelIdx   int
	branchPickerIdx int

	width, height int

	backdrop *field.Backdrop
}

func New() *Screen {
	ci := textarea.New()
	ci.Placeholder = `what did you ship?  (Enter to push, Ctrl+Enter for newline, Esc to cancel)`
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

	return &Screen{
		branches:     seedBranches(),
		commitInput:  ci,
		commentInput: cm,
		backdrop:     field.NewBackdrop(),
	}
}

func (s *Screen) Init() tea.Cmd { return tea.Batch(textarea.Blink, field.TickCmd()) }

// OnEnter is the router's field-driven entry hook.
func (s *Screen) OnEnter() { s.backdrop.Pulse(0.7) }

func (s *Screen) Name() string  { return screens.DiscussionsID }
func (s *Screen) Title() string { return "discussions" }

func (s *Screen) HeaderContext() string {
	b := s.currentBranch()
	if b == nil {
		return ""
	}
	totalCommits, totalLikes, totalComments := repoTotals(s.branches)
	sep := lipgloss.NewStyle().Foreground(theme.Muted).Render(" · ")
	return lipgloss.NewStyle().Foreground(theme.OK).Render("on "+b.Name) + sep +
		lipgloss.NewStyle().Foreground(theme.Muted).Render(fmt.Sprintf("%d commits", totalCommits)) + sep +
		theme.LikeIcon.Render(fmt.Sprintf("♥ %d", totalLikes)) + sep +
		theme.CommentCount.Render(fmt.Sprintf("💬 %d", totalComments))
}

func (s *Screen) Footer() []screens.KeyHint {
	switch s.mode {
	case modeCompose:
		return []screens.KeyHint{
			{Key: "enter", Desc: "push"}, {Key: "ctrl+enter", Desc: "newline"}, {Key: "esc", Desc: "cancel"},
		}
	case modeDetails:
		if s.commentInput.Focused() {
			return []screens.KeyHint{
				{Key: "enter", Desc: "post"}, {Key: "ctrl+enter", Desc: "newline"}, {Key: "esc", Desc: "back to thread"},
			}
		}
		return []screens.KeyHint{
			{Key: "j/k", Desc: "select"}, {Key: "l", Desc: "like"}, {Key: "r", Desc: "reply"}, {Key: "esc", Desc: "close"},
		}
	case modeBranchPicker:
		return []screens.KeyHint{
			{Key: "j/k", Desc: "move"}, {Key: "enter", Desc: "checkout"}, {Key: "esc", Desc: "cancel"},
		}
	}
	return []screens.KeyHint{
		{Key: "n", Desc: "new"}, {Key: "enter", Desc: "open"}, {Key: "l", Desc: "like"}, {Key: "j/k", Desc: "move"},
		{Key: "tab", Desc: "branch"}, {Key: "b", Desc: "branches"}, {Key: "esc", Desc: "lobby"},
	}
}

func (s *Screen) InputFocused() bool {
	return s.mode == modeCompose || (s.mode == modeDetails && s.commentInput.Focused())
}

// BackOut lets the app router pop modes one at a time on esc, falling through
// to the lobby only when the screen is in normal mode. Returns true if the
// screen handled the back action internally.
func (s *Screen) BackOut() bool {
	if s.mode != modeNormal {
		s.mode = modeNormal
		s.commitInput.Blur()
		s.commentInput.Blur()
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (s *Screen) Update(msg tea.Msg) (screens.Screen, tea.Cmd) {
	switch m := msg.(type) {
	case field.TickMsg:
		s.backdrop.Tick(time.Time(m))
		return s, field.TickCmd()
	case tea.MouseMsg:
		s.backdrop.SetCursor(float64(m.X), float64(m.Y))
		return s, nil
	}
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}
	s.backdrop.Pulse(0.04)
	switch s.mode {
	case modeCompose:
		return s.updateCompose(km)
	case modeDetails:
		return s.updateDetails(km)
	case modeBranchPicker:
		return s.updateBranchPicker(km)
	}
	return s.updateNormal(km)
}

func (s *Screen) updateNormal(km tea.KeyMsg) (screens.Screen, tea.Cmd) {
	switch km.String() {
	case "tab":
		s.cycleBranch(+1)
	case "shift+tab":
		s.cycleBranch(-1)
	case "b":
		s.mode = modeBranchPicker
		s.branchPickerIdx = s.branchIdx
	case "n", "i":
		return s, s.enterCompose()
	case "j", "down":
		if b := s.currentBranch(); b != nil && s.commitIdx < len(b.Commits)-1 {
			s.commitIdx++
		}
	case "k", "up":
		if s.commitIdx > 0 {
			s.commitIdx--
		}
	case "l":
		c := s.currentCommit()
		if c == nil {
			return s, nil
		}
		toggleLike(&c.Liked, &c.Likes)
		if c.Liked {
			return s, screens.Flash("liked")
		}
		return s, screens.Flash("unliked")
	case "enter", "o":
		if s.currentCommit() != nil {
			s.mode = modeDetails
			s.detailsSelIdx = -1
			s.commentInput.SetValue("")
			s.commentInput.Blur()
		}
	}
	return s, nil
}

func (s *Screen) cycleBranch(delta int) {
	visible := s.visibleTabBranches()
	if len(visible) == 0 {
		return
	}
	pos := -1
	for i, idx := range visible {
		if idx == s.branchIdx {
			pos = i
			break
		}
	}
	pos = (pos + delta + len(visible)) % len(visible)
	s.branchIdx = visible[pos]
	s.commitIdx = 0
}

// ---------------------------------------------------------------------------
// Accessors
// ---------------------------------------------------------------------------

func (s *Screen) currentBranch() *Branch {
	if len(s.branches) == 0 {
		return nil
	}
	return &s.branches[s.branchIdx]
}

func (s *Screen) currentCommit() *Commit {
	b := s.currentBranch()
	if b == nil || len(b.Commits) == 0 {
		return nil
	}
	if s.commitIdx >= len(b.Commits) {
		s.commitIdx = len(b.Commits) - 1
	}
	return &b.Commits[s.commitIdx]
}

func (s *Screen) peekCommit() *Commit { return s.currentCommit() }

func repoTotals(branches []Branch) (commits, likes, comments int) {
	for _, b := range branches {
		commits += len(b.Commits)
		for _, c := range b.Commits {
			likes += c.Likes
			comments += len(c.Comments)
		}
	}
	return
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (s *Screen) View(width, height int) string {
	switch s.mode {
	case modeCompose:
		return s.renderComposePopup(width, height)
	case modeDetails:
		return s.renderDetailsPopup(width, height)
	case modeBranchPicker:
		return s.renderBranchPicker(width, height)
	}
	return s.renderMain(width, height)
}

func (s *Screen) renderMain(termW, bodyH int) string {
	shellW := ui.FeedShellWidth(termW)
	contentW := shellW - 2

	tabs := s.renderTabs(contentW)
	tabsH := lipgloss.Height(tabs)

	feedH := bodyH - tabsH - 1
	if feedH < 4 {
		feedH = 4
	}
	feed := s.renderFeed(contentW, feedH)
	feedRows := strings.Split(feed, "\n")
	if len(feedRows) < feedH {
		pad := make([]string, feedH-len(feedRows))
		feedRows = append(feedRows, pad...)
	}
	fieldRows := strings.Split(s.backdrop.Render(contentW, feedH), "\n")
	composed := field.Composite(feedRows, fieldRows, feedH)

	col := lipgloss.JoinVertical(lipgloss.Left, tabs, "", composed)
	return lipgloss.Place(termW, bodyH, lipgloss.Left, lipgloss.Top, col)
}
