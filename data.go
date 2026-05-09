package main

import "time"

type Comment struct {
	Author string
	Body   string
	At     time.Time
}

type Commit struct {
	SHA      string
	Author   string
	Message  string
	At       time.Time
	Likes    int
	Liked    bool
	Comments []Comment
}

type Branch struct {
	Name    string
	Commits []Commit
}

func seedBranches() []Branch {
	now := time.Now()
	h := func(d time.Duration) time.Time { return now.Add(-d) }

	return []Branch{
		{
			Name: "main",
			Commits: []Commit{
				{
					SHA: "a1b2c3d", Author: "alice", At: h(2 * time.Hour),
					Message: "Add user authentication flow",
					Likes:   12,
					Comments: []Comment{
						{Author: "bob", Body: "Nice, finally!", At: h(90 * time.Minute)},
						{Author: "carol", Body: "What about SSO?", At: h(80 * time.Minute)},
						{Author: "alice", Body: "SSO comes next sprint.", At: h(70 * time.Minute)},
					},
				},
				{
					SHA: "d4e5f6g", Author: "bob", At: h(5 * time.Hour),
					Message: "Fix database connection pool exhaustion",
					Likes:   5,
					Comments: []Comment{
						{Author: "dave", Body: "Saved my pager today.", At: h(4 * time.Hour)},
					},
				},
				{
					SHA: "9988aa1", Author: "carol", At: h(26 * time.Hour),
					Message: "Bump dependencies and tidy go.mod",
					Likes:   2,
				},
			},
		},
		{
			Name: "feature/auth",
			Commits: []Commit{
				{
					SHA: "ffe1023", Author: "alice", At: h(45 * time.Minute),
					Message: "WIP: OAuth callback handling",
					Likes:   3,
					Comments: []Comment{
						{Author: "bob", Body: "Need PKCE here.", At: h(30 * time.Minute)},
					},
				},
				{
					SHA: "ccd2211", Author: "alice", At: h(3 * time.Hour),
					Message: "Sketch login screen",
					Likes:   1,
				},
			},
		},
		{
			Name: "bugfix/123-null-pointer",
			Commits: []Commit{
				{
					SHA: "7e7e7e7", Author: "dave", At: h(20 * time.Minute),
					Message: "Guard nil session in middleware",
					Likes:   8,
					Comments: []Comment{
						{Author: "alice", Body: "LGTM, ship it.", At: h(15 * time.Minute)},
					},
				},
			},
		},
		{
			Name: "refactor/db-layer",
			Commits: []Commit{
				{
					SHA: "111aaaa", Author: "carol", At: h(50 * time.Hour),
					Message: "Extract repository interfaces",
					Likes:   4,
				},
				{
					SHA: "222bbbb", Author: "carol", At: h(74 * time.Hour),
					Message: "Move query builders to internal/db",
					Likes:   1,
				},
			},
		},
	}
}
