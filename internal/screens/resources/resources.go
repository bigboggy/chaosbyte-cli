// Package resources is the skills + github-repo browser with a search tab.
// Tabs are: trending skills, top skills, github repos, search.
package resources

import (
	"fmt"
	"strings"

	"github.com/bchayka/gitstatus/internal/screens"
	"github.com/bchayka/gitstatus/internal/theme"
	"github.com/bchayka/gitstatus/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	tabTrending = iota
	tabTop
	tabRepos
	tabSearch
)

var tabLabels = []string{"trending skills", "top skills", "github repos", "search"}

type Screen struct {
	tab         int
	idx         int
	query       string
	queryActive bool

	trending []Skill
	top      []Skill
	repos    []Repo
}

func New() *Screen {
	return &Screen{
		trending: seedTrending(),
		top:      seedTop(),
		repos:    seedRepos(),
	}
}

func (s *Screen) Init() tea.Cmd { return nil }

func (s *Screen) Name() string  { return screens.ResourcesID }
func (s *Screen) Title() string { return "resources" }

func (s *Screen) HeaderContext() string {
	return lipgloss.NewStyle().Foreground(theme.Muted).Render(tabLabels[s.tab])
}

func (s *Screen) Footer() []screens.KeyHint {
	if s.queryActive {
		return []screens.KeyHint{
			{Key: "type", Desc: "filter"}, {Key: "enter", Desc: "submit"}, {Key: "esc", Desc: "back"},
		}
	}
	return []screens.KeyHint{
		{Key: "tab", Desc: "tabs"}, {Key: "j/k", Desc: "move"}, {Key: "enter", Desc: "open"},
		{Key: "/", Desc: "search"}, {Key: "esc", Desc: "lobby"},
	}
}

func (s *Screen) InputFocused() bool { return s.queryActive }

func (s *Screen) Update(msg tea.Msg) (screens.Screen, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}
	if s.queryActive {
		return s.updateSearch(km)
	}
	return s.updateBrowse(km)
}

func (s *Screen) updateBrowse(km tea.KeyMsg) (screens.Screen, tea.Cmd) {
	switch km.String() {
	case "tab":
		s.tab = (s.tab + 1) % len(tabLabels)
		s.idx = 0
	case "shift+tab":
		s.tab = (s.tab - 1 + len(tabLabels)) % len(tabLabels)
		s.idx = 0
	case "j", "down":
		if s.idx < s.listLen()-1 {
			s.idx++
		}
	case "k", "up":
		if s.idx > 0 {
			s.idx--
		}
	case "g":
		s.idx = 0
	case "G":
		s.idx = s.listLen() - 1
	case "/":
		s.tab = tabSearch
		s.queryActive = true
		s.idx = 0
	case "enter", "o":
		return s, s.open()
	}
	return s, nil
}

func (s *Screen) updateSearch(km tea.KeyMsg) (screens.Screen, tea.Cmd) {
	switch km.String() {
	case "esc":
		s.queryActive = false
		return s, nil
	case "enter":
		s.queryActive = false
		return s, screens.Flash(fmt.Sprintf("search returned %d results for %q",
			len(s.searchResults()), s.query))
	case "backspace":
		if len(s.query) > 0 {
			s.query = s.query[:len(s.query)-1]
		}
		return s, nil
	}
	t := km.String()
	if len(t) == 1 {
		s.query += t
	} else if t == "space" {
		s.query += " "
	}
	return s, nil
}

func (s *Screen) listLen() int {
	switch s.tab {
	case tabTrending:
		return len(s.trending)
	case tabTop:
		return len(s.top)
	case tabRepos:
		return len(s.repos)
	case tabSearch:
		return len(s.searchResults())
	}
	return 0
}

