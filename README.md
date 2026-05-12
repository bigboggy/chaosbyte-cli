```
 ██████╗██╗  ██╗ █████╗  ██████╗ ███████╗██████╗ ██╗   ██╗████████╗███████╗
██╔════╝██║  ██║██╔══██╗██╔═══██╗██╔════╝██╔══██╗╚██╗ ██╔╝╚══██╔══╝██╔════╝
██║     ███████║███████║██║   ██║███████╗██████╔╝ ╚████╔╝    ██║   █████╗
██║     ██╔══██║██╔══██║██║   ██║╚════██║██╔══██╗  ╚██╔╝     ██║   ██╔══╝
╚██████╗██║  ██║██║  ██║╚██████╔╝███████║██████╔╝   ██║      ██║   ███████╗
 ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═════╝    ╚═╝      ╚═╝   ╚══════╝
```

> an all-in-one place for devs and vibe coders, in your terminal.

**chaosbyte** is a TUI lobby app. You start in a 90s-style chat,
then you `/news`, `/spotlight`, `/games`, `/resources`, or `/discussions` your way around.

Built with [bubbletea](https://github.com/charmbracelet/bubbletea) and
[lipgloss](https://github.com/charmbracelet/lipgloss).

---

## Install

One-liner (Linux / macOS):

```bash
curl -fsSL https://raw.githubusercontent.com/bigboggy/chaosbyte-cli/main/install.sh | bash
```

Installs the **chaosbyte** binary to `~/.local/bin/chaosbyte`.

Uninstall:

```bash
curl -fsSL https://raw.githubusercontent.com/bigboggy/chaosbyte-cli/main/install.sh | bash -s -- --uninstall
```

---

## Quick start

```bash
go run .
```

Requires Go 1.24+ and a terminal at least 80×22. Catppuccin Mocha palette throughout.

---

## What's in the lobby

| Area | Slash command | What it is |
|------|---------------|------------|
| **Lobby** | (default) | IRC-style chat with channels, autocompleted slash commands, message history |
| **News** | `/news` | Combined feed: HN, Lobsters, /r/programming, DevHQ, ArsTechnica |
| **Spotlight** | `/spotlight` | One featured project + live discussion. Rotates every 5 minutes. |
| **Resources** | `/resources` (alias `/skills`) | Trending skills, top skills, highlighted GitHub repos, search |
| **Games** | `/games` | Mini-games. **Bug Hunter** is playable, the rest are aspirational. |
| **Discussions** | `/discussions` (alias `/commits`, `/feed`) | Threaded commit-style posts across branches |

The intro animation plays once on startup — `CHAOSBYTE` boots, holds, collapses to a single
byte, then drops you in `#lobby`. Press any key to skip.

---

## Slash commands

Type `/` in the lobby and Tab to cycle suggestions.

```
/news          open news feed
/spotlight     open featured project
/resources     open skills & github repos
/games         open mini-games
/discussions   open commit feed
/join #name    join or switch channel
/leave         return to #lobby
/list          list channels
/who           list users
/topic [text]  view or set topic
/me <action>   third-person action
/clear         clear scrollback
/help          show all commands
/quit          exit chaosbyte
```

Aliases: `/skills` → `/resources`, `/commits` → `/discussions`, `/exit` / `/bye` → `/quit`,
`/part` → `/leave`, `/channels` → `/list`, `/users` → `/who`, `/?` → `/help`.

---

## Keyboard

**Anywhere**
- `ctrl+c` — quit
- `esc` — back to lobby (or pop one sub-mode if you're in a popup)

**Lobby**
- `enter` — send / run slash command
- `tab` / `shift+tab` — cycle autocomplete
- `↑` / `↓` — recall message history
- `pgup` / `pgdn` — scroll the scrollback

**Other screens**
- `j` / `k` — move
- `enter` — open
- See the footer hints — they update per screen

**Discussions** (the original feed)
- `n` — new commit
- `enter` — open post + comments
- `l` — like
- `tab` — next branch · `b` — branch picker
- `r` — reply (inside a post)

---

## Project layout

```
gitstatus/
├── main.go                 # entrypoint, wires app to bubbletea
└── internal/
    ├── theme/              # Catppuccin palette, shared styles, logo
    ├── ui/                 # layout + text + chat helpers
    ├── screens/
    │   ├── screen.go       # Screen interface + Navigate/Flash messages
    │   ├── intro/          # boot animation
    │   ├── lobby/          # chat, slash commands, autocomplete
    │   ├── news/           # combined news feed
    │   ├── resources/      # skills + repos + search
    │   ├── spotlight/      # featured project + live chat
    │   ├── games/          # launcher + bug hunter
    │   └── discussions/    # threaded commit feed
    └── app/                # router, header, footer, top-level View
```

Each feature screen implements `screens.Screen` and never imports another
feature screen. Navigation flows through `screens.Navigate(target)` messages
caught by `internal/app/router.go`. The dependency graph is a star.

---

## License

MIT. Do whatever; please don't ship an "AI-powered" fork.
