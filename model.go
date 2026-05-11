package main

import (
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type screen int

const (
	screenIntro screen = iota
	screenLobby
	screenNews
	screenResources
	screenSpotlight
	screenGames
	screenDiscussions
)

func (s screen) name() string {
	switch s {
	case screenIntro:
		return "intro"
	case screenLobby:
		return "lobby"
	case screenNews:
		return "news"
	case screenResources:
		return "resources"
	case screenSpotlight:
		return "spotlight"
	case screenGames:
		return "games"
	case screenDiscussions:
		return "discussions"
	}
	return ""
}

type mode int

const (
	modeNormal mode = iota
	modeCompose
	modeDetails
	modeBranchPicker
)

const visibleTabs = 3

type model struct {
	screen screen

	// intro
	introStart time.Time

	// lobby / chat
	channels       []Channel
	chatActive     int
	chatScroll     int
	lobbyInput     textinput.Model
	lobbyHistory   []string
	historyIdx     int
	joinPosted     bool
	completionStem string
	completionIdx  int

	// news
	newsItems  []NewsItem
	newsIdx    int
	newsScroll int

	// resources
	resourcesTab         int
	resourcesIdx         int
	resourcesQuery       string
	resourcesQueryActive bool
	skillsTrending       []Skill
	skillsTop            []Skill
	repos                []Repo

	// spotlight
	spotlights           []Spotlight
	spotlightChat        []ChatMessage
	spotlightChatScroll  int
	spotlightInput       textarea.Model
	spotlightInputActive bool

	// games
	games     []Game
	gameIdx   int
	gameState gameState
	bugHunter bugHunterState

	// discussions
	branches        []Branch
	branchIdx       int
	commitIdx       int
	mode            mode
	commitInput     textarea.Model
	commentInput    textarea.Model
	detailsSelIdx   int
	branchPickerIdx int

	// shared
	width  int
	height int

	flash   string
	flashAt time.Time

	now time.Time
}

type tickMsg time.Time

func newModel() model {
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

	spot := textarea.New()
	spot.Placeholder = "join the discussion... (enter to send)"
	spot.Prompt = ""
	spot.ShowLineNumbers = false
	spot.CharLimit = 0
	spot.SetHeight(3)

	return model{
		screen:         screenIntro,
		introStart:     time.Now(),
		branches:       seedBranches(),
		channels:       seedChannels(),
		chatActive:     0,
		newsItems:      seedNews(),
		skillsTrending: seedTrendingSkills(),
		skillsTop:      seedTopSkills(),
		repos:          seedRepos(),
		spotlights:     seedSpotlights(),
		spotlightChat:  seedSpotlightChat(),
		games:          seedGames(),
		bugHunter:      newBugHunter(),
		commitInput:    ci,
		commentInput:   cm,
		spotlightInput: spot,
		lobbyInput:     newLobbyInput(),
		now:            time.Now(),
	}
}

func (m *model) visibleTabBranches() []int {
	n := len(m.branches)
	if n == 0 {
		return nil
	}
	limit := visibleTabs
	if n < limit {
		limit = n
	}
	idxs := make([]int, 0, limit)
	for i := 0; i < limit; i++ {
		idxs = append(idxs, i)
	}
	hasActive := false
	for _, i := range idxs {
		if i == m.branchIdx {
			hasActive = true
			break
		}
	}
	if !hasActive {
		idxs[len(idxs)-1] = m.branchIdx
		sort.Ints(idxs)
	}
	return idxs
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, textinput.Blink, tickEvery(), introTickCmd())
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
		m.spotlightInput.SetWidth(taW)
		return m, nil

	case tickMsg:
		m.now = time.Time(msg)
		if !m.flashAt.IsZero() && time.Since(m.flashAt) > 3*time.Second {
			m.flash = ""
			m.flashAt = time.Time{}
		}
		return m, tickEvery()

	case introTickMsg:
		if m.screen != screenIntro {
			return m, nil
		}
		if time.Since(m.introStart).Milliseconds() >= introFadeEnd {
			return m.finishIntro(), nil
		}
		return m, introTickCmd()

	case tea.KeyMsg:
		return m.routeKey(msg)
	}

	return m, nil
}

func (m model) routeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.screen == screenIntro {
		return m.updateIntro(msg)
	}
	if m.screen == screenLobby {
		return m.updateLobby(msg)
	}
	if m.inputFocused() {
		return m.updateScreen(msg)
	}

	// Global keys for non-lobby screens (when nothing else is grabbing input).
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		mm, handled, cmd := m.screenInterceptEsc()
		if handled {
			return mm, cmd
		}
		return m.toLobby(), nil
	case "q":
		return m.toLobby(), nil
	}
	return m.updateScreen(msg)
}