func (s *Screen) open() tea.Cmd {
	switch s.tab {
	case tabTrending:
		if s.idx < len(s.trending) {
			return screens.Flash("skills.sh/" + s.trending[s.idx].Name)
		}
	case tabTop:
		if s.idx < len(s.top) {
			return screens.Flash("skills.sh/" + s.top[s.idx].Name)
		}
	case tabRepos:
		if s.idx < len(s.repos) {
			return screens.Flash("opening: " + s.repos[s.idx].URL)
		}
	case tabSearch:
		results := s.searchResults()
		if s.idx < len(results) {
			return screens.Flash("skills.sh/" + results[s.idx].Name)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (s *Screen) View(width, height int) string {
	w := ui.FeedShellWidth(width)
	contentW := w - 2

	title := theme.Title.Render("resources")
	tabsRow := renderTabs(s.tab, contentW)

	var body string
	switch s.tab {
	case tabTrending:
		body = s.renderSkillList(s.trending, contentW, "trending")
	case tabTop:
		body = s.renderSkillList(s.top, contentW, "top")
	case tabRepos:
		body = s.renderRepoList(contentW)
	case tabSearch:
		body = s.renderSearch(contentW)
	}

	stacked := lipgloss.JoinVertical(lipgloss.Left, title, "", tabsRow, "", body)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Top, stacked)
}

func renderTabs(active, width int) string {
	var tabs []string
	for i, t := range tabLabels {
		label := fmt.Sprintf("%d %s", i+1, t)
		if i == active {
			tabs = append(tabs, theme.TabActive.Render(label))
		} else {
			tabs = append(tabs, theme.TabInactive.Render(label))
		}
	}
	return strings.Join(tabs, "  ")
}

func (s *Screen) renderSkillList(skills []Skill, width int, kind string) string {
	hint := lipgloss.NewStyle().Foreground(theme.Muted).Italic(true).
		Render(fmt.Sprintf("%d %s skills · enter to open · / to search", len(skills), kind))

	var rows []string
	for i, sk := range skills {
		rows = append(rows, renderSkillRow(sk, width, i == s.idx))
	}
	return hint + "\n\n" + strings.Join(rows, "\n")
}

func renderSkillRow(sk Skill, width int, focused bool) string {
	marker := "  "
	if focused {
		marker = "▸ "
	}
	trendColor := theme.OK
	if !sk.Up {
		trendColor = theme.Like
	}
	trend := lipgloss.NewStyle().Foreground(trendColor).Bold(true).Render(fmt.Sprintf("%6s", sk.Trend))
	name := lipgloss.NewStyle().Foreground(theme.Accent2).Bold(true).Render(fmt.Sprintf("%-14s", ui.Truncate(sk.Name, 14)))
	cat := lipgloss.NewStyle().Foreground(theme.Accent).Render(fmt.Sprintf("%-11s", ui.Truncate(sk.Category, 11)))
	desc := lipgloss.NewStyle().Foreground(theme.Fg).Render(ui.Truncate(sk.Description, width-44))
	score := lipgloss.NewStyle().Foreground(theme.Muted).Render(fmt.Sprintf("· %d", sk.Score))

	line := fmt.Sprintf("%s%s  %s  %s  %s  %s", marker, trend, name, cat, desc, score)
	if focused {
		return theme.BranchItemSel.Width(width).Render(line)
	}
	return theme.BranchItem.Render(line)
}

func (s *Screen) renderRepoList(width int) string {
	hint := lipgloss.NewStyle().Foreground(theme.Muted).Italic(true).
		Render(fmt.Sprintf("%d highlighted github repos · enter to open", len(s.repos)))

	var rows []string
	for i, r := range s.repos {
		rows = append(rows, renderRepoRow(r, width, i == s.idx))
	}
	return hint + "\n\n" + strings.Join(rows, "\n")
}

func renderRepoRow(r Repo, width int, focused bool) string {
	marker := "  "
	if focused {
		marker = "▸ "
	}
	name := lipgloss.NewStyle().Foreground(theme.Accent2).Bold(true).Render(r.Owner + "/" + r.Name)
	lang := lipgloss.NewStyle().Foreground(theme.Accent).Render(fmt.Sprintf("[%s]", r.Language))
	stars := lipgloss.NewStyle().Foreground(theme.Warn).Render(fmt.Sprintf("★ %d", r.Stars))
	forks := lipgloss.NewStyle().Foreground(theme.Muted).Render(fmt.Sprintf("⑂ %d", r.Forks))
	desc := lipgloss.NewStyle().Foreground(theme.Fg).Render(ui.Truncate(r.Description, width-8))

	head := fmt.Sprintf("%s%s  %s  %s  %s", marker, name, lang, stars, forks)
	box := lipgloss.NewStyle().Padding(0, 1).Width(width - 2)
	if focused {
		box = box.Border(lipgloss.RoundedBorder()).BorderForeground(theme.Accent)
	} else {
		box = box.Border(lipgloss.HiddenBorder())
	}
	return box.Render(head + "\n" + desc)
}

func (s *Screen) renderSearch(width int) string {
	cursor := " "
	if s.queryActive {
		cursor = lipgloss.NewStyle().Foreground(theme.Accent).Render("▌")
	}
	prompt := lipgloss.NewStyle().Foreground(theme.OK).Render("search > ")
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Accent).
		Padding(0, 1).
		Width(width - 2).
		Render(prompt + s.query + cursor)

	results := s.searchResults()
	hint := lipgloss.NewStyle().Foreground(theme.Muted).Italic(true).
		Render(fmt.Sprintf("%d results · press / to edit · enter to submit · esc to leave search",
			len(results)))

	var rows []string
	for i, sk := range results {
		if i >= 12 {
			rows = append(rows, lipgloss.NewStyle().Foreground(theme.Muted).
				Render(fmt.Sprintf("... %d more", len(results)-12)))
			break
		}
		rows = append(rows, renderSearchRow(sk, width, i == s.idx))
	}
	return lipgloss.JoinVertical(lipgloss.Left, box, hint, "", strings.Join(rows, "\n"))
}

func renderSearchRow(sk Skill, width int, focused bool) string {
	marker := "  "
	if focused {
		marker = "▸ "
	}
	name := lipgloss.NewStyle().Foreground(theme.Accent2).Bold(true).Render(fmt.Sprintf("%-14s", ui.Truncate(sk.Name, 14)))
	cat := lipgloss.NewStyle().Foreground(theme.Accent).Render(fmt.Sprintf("%-11s", ui.Truncate(sk.Category, 11)))
	desc := lipgloss.NewStyle().Foreground(theme.Fg).Render(ui.Truncate(sk.Description, width-32))
	line := fmt.Sprintf("%s%s  %s  %s", marker, name, cat, desc)
	if focused {
		return theme.BranchItemSel.Width(width - 2).Render(line)
	}
	return theme.BranchItem.Render(line)
}
