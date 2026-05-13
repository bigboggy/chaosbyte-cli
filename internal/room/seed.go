package room

import (
	"time"

	"github.com/bchayka/gitstatus/internal/ui"
)

// seedMessages returns the scrollback the broker starts with so new sessions
// see a conversation in progress. All channels here are shared across SSH
// sessions; per-session metadata (topic, member count) stays on the lobby.
func seedMessages() map[string][]ui.ChatMessage {
	now := time.Now()
	h := func(d time.Duration) time.Time { return now.Add(-d) }
	return map[string][]ui.ChatMessage{
		"#lobby": {
			{Author: "server", Body: "the workshop is open. :help when you need it. :leave when you go.", At: h(2 * time.Hour), Kind: ui.ChatSystem},
			{Author: "server", Body: "tonight in the spotlight: tinytty, by rin. a 4kb terminal renderer. :read to open.", At: h(2 * time.Hour), Kind: ui.ChatSystem},
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
		"#general": {
			{Author: "@vibe_master", Body: "anyone tried jujutsu yet", At: h(50 * time.Minute), Kind: ui.ChatNormal},
			{Author: "@borrow_checker", Body: "yes. it's git but without the trauma. would recommend.", At: h(45 * time.Minute), Kind: ui.ChatNormal},
		},
		"#vibe-coding": {
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
		"#rust-anonymous": {
			{Author: "@borrow_checker", Body: "i fought the compiler. the compiler won. again.", At: h(4 * time.Hour), Kind: ui.ChatNormal},
			{Author: "@nullpointer", Body: "you'll thank it in production", At: h(3*time.Hour + 50*time.Minute), Kind: ui.ChatNormal},
			{Author: "@borrow_checker", Body: "i AM in production. i'm just very tired.", At: h(3*time.Hour + 45*time.Minute), Kind: ui.ChatNormal},
			{Author: "@yamlhater", Body: "rewrite it in rust", At: h(3 * time.Hour), Kind: ui.ChatNormal},
			{Author: "@senior_intern", Body: "rewrote it in rust. now it segfaults faster.", At: h(2*time.Hour + 30*time.Minute), Kind: ui.ChatNormal},
		},
		"#help": {
			{Author: "@junior_dev", Body: "is it normal for `git push --force` to feel this good", At: h(50 * time.Minute), Kind: ui.ChatNormal},
			{Author: "@devops_bard", Body: "yes. that's how they get you.", At: h(48 * time.Minute), Kind: ui.ChatNormal},
			{Author: "@yamlhater", Body: "the rush is real and the consequences are realer", At: h(45 * time.Minute), Kind: ui.ChatNormal},
		},
		"#side-projects": {
			{Author: "@vibe_master", Body: "spent 6 hours on the landing page, 12 minutes on the product", At: h(3 * time.Hour), Kind: ui.ChatNormal},
			{Author: "@yamlhater", Body: "the landing page IS the product", At: h(2*time.Hour + 50*time.Minute), Kind: ui.ChatNormal},
		},
		"#offtopic": {
			{Author: "@standup_ghost", Body: "objectively the best ide is the one that doesn't crash today", At: h(6 * time.Hour), Kind: ui.ChatNormal},
			{Author: "@yamlhater", Body: "that's a moving target", At: h(5*time.Hour + 50*time.Minute), Kind: ui.ChatNormal},
		},
	}
}
