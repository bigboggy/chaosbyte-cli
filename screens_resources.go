package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Skill struct {
	Name        string
	Trend       string // "+12%", "-3%"
	Up          bool
	Score       int // popularity score
	Category    string
	Description string
}

type Repo struct {
	Owner       string
	Name        string
	Description string
	Stars       int
	Forks       int
	Language    string
	URL         string
}

func seedTrendingSkills() []Skill {
	return []Skill{
		{"bun", "+47%", true, 9120, "runtime", "the node-killer that still hasn't killed node but is trying"},
		{"zod-3", "+38%", true, 7204, "validation", "runtime types because typescript can't be trusted alone"},
		{"htmx", "+33%", true, 8431, "frontend", "the html-is-fine framework for people tired of frameworks"},
		{"sqlite-vec", "+28%", true, 4112, "embeddings", "vectors in sqlite, because you didn't need a vector db"},
		{"tauri-2", "+24%", true, 6801, "desktop", "electron but it doesn't eat 800mb of ram to render a button"},
		{"effect-ts", "+19%", true, 3920, "fp", "the functional library that explains itself in 47 medium posts"},
		{"valkey", "+18%", true, 5102, "cache", "redis but for people with feelings about licenses"},
		{"deno-2", "+15%", true, 4881, "runtime", "still trying, still kind of working"},
		{"jujutsu", "+13%", true, 2104, "vcs", "git but with fewer footguns and more confused devs"},
		{"astro-5", "+11%", true, 6712, "ssg", "ship html, get medals"},
		{"rspack", "+9%", true, 3204, "bundler", "webpack in rust, for when you have 47 entry points"},
		{"biome", "+8%", true, 4109, "tooling", "linter + formatter that finally agrees with itself"},
		{"webgpu", "+6%", true, 2410, "gpu", "the graphics api the browser actually deserves"},
		{"oxc", "+4%", true, 1820, "tooling", "js tooling in rust, because everything must be in rust"},
		{"react-19", "-2%", false, 12044, "frontend", "the framework you can't quit even when you want to"},
	}
}

func seedTopSkills() []Skill {
	return []Skill{
		{"typescript", "—", true, 99412, "language", "javascript with feelings about correctness"},
		{"react", "—", true, 92044, "frontend", "the framework that won and now everyone resents"},
		{"postgres", "—", true, 87120, "database", "the answer to every database question, including the wrong ones"},
		{"docker", "—", true, 81002, "infra", "it works on your container, ship the container"},
		{"kubernetes", "—", true, 74301, "infra", "1000 yaml files in a trench coat pretending to be infra"},
		{"rust", "—", true, 71204, "language", "your borrow checker is the senior engineer you deserve"},
		{"go", "—", true, 68210, "language", "if err != nil { return nil, err }. and again. and again."},
		{"nginx", "—", true, 65120, "web", "reverse-proxy your problems away"},
		{"redis", "—", true, 61420, "cache", "the in-memory store that's also a queue, db, lock, oracle"},
		{"vim", "—", true, 58910, "editor", "you'll figure out how to quit eventually. or not."},
	}
}

func seedRepos() []Repo {
	return []Repo{
		{"oven-sh", "bun", "Incredibly fast JavaScript runtime, bundler, transpiler, and package manager",
			78201, 2940, "Zig", "https://github.com/oven-sh/bun"},
		{"ggerganov", "llama.cpp", "Inference of LLaMA models in pure C/C++",
			69210, 9810, "C++", "https://github.com/ggerganov/llama.cpp"},
		{"jj-vcs", "jj", "A Git-compatible VCS that is both simple and powerful",
			18402, 612, "Rust", "https://github.com/jj-vcs/jj"},
		{"astral-sh", "uv", "An extremely fast Python package and project manager, written in Rust",
			34102, 920, "Rust", "https://github.com/astral-sh/uv"},
		{"biomejs", "biome", "A toolchain for web projects, aimed to provide functionalities to maintain them",
			17820, 540, "Rust", "https://github.com/biomejs/biome"},
		{"htmx-org", "htmx", "</> htmx - high power tools for HTML",
			41210, 1402, "JavaScript", "https://github.com/htmx-org/htmx"},
		{"tauri-apps", "tauri", "Build smaller, faster, and more secure desktop applications with a web frontend",
			84102, 2540, "Rust", "https://github.com/tauri-apps/tauri"},
		{"colinhacks", "zod", "TypeScript-first schema validation with static type inference",
			37210, 1320, "TypeScript", "https://github.com/colinhacks/zod"},
		{"withastro", "astro", "The web framework for content-driven websites",
			47102, 2410, "TypeScript", "https://github.com/withastro/astro"},
		{"charmbracelet", "bubbletea", "A powerful little TUI framework",
			28412, 880, "Go", "https://github.com/charmbracelet/bubbletea"},
		{"sqlite", "sqlite", "Official Git mirror of the SQLite source tree",
			7102, 410, "C", "https://github.com/sqlite/sqlite"},
		{"valkey-io", "valkey", "A flexible distributed key-value datastore, BSD-licensed",
			19204, 720, "C", "https://github.com/valkey-io/valkey"},
	}
}

