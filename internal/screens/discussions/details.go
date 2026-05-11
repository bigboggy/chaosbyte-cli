package discussions

import (
	"fmt"
	"strings"
	"time"

	"github.com/bchayka/gitstatus/internal/screens"
	"github.com/bchayka/gitstatus/internal/theme"
	"github.com/bchayka/gitstatus/internal/ui"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// details mode is the modal post-view with threaded comments and reply input.

func (s *Screen) updateDetails(km tea.KeyMsg) (screens.Screen, tea.Cmd) {
	if s.commentInput.Focused() {
		return s.updateDetailsReply(km)
	}
	switch km.String() {
	case "esc":
		s.commentInput.SetValue("")
		s.mode = modeNormal
		return s, nil
	case "j", "down":
		flat := s.detailsFlat()
		if s.detailsSelIdx < len(flat)-1 {
			s.detailsSelIdx++
		}
	case "k", "up":
		if s.detailsSelIdx > -1 {
			s.detailsSelIdx--
		}
	case "g":
		s.detailsSelIdx = -1
	case "G":
		flat := s.detailsFlat()
		s.detailsSelIdx = len(flat) - 1
	case "l":
		s.detailsLikeSelected()
	case "r", "i", "enter":
		s.commentInput.Focus()
		return s, textarea.Blink
	}
	return s, nil
}

func (s *Screen) updateDetailsReply(km tea.KeyMsg) (screens.Screen, tea.Cmd) {
	switch km.String() {
	case "ctrl+enter", "alt+enter", "ctrl+j":
		s.commentInput.InsertString("\n")
		return s, nil
	case "enter", "ctrl+s", "ctrl+d":
		body := strings.TrimRight(strings.TrimSpace(s.commentInput.Value()), "\n")
		var cmd tea.Cmd
		if body != "" {
			if target := s.detailsReplyTarget(); target != nil {
				*target = append(*target, Comment{
					Author: meUser,
					Body:   body,
					At:     time.Now(),
				})
				cmd = screens.Flash("reply posted")
			}
		}
		s.commentInput.SetValue("")
		s.commentInput.Blur()
		return s, cmd
	case "esc":
		s.commentInput.Blur()
		return s, nil
	}
	var cmd tea.Cmd
	s.commentInput, cmd = s.commentInput.Update(km)
	return s, cmd
}

// detailsFlat returns the comment tree flattened for navigation.
func (s *Screen) detailsFlat() []flatComment {
	c := s.currentCommit()
	if c == nil {
		return nil
	}
	return flattenComments(c.Comments, 0)
}

// detailsReplyTarget returns the comment slice the next reply should append to:
// either the post itself (selected index -1) or the currently selected comment's
// children.
func (s *Screen) detailsReplyTarget() *[]Comment {
	c := s.currentCommit()
	if c == nil {
		return nil
	}
	if s.detailsSelIdx < 0 {
		return &c.Comments
	}
	flat := flattenComments(c.Comments, 0)
	if s.detailsSelIdx >= len(flat) {
		return &c.Comments
	}
	return &flat[s.detailsSelIdx].c.Comments
}

func (s *Screen) detailsLikeSelected() {
	c := s.currentCommit()
	if c == nil {
		return
	}
	if s.detailsSelIdx < 0 {
		toggleLike(&c.Liked, &c.Likes)
		return
	}
	flat := flattenComments(c.Comments, 0)
	if s.detailsSelIdx >= len(flat) {
		return
	}
	toggleLike(&flat[s.detailsSelIdx].c.Liked, &flat[s.detailsSelIdx].c.Likes)
}

// renderDetailsPopup draws the post + threaded comments + reply input modal.
func (s *Screen) renderDetailsPopup(width, height int) string {
	c := s.peekCommit()
	if c == nil {
		return ""
	}

	popupW := width * 80 / 100
	if popupW > 100 {
		popupW = 100
	}
	if popupW < 60 {
		popupW = 60
	}
	contentW := popupW - 6
	if contentW < 30 {
		contentW = 30
	}

	rule := lipgloss.NewStyle().Foreground(theme.BorderLo).Render(strings.Repeat("─", contentW))

	postSelected := s.detailsSelIdx == -1
	post := renderPostBlock(c, contentW, postSelected)

	flat := flattenComments(c.Comments, 0)

	target := "the post"
	if postSelected {
		target = c.Author
	} else if s.detailsSelIdx >= 0 && s.detailsSelIdx < len(flat) {
		target = flat[s.detailsSelIdx].c.Author
	}
	var replyHeader string
	if s.commentInput.Focused() {
		replyHeader = lipgloss.NewStyle().Foreground(theme.OK).Bold(true).
			Render("replying to " + target)
	} else {
		replyHeader = lipgloss.NewStyle().Foreground(theme.Muted).Italic(true).
			Render("press r to reply to " + target)
	}

	s.commentInput.SetWidth(contentW)
	s.commentInput.SetHeight(3)
	inputView := s.commentInput.View()
	var hint string
	if s.commentInput.Focused() {
		hint = theme.HelpDesc.Render("enter post   ·   ctrl+enter newline   ·   esc back to thread")
	} else {
		hint = theme.HelpDesc.Render("j/k select   ·   l like   ·   r reply   ·   esc close")
	}

	top := lipgloss.JoinVertical(lipgloss.Left, post, "", rule)
	bottom := lipgloss.JoinVertical(lipgloss.Left, rule, "", replyHeader, inputView, "", hint)

	const popupBoxOverhead = 4
	availInner := height - popupBoxOverhead
	if availInner < 10 {
		availInner = 10
	}

	topH := lipgloss.Height(top)
	bottomH := lipgloss.Height(bottom)
	middleH := availInner - topH - bottomH
	if middleH < 4 {
		middleH = 4
	}

	var blocks []string
	var startLines []int
	line := 0
	for i, f := range flat {
		selected := i == s.detailsSelIdx
		block := renderCommentBlock(*f.c, f.depth, contentW, selected)
		startLines = append(startLines, line)
		blocks = append(blocks, block)
		line += lipgloss.Height(block)
	}

	var middleStr string
	if len(flat) == 0 {
		middleStr = lipgloss.NewStyle().Foreground(theme.Muted).Italic(true).
			Render("no replies yet — be the first.")
	} else {
		middleStr = strings.Join(blocks, "\n")
	}
	middleLines := strings.Split(middleStr, "\n")

	scroll := 0
	if s.detailsSelIdx >= 0 && s.detailsSelIdx < len(blocks) {
		selStart := startLines[s.detailsSelIdx]
		selBlockH := lipgloss.Height(blocks[s.detailsSelIdx])
		selEnd := selStart + selBlockH
		if selEnd > middleH {
			scroll = selEnd - middleH
		}
		if selStart < scroll {
			scroll = selStart
		}
	}
	if scroll < 0 {
		scroll = 0
	}
	if scroll+middleH > len(middleLines) {
		scroll = len(middleLines) - middleH
	}
	if scroll < 0 {
		scroll = 0
	}

	end := scroll + middleH
	if end > len(middleLines) {
		end = len(middleLines)
	}

	var visible string
	if scroll < len(middleLines) {
		visible = strings.Join(middleLines[scroll:end], "\n")
	}
	actualH := lipgloss.Height(visible)
	if actualH < middleH {
		visible += strings.Repeat("\n", middleH-actualH)
	}

	if len(middleLines) > middleH {
		pct := 100
		if denom := len(middleLines) - middleH; denom > 0 {
			pct = scroll * 100 / denom
		}
		indicator := lipgloss.NewStyle().
			Foreground(theme.Muted).
			Width(contentW).
			Align(lipgloss.Right).
			Render(fmt.Sprintf("scroll %d%%", pct))
		top = lipgloss.JoinVertical(lipgloss.Left, post, "", rule, indicator)
		topH = lipgloss.Height(top)
		middleH = availInner - topH - bottomH
		if middleH < 4 {
			middleH = 4
		}
		if scroll+middleH > len(middleLines) {
			scroll = len(middleLines) - middleH
		}
		if scroll < 0 {
			scroll = 0
		}
		end = scroll + middleH
		if end > len(middleLines) {
			end = len(middleLines)
		}
		visible = strings.Join(middleLines[scroll:end], "\n")
		actualH = lipgloss.Height(visible)
		if actualH < middleH {
			visible += strings.Repeat("\n", middleH-actualH)
		}
	}

	_ = ui.PadToHeight // keep import alive even when not used directly

	inner := lipgloss.JoinVertical(lipgloss.Left, top, visible, bottom)
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Accent).
		Padding(1, 2).
		Render(inner)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
