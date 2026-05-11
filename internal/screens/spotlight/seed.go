package spotlight

import (
	"time"

	"github.com/bchayka/gitstatus/internal/ui"
)

type Spotlight struct {
	Project     string
	Author      string
	RepoURL     string
	Description string
	Stars       int
	Language    string
	Highlights  []string
}

func seedSpotlights() []Spotlight {
	return []Spotlight{
		{
			Project:     "lazygit",
			Author:      "@jesseduffield",
			RepoURL:     "https://github.com/jesseduffield/lazygit",
			Description: "simple terminal UI for git commands. the thing your senior engineer secretly uses.",
			Stars:       54820,
			Language:    "Go",
			Highlights: []string{
				"fully keyboard driven git, no mouse no problem",
				"stage, commit, push, pull, rebase, cherry-pick from one screen",
				"the answer to 'why am i typing all this git out by hand'",
			},
		},
		{
			Project:     "atuin",
			Author:      "@ellie",
			RepoURL:     "https://github.com/atuinsh/atuin",
			Description: "magical shell history. you will weep when you realize what you've been missing.",
			Stars:       21102,
			Language:    "Rust",
			Highlights: []string{
				"searchable, syncable, encrypted shell history across machines",
				"fzf-style fuzzy search over every command you've ever run",
				"finally, a reason to be proud of your bash history",
			},
		},
		{
			Project:     "zellij",
			Author:      "@aram",
			RepoURL:     "https://github.com/zellij-org/zellij",
			Description: "a terminal workspace with batteries included. tmux had a kid, the kid is opinionated.",
			Stars:       24410,
			Language:    "Rust",
			Highlights: []string{
				"layouts, panes, tabs, plugins — all configurable in kdl",
				"the UI tells you the keybinds, so you won't print a cheat sheet",
				"floating windows in your terminal. yes really.",
			},
		},
		{
			Project:     "helix",
			Author:      "@helix-editor",
			RepoURL:     "https://github.com/helix-editor/helix",
			Description: "post-modern modal text editor. it's vim if vim had therapy.",
			Stars:       38201,
			Language:    "Rust",
			Highlights: []string{
				"selection → action grammar, opposite of vim's verb → motion",
				"multi-cursor first class, no plugin gymnastics",
				"LSP and tree-sitter built in, you do nothing to get smart features",
			},
		},
		{
			Project:     "uv",
			Author:      "@astral-sh",
			RepoURL:     "https://github.com/astral-sh/uv",
			Description: "the python package manager you wanted in 2014, finally arrived in 2025.",
			Stars:       34102,
			Language:    "Rust",
			Highlights: []string{
				"10-100x faster than pip, written in rust because of course",
				"replaces pip, pip-tools, pipx, poetry, pyenv, virtualenv — in one tool",
				"your python environment is now actually reproducible, somehow",
			},
		},
	}
}

func seedChat() []ui.ChatMessage {
	now := time.Now()
	h := func(d time.Duration) time.Time { return now.Add(-d) }
	return []ui.ChatMessage{
		{Author: "@yamlhater", Body: "wait this is what i've been needing the whole time", At: h(4 * time.Minute), Kind: ui.ChatNormal},
		{Author: "@nullpointer", Body: "the demo gif alone sold me, send help i'm installing it now", At: h(3 * time.Minute), Kind: ui.ChatNormal},
		{Author: "@vibe_master", Body: "imagine using this for one (1) week and writing a medium post about it", At: h(150 * time.Second), Kind: ui.ChatNormal},
		{Author: "@devops_bard", Body: "i tried this, then i tried to uninstall it. could not.", At: h(70 * time.Second), Kind: ui.ChatNormal},
		{Author: "@junior_dev", Body: "is this the one where you press a key and it just works? or is this the OTHER one", At: h(40 * time.Second), Kind: ui.ChatNormal},
		{Author: "@standup_ghost", Body: "they're all the one where you press a key and it just works", At: h(20 * time.Second), Kind: ui.ChatNormal},
	}
}