var resourceTabs = []string{"trending skills", "top skills", "github repos", "search"}

func (m model) updateResources(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.resourcesQueryActive {
		return m.updateResourcesSearch(msg)
	}
	switch msg.String() {
	case "tab":
		m.resourcesTab = (m.resourcesTab + 1) % len(resourceTabs)
		m.resourcesIdx = 0
	case "shift+tab":
		m.resourcesTab = (m.resourcesTab - 1 + len(resourceTabs)) % len(resourceTabs)
		m.resourcesIdx = 0
	case "j", "down":
		max := m.resourcesListLen() - 1
		if m.resourcesIdx < max {
			m.resourcesIdx++
		}
	case "k", "up":
		if m.resourcesIdx > 0 {
			m.resourcesIdx--
		}
	case "g":
		m.resourcesIdx = 0
	case "G":
		m.resourcesIdx = m.resourcesListLen() - 1
	case "/":
		m.resourcesTab = 3
		m.resourcesQueryActive = true
		m.resourcesIdx = 0
	case "enter", "o":
		m.openResource()
	}
	return m, nil
}

func (m model) updateResourcesSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.resourcesQueryActive = false
		return m, nil
	case "enter":
		m.resourcesQueryActive = false
		m.setFlash(fmt.Sprintf("search returned %d results for %q", len(m.resourcesSearchResults()), m.resourcesQuery))
		return m, nil
	case "backspace":
		if len(m.resourcesQuery) > 0 {
			m.resourcesQuery = m.resourcesQuery[:len(m.resourcesQuery)-1]
		}
		return m, nil
	}
	s := msg.String()
	if len(s) == 1 {
		m.resourcesQuery += s
	} else if s == "space" {
		m.resourcesQuery += " "
	}
	return m, nil
}

func (m model) resourcesListLen() int {
	switch m.resourcesTab {
	case 0:
		return len(m.skillsTrending)
	case 1:
		return len(m.skillsTop)
	case 2:
		return len(m.repos)
	case 3:
		return len(m.resourcesSearchResults())
	}
	return 0
}

func (m model) resourcesSearchResults() []Skill {
	if m.resourcesQuery == "" {
		// when empty, show all skills concatenated
		out := append([]Skill{}, m.skillsTrending...)
		return append(out, m.skillsTop...)
	}
	q := strings.ToLower(m.resourcesQuery)
	var out []Skill
	for _, s := range append(append([]Skill{}, m.skillsTrending...), m.skillsTop...) {
		if strings.Contains(strings.ToLower(s.Name), q) ||
			strings.Contains(strings.ToLower(s.Category), q) ||
			strings.Contains(strings.ToLower(s.Description), q) {
			out = append(out, s)
		}
	}
	return out
}

func (m *model) openResource() {
	switch m.resourcesTab {
	case 0:
		if m.resourcesIdx < len(m.skillsTrending) {
			m.setFlash("skills.sh/" + m.skillsTrending[m.resourcesIdx].Name)
		}
	case 1:
		if m.resourcesIdx < len(m.skillsTop) {
			m.setFlash("skills.sh/" + m.skillsTop[m.resourcesIdx].Name)
		}
	case 2:
		if m.resourcesIdx < len(m.repos) {
			m.setFlash("opening: " + m.repos[m.resourcesIdx].URL)
		}
	case 3:
		results := m.resourcesSearchResults()
		if m.resourcesIdx < len(results) {
			m.setFlash("skills.sh/" + results[m.resourcesIdx].Name)
		}
	}
}

func (m model) renderResources(width, height int) string {
	w := feedShellWidth(width)
	contentW := w - 2

	title := titleStyle.Render("resources")
	tabs := m.renderResourceTabs(contentW)

	var body string
	switch m.resourcesTab {
	case 0:
		body = m.renderSkillList(m.skillsTrending, contentW, "trending")
	case 1:
		body = m.renderSkillList(m.skillsTop, contentW, "top")
	case 2:
		body = m.renderRepoList(contentW)
	case 3:
		body = m.renderResourcesSearch(contentW)
	}

	stacked := lipgloss.JoinVertical(lipgloss.Left, title, "", tabs, "", body)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Top, stacked)
}

func (m model) renderResourceTabs(width int) string {
	var tabs []string
	for i, t := range resourceTabs {
		label := fmt.Sprintf("%d %s", i+1, t)
		if i == m.resourcesTab {
			tabs = append(tabs, tabActiveStyle.Render(label))
		} else {
			tabs = append(tabs, tabInactiveStyle.Render(label))
		}
	}
	return strings.Join(tabs, "  ")
}

