package hub

import (
	"time"

	"github.com/bchayka/gitstatus/internal/ui"
)

// seed returns the initial set of channels. The lobby is always index 0 —
// sessions start there. Only the lobby gets a MOTD; everywhere else starts
// empty.
func seed() []*Channel {
	return []*Channel{
		{
			Name: "#lobby",
			Messages: []ui.ChatMessage{
				{Author: "server", Body: "MOTD: welcome to vibespace — type /help to see what you can do", At: time.Now(), Kind: ui.ChatSystem},
			},
		},
		{Name: "#general"},
		{Name: "#vibe-coding"},
		{Name: "#rust-anonymous"},
		{Name: "#help"},
		{Name: "#side-projects"},
		{Name: "#offtopic"},
	}
}