func (m model) inputFocused() bool {
	switch m.screen {
	case screenLobby:
		return true
	case screenDiscussions:
		return m.mode == modeCompose || (m.mode == modeDetails && m.commentInput.Focused())
	case screenSpotlight:
		return m.spotlightInputActive
	case screenResources:
		return m.resourcesQueryActive
	}
	return false
}

func (m model) toLobby() tea.Model {
	m.screen = screenLobby
	m.mode = modeNormal
	m.commitInput.Blur()
	m.commentInput.Blur()
	m.spotlightInput.Blur()
	m.spotlightInputActive = false
	m.resourcesQueryActive = false
	m.lobbyInput.Focus()
	return m
}

func (m model) screenInterceptEsc() (tea.Model, bool, tea.Cmd) {
	switch m.screen {
	case screenDiscussions:
		if m.mode != modeNormal {
			m.mode = modeNormal
			m.commitInput.Blur()
			m.commentInput.Blur()
			return m, true, nil
		}
	case screenGames:
		if m.gameState != gameStateList {
			m.gameState = gameStateList
			return m, true, nil
		}
	}
	return m, false, nil
}

func (m model) updateScreen(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenLobby:
		return m.updateLobby(msg)
	case screenNews:
		return m.updateNews(msg)
	case screenResources:
		return m.updateResources(msg)
	case screenSpotlight:
		return m.updateSpotlight(msg)
	case screenGames:
		return m.updateGames(msg)
	case screenDiscussions:
		return m.updateDiscussions(msg)
	}
	return m, nil
}

// ============================================================================
// Discussions screen (existing functionality)
// ============================================================================

func (m model) updateDiscussions(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeCompose:
		return m.updateCompose(msg)
	case modeDetails:
		return m.updateDetails(msg)
	case modeBranchPicker:
		return m.updateBranchPicker(msg)
	}
	return m.updateDiscussionsNormal(msg)
}

