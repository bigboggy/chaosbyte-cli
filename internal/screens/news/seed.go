package news

import "time"

type Item struct {
	Source   string // HN, Lobsters, /r/programming, DevHQ, ArsTechnica
	Title    string
	URL      string
	Author   string
	Score    int
	Comments int
	At       time.Time
}

func seedItems() []Item {
	now := time.Now()
	h := func(d time.Duration) time.Time { return now.Add(-d) }
	return []Item{
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
