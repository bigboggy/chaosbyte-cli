package main

import (
	"time"
)

// ChatKind classifies a chat message for rendering.
type ChatKind int

const (
	ChatNormal ChatKind = iota
	ChatSystem
	ChatAction
	ChatJoin
)

type ChatMessage struct {
	Author string
	Body   string
	At     time.Time
	Kind   ChatKind
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
			Name: "#lobby", Topic: "main hall · be excellent · /help for commands",
			Members: 4218, Online: 612,
			Messages: []ChatMessage{
				{"server", "MOTD: welcome to chaosbyte — type /help to see what you can do", h(2 * time.Hour), ChatSystem},
				{"server", "topic: main hall · be excellent · /help for commands", h(2 * time.Hour), ChatSystem},
				{"@yamlhater", "joined the lobby", h(48 * time.Minute), ChatJoin},
				{"@yamlhater", "anyone else's CI just decide today was the day to be slow", h(38 * time.Minute), ChatNormal},
				{"@nullpointer", "ours is just printing fortune cookies now. devops did it.", h(36 * time.Minute), ChatNormal},
				{"@devops_bard", "the fortune cookies ARE the test output, you just have to interpret them", h(35 * time.Minute), ChatNormal},
				{"@junior_dev", "wait is that real or are we doing bits", h(32 * time.Minute), ChatNormal},
				{"@standup_ghost", "yes", h(30 * time.Minute), ChatNormal},
				{"@yamlhater", "lmao", h(29 * time.Minute), ChatNormal},
				{"@recovering_pm", "putting it on the roadmap", h(25 * time.Minute), ChatNormal},
				{"@ai_grifter", "shrugs in latent space", h(20 * time.Minute), ChatAction},
				{"@senior_intern", "i shipped a feature today by accident. it's now load-bearing.", h(12 * time.Minute), ChatNormal},
				{"@ai_grifter", "every feature is an accident if you squint", h(8 * time.Minute), ChatNormal},
				{"@nullpointer", "philosophy hour again i see", h(4 * time.Minute), ChatNormal},
				{"@yamlhater", "the only philosophy is rollback", h(90 * time.Second), ChatNormal},
			},
		},
		{
			Name: "#general", Topic: "general dev chatter",
			Members: 1842, Online: 312,
			Messages: []ChatMessage{
				{"@vibe_master", "anyone tried jujutsu yet", h(50 * time.Minute), ChatNormal},
				{"@borrow_checker", "yes. it's git but without the trauma. would recommend.", h(45 * time.Minute), ChatNormal},
			},
		},
		{
			Name: "#vibe-coding", Topic: "for when the linter has feelings",
			Members: 904, Online: 198,
			Messages: []ChatMessage{
				{"@vibe_master", "tonight's stack: bun + zod + a single prayer", h(2 * time.Hour), ChatNormal},
				{"@yamlhater", "you forgot the README that lies about what it does", h(95 * time.Minute), ChatNormal},
				{"@vibe_master", "the README is implied. the vibes ARE the README.", h(90 * time.Minute), ChatNormal},
				{"@ai_grifter", "i asked claude to refactor and it wrote a haiku instead. shipped.", h(70 * time.Minute), ChatNormal},
				{"@standup_ghost", "post the haiku", h(65 * time.Minute), ChatNormal},
				{"@ai_grifter", "your tests pass / the prod servers all weep / nobody knows why", h(63 * time.Minute), ChatNormal},
				{"@nullpointer", "okay that's actually good", h(60 * time.Minute), ChatNormal},
				{"@vibe_master", "deploy on a friday they said. it'll be fine they said.", h(20 * time.Minute), ChatNormal},
				{"@yamlhater", "what could go wrong, it's just a small change", h(18 * time.Minute), ChatNormal},
				{"@devops_bard", "the change was small. the blast radius was not.", h(15 * time.Minute), ChatNormal},
			},
		},
		{
			Name: "#rust-anonymous", Topic: "you don't have to talk about rust. but you will.",
			Members: 521, Online: 87,
			Messages: []ChatMessage{
				{"@borrow_checker", "i fought the compiler. the compiler won. again.", h(4 * time.Hour), ChatNormal},
				{"@nullpointer", "you'll thank it in production", h(3*time.Hour + 50*time.Minute), ChatNormal},
				{"@borrow_checker", "i AM in production. i'm just very tired.", h(3*time.Hour + 45*time.Minute), ChatNormal},
				{"@yamlhater", "rewrite it in rust", h(3 * time.Hour), ChatNormal},
				{"@senior_intern", "rewrote it in rust. now it segfaults faster.", h(2*time.Hour + 30*time.Minute), ChatNormal},
			},
		},
		{
			Name: "#help", Topic: "actual help, occasionally",
			Members: 2104, Online: 411,
			Messages: []ChatMessage{
				{"@junior_dev", "is it normal for `git push --force` to feel this good", h(50 * time.Minute), ChatNormal},
				{"@devops_bard", "yes. that's how they get you.", h(48 * time.Minute), ChatNormal},
				{"@yamlhater", "the rush is real and the consequences are realer", h(45 * time.Minute), ChatNormal},
			},
		},
		{
			Name: "#side-projects", Topic: "show what you're building (or pretending to)",
			Members: 1207, Online: 256,
			Messages: []ChatMessage{
				{"@vibe_master", "spent 6 hours on the landing page, 12 minutes on the product", h(3 * time.Hour), ChatNormal},
				{"@yamlhater", "the landing page IS the product", h(2*time.Hour + 50*time.Minute), ChatNormal},
			},
		},
		{
			Name: "#offtopic", Topic: "feelings, snacks, and bad takes",
			Members: 887, Online: 142,
			Messages: []ChatMessage{
				{"@standup_ghost", "objectively the best ide is the one that doesn't crash today", h(6 * time.Hour), ChatNormal},
				{"@yamlhater", "that's a moving target", h(5*time.Hour + 50*time.Minute), ChatNormal},
			},
		},
	}
}