func (m model) renderSkillList(skills []Skill, width int, kind string) string {
	hint := lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
		Render(fmt.Sprintf("%d %s skills · enter to open · / to search", len(skills), kind))

	var rows []string
	for i, s := range skills {
		marker := "  "
		if i == m.resourcesIdx {
			marker = "▸ "
		}
		trendColor := colorOk
		if !s.Up {
			trendColor = colorLike
		}
		trend := lipgloss.NewStyle().Foreground(trendColor).Bold(true).Render(fmt.Sprintf("%6s", s.Trend))
		name := lipgloss.NewStyle().Foreground(colorAccent2).Bold(true).Render(fmt.Sprintf("%-14s", truncate(s.Name, 14)))
		cat := lipgloss.NewStyle().Foreground(colorAccent).Render(fmt.Sprintf("%-11s", truncate(s.Category, 11)))
		desc := lipgloss.NewStyle().Foreground(colorFg).Render(truncate(s.Description, width-44))
		score := lipgloss.NewStyle().Foreground(colorMuted).Render(fmt.Sprintf("· %d", s.Score))

		line := fmt.Sprintf("%s%s  %s  %s  %s  %s", marker, trend, name, cat, desc, score)
		if i == m.resourcesIdx {
			line = branchItemSelStyle.Width(width).Render(line)
		} else {
			line = branchItemStyle.Render(line)
		}
		rows = append(rows, line)
	}
	return hint + "\n\n" + strings.Join(rows, "\n")
}

func (m model) renderRepoList(width int) string {
	hint := lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
		Render(fmt.Sprintf("%d highlighted github repos · enter to open", len(m.repos)))

	var rows []string
	for i, r := range m.repos {
		marker := "  "
		if i == m.resourcesIdx {
			marker = "▸ "
		}
		name := lipgloss.NewStyle().Foreground(colorAccent2).Bold(true).Render(r.Owner + "/" + r.Name)
		lang := lipgloss.NewStyle().Foreground(colorAccent).Render(fmt.Sprintf("[%s]", r.Language))
		stars := lipgloss.NewStyle().Foreground(colorWarn).Render(fmt.Sprintf("★ %d", r.Stars))
		forks := lipgloss.NewStyle().Foreground(colorMuted).Render(fmt.Sprintf("⑂ %d", r.Forks))
		desc := lipgloss.NewStyle().Foreground(colorFg).Render(truncate(r.Description, width-8))

		head := fmt.Sprintf("%s%s  %s  %s  %s", marker, name, lang, stars, forks)
		line := head + "\n     " + desc
		if i == m.resourcesIdx {
			line = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorAccent).
				Padding(0, 1).
				Width(width - 2).
				Render(head + "\n" + desc)
		} else {
			line = lipgloss.NewStyle().
				Border(lipgloss.HiddenBorder()).
				Padding(0, 1).
				Width(width - 2).
				Render(head + "\n" + desc)
		}
		rows = append(rows, line)
	}
	return hint + "\n\n" + strings.Join(rows, "\n")
}

func (m model) renderResourcesSearch(width int) string {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccent).
		Padding(0, 1).
		Width(width - 2)

	cursor := " "
	if m.resourcesQueryActive {
		cursor = lipgloss.NewStyle().Foreground(colorAccent).Render("▌")
	}
	prompt := lipgloss.NewStyle().Foreground(colorOk).Render("search > ")
	inputLine := prompt + m.resourcesQuery + cursor
	searchBox := box.Render(inputLine)

	results := m.resourcesSearchResults()
	hint := lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render(
		fmt.Sprintf("%d results · press / to edit · enter to submit · esc to leave search", len(results)),
	)

	var rows []string
	for i, s := range results {
		if i >= 12 {
			rows = append(rows, lipgloss.NewStyle().Foreground(colorMuted).Render(fmt.Sprintf("... %d more", len(results)-12)))
			break
		}
		marker := "  "
		if i == m.resourcesIdx {
			marker = "▸ "
		}
		name := lipgloss.NewStyle().Foreground(colorAccent2).Bold(true).Render(fmt.Sprintf("%-14s", truncate(s.Name, 14)))
		cat := lipgloss.NewStyle().Foreground(colorAccent).Render(fmt.Sprintf("%-11s", truncate(s.Category, 11)))
		desc := lipgloss.NewStyle().Foreground(colorFg).Render(truncate(s.Description, width-32))
		line := fmt.Sprintf("%s%s  %s  %s", marker, name, cat, desc)
		if i == m.resourcesIdx {
			line = branchItemSelStyle.Width(width - 2).Render(line)
		} else {
			line = branchItemStyle.Render(line)
		}
		rows = append(rows, line)
	}
	return lipgloss.JoinVertical(lipgloss.Left, searchBox, hint, "", strings.Join(rows, "\n"))
}