func (m model) updateCompose(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if isNewlineKey(msg.String()) {
		m.commitInput.InsertString("\n")
		return m, nil
	}
	if isSubmitKey(msg.String()) {
		text := strings.TrimRight(strings.TrimSpace(m.commitInput.Value()), "\n")
		if text != "" {
			b := m.currentBranch()
			if b != nil {
				b.Commits = append([]Commit{{
					SHA:     fakeSHA(),
					Author:  meUser,
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

func (m model) updateDetails(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.commentInput.Focused() {
		if isNewlineKey(msg.String()) {
			m.commentInput.InsertString("\n")
			return m, nil
		}
		if isSubmitKey(msg.String()) {
			body := strings.TrimRight(strings.TrimSpace(m.commentInput.Value()), "\n")
			if body != "" {
				target := m.detailsReplyTarget()
				if target != nil {
					*target = append(*target, Comment{
						Author: meUser,
						Body:   body,
						At:     time.Now(),
					})
					m.setFlash("reply posted")
				}
			}
			m.commentInput.SetValue("")
			m.commentInput.Blur()
			return m, nil
		}
		if msg.String() == "esc" {
			m.commentInput.Blur()
			return m, nil
		}
		var cmd tea.Cmd
		m.commentInput, cmd = m.commentInput.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "esc":
		m.commentInput.SetValue("")
		m.mode = modeNormal
		return m, nil
	case "j", "down":
		flat := m.detailsFlat()
		if m.detailsSelIdx < len(flat)-1 {
			m.detailsSelIdx++
		}
		return m, nil
	case "k", "up":
		if m.detailsSelIdx > -1 {
			m.detailsSelIdx--
		}
		return m, nil
	case "g":
		m.detailsSelIdx = -1
		return m, nil
	case "G":
		flat := m.detailsFlat()
		m.detailsSelIdx = len(flat) - 1
		return m, nil
	case "l":
		m.detailsLikeSelected()
		return m, nil
	case "r", "i", "enter":
		m.commentInput.Focus()
		return m, textarea.Blink
	}
	return m, nil
}

func (m model) updateDiscussionsNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		visible := m.visibleTabBranches()
		if len(visible) > 0 {
			pos := -1
			for i, idx := range visible {
				if idx == m.branchIdx {
					pos = i
					break
				}
			}
			next := (pos + 1) % len(visible)
			m.branchIdx = visible[next]
			m.commitIdx = 0
		}
		return m, nil

	case "shift+tab":
		visible := m.visibleTabBranches()
		if len(visible) > 0 {
			pos := -1
			for i, idx := range visible {
				if idx == m.branchIdx {
					pos = i
					break
				}
			}
			prev := pos - 1
			if prev < 0 {
				prev = len(visible) - 1
			}
			m.branchIdx = visible[prev]
			m.commitIdx = 0
		}
		return m, nil

	case "b":
		m.mode = modeBranchPicker
		m.branchPickerIdx = m.branchIdx
		return m, nil

	case "n", "i":
		m.mode = modeCompose
		m.commitInput.SetValue("")
		m.commitInput.Focus()
		return m, textarea.Blink

	case "j", "down":
		b := m.currentBranch()
		if b != nil && m.commitIdx < len(b.Commits)-1 {
			m.commitIdx++
		}
		return m, nil

	case "k", "up":
		if m.commitIdx > 0 {
			m.commitIdx--
		}
		return m, nil

	case "l":
		c := m.currentCommit()
		if c != nil {
			toggleLike(&c.Liked, &c.Likes)
			if c.Liked {
				m.setFlash("liked")
			} else {
				m.setFlash("unliked")
			}
		}
		return m, nil

	case "enter", "o":
		if m.currentCommit() != nil {
			m.mode = modeDetails
			m.detailsSelIdx = -1
			m.commentInput.SetValue("")
			m.commentInput.Blur()
			return m, nil
		}
		return m, nil
	}

	return m, nil
}

func (m model) updateBranchPicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "b":
		m.mode = modeNormal
		return m, nil
	case "j", "down":
		if m.branchPickerIdx < len(m.branches)-1 {
			m.branchPickerIdx++
		}
		return m, nil
	case "k", "up":
		if m.branchPickerIdx > 0 {
			m.branchPickerIdx--
		}
		return m, nil
	case "g":
		m.branchPickerIdx = 0
		return m, nil
	case "G":
		m.branchPickerIdx = len(m.branches) - 1
		return m, nil
	case "enter":
		if m.branchPickerIdx >= 0 && m.branchPickerIdx < len(m.branches) {
			m.branchIdx = m.branchPickerIdx
			m.commitIdx = 0
			m.setFlash("checked out " + m.branches[m.branchIdx].Name)
		}
		m.mode = modeNormal
		return m, nil
	}
	return m, nil
}

type flatComment struct {
	depth int
	c     *Comment
}

func flattenComments(comments []Comment, depth int) []flatComment {
	idxs := make([]int, len(comments))
	for i := range idxs {
		idxs[i] = i
	}
	sort.SliceStable(idxs, func(i, j int) bool {
		return comments[idxs[i]].Likes > comments[idxs[j]].Likes
	})
	var out []flatComment
	for _, i := range idxs {
		c := &comments[i]
		out = append(out, flatComment{depth: depth, c: c})
		out = append(out, flattenComments(c.Comments, depth+1)...)
	}
	return out
}

func (m *model) detailsFlat() []flatComment {
	c := m.currentCommit()
	if c == nil {
		return nil
	}
	return flattenComments(c.Comments, 0)
}

func (m *model) detailsReplyTarget() *[]Comment {
	c := m.currentCommit()
	if c == nil {
		return nil
	}
	if m.detailsSelIdx < 0 {
		return &c.Comments
	}
	flat := flattenComments(c.Comments, 0)
	if m.detailsSelIdx >= len(flat) {
		return &c.Comments
	}
	return &flat[m.detailsSelIdx].c.Comments
}

func (m *model) detailsLikeSelected() {
	c := m.currentCommit()
	if c == nil {
		return
	}
	if m.detailsSelIdx < 0 {
		toggleLike(&c.Liked, &c.Likes)
		return
	}
	flat := flattenComments(c.Comments, 0)
	if m.detailsSelIdx >= len(flat) {
		return
	}
	toggleLike(&flat[m.detailsSelIdx].c.Liked, &flat[m.detailsSelIdx].c.Likes)
}

func toggleLike(liked *bool, likes *int) {
	if *liked {
		*liked = false
		*likes--
	} else {
		*liked = true
		*likes++
	}
}

func isSubmitKey(s string) bool {
	switch s {
	case "enter", "ctrl+s", "ctrl+d":
		return true
	}
	return false
}

func isNewlineKey(s string) bool {
	switch s {
	case "ctrl+enter", "alt+enter", "ctrl+j":
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
