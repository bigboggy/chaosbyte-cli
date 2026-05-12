package room

import (
	"time"

	"github.com/bchayka/gitstatus/internal/ui"
)

// seedMessages returns the scrollback the broker starts with so new sessions
// see a conversation in progress. Only #lobby is shared today; other channels
// stay on the lobby Screen.
func seedMessages() map[string][]ui.ChatMessage {
	now := time.Now()
	h := func(d time.Duration) time.Time { return now.Add(-d) }
	return map[string][]ui.ChatMessage{
		"#lobby": {
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
	}
}
