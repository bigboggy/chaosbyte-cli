package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type NewsItem struct {
	Source   string // HN, Lobsters, /r/programming, DevHQ, ArsTechnica
	Title    string
	URL      string
	Author   string
	Score    int
	Comments int
	At       time.Time
}

func seedNews() []NewsItem {
	now := time.Now()
	h := func(d time.Duration) time.Time { return now.Add(-d) }
	return []NewsItem{
		{"HN", "Show HN: I built a thing in 4 hours that probably shouldn't exist",
			"https://news.ycombinator.com/item?id=40000001", "@vibe_master", 1842, 312, h(38 * time.Minute)},
		{"HN", "Ask HN: my CI has feelings now, is this normal?",
			"https://news.ycombinator.com/item?id=40000002", "@yamlhater", 904, 198, h(58 * time.Minute)},
		{"Lobsters", "Why we rewrote our compiler in our compiler, again",
			"https://lobste.rs/s/abcd01/why_we_rewrote", "@nullpointer", 421, 76, h(2 * time.Hour)},
		{"HN", "A 12,000-line postmortem of the time I wrote one if-statement",
			"https://news.ycombinator.com/item?id=40000003", "@devops_bard", 2810, 511, h(3 * time.Hour)},
		{"/r/programming", "PSA: that StackOverflow answer from 2014 is now load-bearing infrastructure",
			"https://reddit.com/r/programming/comments/psa", "@senior_intern", 6712, 884, h(4 * time.Hour)},
		{"DevHQ", "The state of TypeScript types in 2026: still arguing about narrowing",
			"https://devhq.example/state-of-ts-2026", "@borrow_checker", 1320, 207, h(5 * time.Hour)},
		{"HN", "Show HN: chrome extension that replaces \"AI\" with \"a guess\" on every page",
			"https://news.ycombinator.com/item?id=40000004", "@ai_grifter", 8120, 1402, h(7 * time.Hour)},
		{"Lobsters", "I read the entire Linux kernel mailing list so you don't have to",
			"https://lobste.rs/s/efgh02/lkml", "@yamlhater", 590, 41, h(9 * time.Hour)},
		{"ArsTechnica", "AI coding tools now responsible for 80% of bugs they were hired to fix",
			"https://arstechnica.example/ai-bugs", "ars staff", 3402, 712, h(11 * time.Hour)},
		{"/r/programming", "Anyone else's deploy script just print fortune cookies now",
			"https://reddit.com/r/programming/comments/fortune", "@standup_ghost", 1102, 184, h(13 * time.Hour)},
		{"HN", "The vibes-driven development manifesto",
			"https://news.ycombinator.com/item?id=40000005", "@vibe_master", 4982, 922, h(16 * time.Hour)},
		{"DevHQ", "Postgres is faster than you. It's faster than your startup. It's faster than your dreams.",
			"https://devhq.example/postgres-faster", "@nullpointer", 2240, 411, h(20 * time.Hour)},
		{"Lobsters", "How we cut p99 latency by 87% by removing the part that did things",
			"https://lobste.rs/s/ijkl03/p99", "@recovering_pm", 712, 102, h(22 * time.Hour)},
		{"HN", "Show HN: a static site generator that's 12 lines of bash. yes it's HTML.",
			"https://news.ycombinator.com/item?id=40000006", "@junior_dev", 1881, 244, h(28 * time.Hour)},
		{"ArsTechnica", "FAANG hiring is back. So is the LeetCode hazing. So is the bathroom crying.",
			"https://arstechnica.example/faang-hiring", "ars staff", 5601, 1881, h(30 * time.Hour)},
	}
}

func (m model) updateNews(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.newsIdx < len(m.newsItems)-1 {
			m.newsIdx++
		}
	case "k", "up":
		if m.newsIdx > 0 {
			m.newsIdx--
		}
	case "g":
		m.newsIdx = 0
	case "G":
		m.newsIdx = len(m.newsItems) - 1
	case "enter", "o":
		if m.newsIdx < len(m.newsItems) {
			m.setFlash("opening: " + m.newsItems[m.newsIdx].URL)
		}
	case "y":
		if m.newsIdx < len(m.newsItems) {
			m.setFlash("url copied (in spirit): " + m.newsItems[m.newsIdx].URL)
		}
	}
	return m, nil
}

func (m model) renderNews(width, height int) string {
	w := feedShellWidth(width)
	title := titleStyle.Render("news · combined feed")
	subtitle := lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
		Render("HN · Lobsters · /r/programming · DevHQ · ArsTechnica")

	contentW := w - 2
	cardH := 4
	bodyH := height - 4 // title + subtitle + blank + footer-ish
	if bodyH < cardH {
		bodyH = cardH
	}

	visibleCount := bodyH / (cardH + 1)
	if visibleCount < 1 {
		visibleCount = 1
	}

	// keep selection in view
	if m.newsIdx < m.newsScroll {
		m.newsScroll = m.newsIdx
	}
	if m.newsIdx >= m.newsScroll+visibleCount {
		m.newsScroll = m.newsIdx - visibleCount + 1
	}
	end := m.newsScroll + visibleCount
	if end > len(m.newsItems) {
		end = len(m.newsItems)
	}

	var cards []string
	for i := m.newsScroll; i < end; i++ {
		cards = append(cards, renderNewsCard(m.newsItems[i], contentW, i == m.newsIdx))
	}
	body := strings.Join(cards, "\n")

	indicator := ""
	if len(m.newsItems) > visibleCount {
		pct := 100
		if denom := len(m.newsItems) - visibleCount; denom > 0 {
			pct = m.newsScroll * 100 / denom
		}
		indicator = lipgloss.NewStyle().Foreground(colorMuted).Width(contentW).Align(lipgloss.Right).
			Render(fmt.Sprintf("scroll %d%%   %d/%d", pct, m.newsIdx+1, len(m.newsItems)))
	}

	stacked := lipgloss.JoinVertical(lipgloss.Left,
		title, subtitle, indicator, "", body,
	)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Top, stacked)
}

func renderNewsCard(it NewsItem, width int, focused bool) string {
	innerW := width - 4

	source := newsSourceTag(it.Source)
	title := lipgloss.NewStyle().Foreground(colorFg).Bold(true).
		Render(truncate(it.Title, innerW-len(it.Source)-3))
	header := source + "  " + title

	meta := fmt.Sprintf("%s ▲ %d   %s 💬 %d   %s   %s",
		likeStyle.Render("▲"), it.Score,
		commentCountStyle.Render("💬"), it.Comments,
		commitAuthorStyle.Render(it.Author),
		commitTimeStyle.Render(humanizeTime(it.At)),
	)
	_ = meta

	url := lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render(truncate(it.URL, innerW))

	content := lipgloss.JoinVertical(lipgloss.Left, header, meta, url)

	box := lipgloss.NewStyle().Padding(0, 1).Width(innerW)
	if focused {
		box = box.Border(lipgloss.RoundedBorder()).BorderForeground(colorAccent)
	} else {
		box = box.Border(lipgloss.HiddenBorder())
	}
	return box.Render(content)
}

func newsSourceTag(s string) string {
	var bg lipgloss.Color
	switch s {
	case "HN":
		bg = colorWarn
	case "Lobsters":
		bg = colorOk
	case "/r/programming":
		bg = colorLike
	case "DevHQ":
		bg = colorAccent
	case "ArsTechnica":
		bg = colorAccent2
	default:
		bg = colorMuted
	}
	return lipgloss.NewStyle().Foreground(colorBg).Background(bg).Bold(true).Padding(0, 1).Render(s)
}
