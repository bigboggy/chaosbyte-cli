package games

type Game struct {
	Name        string
	Description string
	Playable    bool
}

func seedGames() []Game {
	return []Game{
		{"bug hunter", "guess which line the bug is on (1-100). lower attempts = more dignity.", true},
		{"sha sprint", "memorize a 7-char SHA in 3s, then type it back. coming soon.", false},
		{"vibe roulette", "spin the wheel of vibes. land on 'ship it' or 'rewrite in rust'. coming soon.", false},
		{"rubber duck", "explain your bug to a duck. the duck has opinions. coming soon.", false},
		{"git blame bingo", "fill the card with classic blame quotes. coming soon.", false},
	}
}
