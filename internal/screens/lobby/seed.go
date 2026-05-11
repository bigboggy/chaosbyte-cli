package lobby

import (
	"time"

	"github.com/bchayka/gitstatus/internal/ui"
)

// MeUser is the local user's display name. Hardcoded for this build; in a
// real app this would come from config.
const MeUser = "@boggy"

// Channel is a single chat room. Channels live entirely inside the lobby
// package — no other screen reaches in.
type Channel struct {
	Name     string
	Topic    string
	Members  int
	Online   int
	Unread   int
	Messages []ui.ChatMessage
}

func seedChannels() []Channel {
	now := time.Now()
	h := func(d time.Duration) time.Time { return now.Add(-d) }
	return []Channel{
		{
			Name: "#lobby", Topic: "main hall · be excellent · /help for commands",
			Members: 4218, Online: 612,
			Messages: []ui.ChatMessage{
				{Author: "server", Body: "MOTD: welcome to chaosbyte — type /help to see what you can do", At: h(2 * time.Hour), Kind: ui.ChatSystem},
				{Author: "server", Body: "topic: main hall · be excellent · /help for commands", At: h(2 * time.Hour), Kind: ui.ChatSystem},
				{Author: "@yamlhater", Body: "joined the lobby", At: h(48 * time.Minute), Kind: ui.ChatJoin},
				{Author: "@yamlhater", Body: "anyone else's CI just decide today was the day to be slow", At: h(38 * time.Minute), Kind: ui.ChatNormal},
				{Author: "@nullpointer", Body: "ours is just printing fortune cookies now. devops did it.", At: h(36 * time.Minute), Kind: ui.ChatNormal},
				{Author: "@devops_bard", Body: "the fortune cookies ARE the test output, you just have to interpret them", At: h(35 * time.Minute), Kind: ui.ChatNormal},
				{Author: "@junior_dev", Body: "wait is that real or are we doing bits", At: h(32 * time.Minute), Kind: ui.ChatNormal},
				{Author: "@standup_ghost", Body: "yes", At: h(30 * time.Minute), Kind: ui.ChatNormal},
				{Author: "@yamlhater", Body: "lmao", At: h(29 * time.Minute), Kind: ui.ChatNormal},
				{Author: "@recovering_pm", Body: "putting it on the roadmap", At: h(25 * time.Minute), Kind: ui.ChatNormal},
				{Author: "@ai_grifter", Body: "shrugs in latent space", At: h(20 * time.Minute), Kind: ui.ChatAction},
				{Author: "@senior_intern", Body: "i shipped a feature today by accident. it's now load-bearing.", At: h(12 * time.Minute), Kind: ui.ChatNormal},
				{Author: "@ai_grifter", Body: "every feature is an accident if you squint", At: h(8 * time.Minute), Kind: ui.ChatNormal},
				{Author: "@nullpointer", Body: "philosophy hour again i see", At: h(4 * time.Minute), Kind: ui.ChatNormal},
				{Author: "@yamlhater", Body: "the only philosophy is rollback", At: h(90 * time.Second), Kind: ui.ChatNormal},
			},
		},
		{
			Name: "#general", Topic: "general dev chatter",
			Members: 1842, Online: 312,
			Messages: []ui.ChatMessage{
				{Author: "@vibe_master", Body: "anyone tried jujutsu yet", At: h(50 * time.Minute), Kind: ui.ChatNormal},
				{Author: "@borrow_checker", Body: "yes. it's git but without the trauma. would recommend.", At: h(45 * time.Minute), Kind: ui.ChatNormal},
			},
		},
		{
			Name: "#vibe-coding", Topic: "for when the linter has feelings",
			Members: 904, Online: 198,
			Messages: []ui.ChatMessage{
				{Author: "@vibe_master", Body: "tonight's stack: bun + zod + a single prayer", At: h(2 * time.Hour), Kind: ui.ChatNormal},
				{Author: "@yamlhater", Body: "you forgot the README that lies about what it does", At: h(95 * time.Minute), Kind: ui.ChatNormal},
				{Author: "@vibe_master", Body: "the README is implied. the vibes ARE the README.", At: h(90 * time.Minute), Kind: ui.ChatNormal},
				{Author: "@ai_grifter", Body: "i asked claude to refactor and it wrote a haiku instead. shipped.", At: h(70 * time.Minute), Kind: ui.ChatNormal},
				{Author: "@standup_ghost", Body: "post the haiku", At: h(65 * time.Minute), Kind: ui.ChatNormal},
				{Author: "@ai_grifter", Body: "your tests pass / the prod servers all weep / nobody knows why", At: h(63 * time.Minute), Kind: ui.ChatNormal},
				{Author: "@nullpointer", Body: "okay that's actually good", At: h(60 * time.Minute), Kind: ui.ChatNormal},
				{Author: "@vibe_master", Body: "deploy on a friday they said. it'll be fine they said.", At: h(20 * time.Minute), Kind: ui.ChatNormal},
				{Author: "@yamlhater", Body: "what could go wrong, it's just a small change", At: h(18 * time.Minute), Kind: ui.ChatNormal},
				{Author: "@devops_bard", Body: "the change was small. the blast radius was not.", At: h(15 * time.Minute), Kind: ui.ChatNormal},
			},
		},
		{
			Name: "#rust-anonymous", Topic: "you don't have to talk about rust. but you will.",
			Members: 521, Online: 87,
			Messages: []ui.ChatMessage{
				{Author: "@borrow_checker", Body: "i fought the compiler. the compiler won. again.", At: h(4 * time.Hour), Kind: ui.ChatNormal},
				{Author: "@nullpointer", Body: "you'll thank it in production", At: h(3*time.Hour + 50*time.Minute), Kind: ui.ChatNormal},
				{Author: "@borrow_checker", Body: "i AM in production. i'm just very tired.", At: h(3*time.Hour + 45*time.Minute), Kind: ui.ChatNormal},
				{Author: "@yamlhater", Body: "rewrite it in rust", At: h(3 * time.Hour), Kind: ui.ChatNormal},
				{Author: "@senior_intern", Body: "rewrote it in rust. now it segfaults faster.", At: h(2*time.Hour + 30*time.Minute), Kind: ui.ChatNormal},
			},
		},
		{
			Name: "#help", Topic: "actual help, occasionally",
			Members: 2104, Online: 411,
			Messages: []ui.ChatMessage{
				{Author: "@junior_dev", Body: "is it normal for `git push --force` to feel this good", At: h(50 * time.Minute), Kind: ui.ChatNormal},
				{Author: "@devops_bard", Body: "yes. that's how they get you.", At: h(48 * time.Minute), Kind: ui.ChatNormal},
				{Author: "@yamlhater", Body: "the rush is real and the consequences are realer", At: h(45 * time.Minute), Kind: ui.ChatNormal},
			},
		},
		{
			Name: "#side-projects", Topic: "show what you're building (or pretending to)",
			Members: 1207, Online: 256,
			Messages: []ui.ChatMessage{
				{Author: "@vibe_master", Body: "spent 6 hours on the landing page, 12 minutes on the product", At: h(3 * time.Hour), Kind: ui.ChatNormal},
				{Author: "@yamlhater", Body: "the landing page IS the product", At: h(2*time.Hour + 50*time.Minute), Kind: ui.ChatNormal},
			},
		},
		{
			Name: "#offtopic", Topic: "feelings, snacks, and bad takes",
			Members: 887, Online: 142,
			Messages: []ui.ChatMessage{
				{Author: "@standup_ghost", Body: "objectively the best ide is the one that doesn't crash today", At: h(6 * time.Hour), Kind: ui.ChatNormal},
				{Author: "@yamlhater", Body: "that's a moving target", At: h(5*time.Hour + 50*time.Minute), Kind: ui.ChatNormal},
			},
		},
	}
}
