```
██╗   ██╗ ██╗ ██████╗  ███████╗ ███████╗ ██████╗   █████╗   ██████╗ ███████╗
██║   ██║ ██║ ██╔══██╗ ██╔════╝ ██╔════╝ ██╔══██╗ ██╔══██╗ ██╔════╝ ██╔════╝
██║   ██║ ██║ ██████╔╝ █████╗   ███████╗ ██████╔╝ ███████║ ██║      █████╗
╚██╗ ██╔╝ ██║ ██╔══██╗ ██╔══╝   ╚════██║ ██╔═══╝  ██╔══██║ ██║      ██╔══╝
 ╚████╔╝  ██║ ██████╔╝ ███████╗ ███████║ ██║      ██║  ██║ ╚██████╗ ███████╗
  ╚═══╝   ╚═╝ ╚═════╝  ╚══════╝ ╚══════╝ ╚═╝      ╚═╝  ╚═╝  ╚═════╝ ╚══════╝
```

> a 90s-style chat lobby for devs and vibe coders, in your terminal.

**vibespace** is an IRC-style TUI chat: channels, slash commands, autocomplete,
message history.

Built with [bubbletea](https://github.com/charmbracelet/bubbletea) and
[lipgloss](https://github.com/charmbracelet/lipgloss).

---

## Install

One-liner (Linux / macOS):

```bash
curl -fsSL https://raw.githubusercontent.com/bigboggy/vibespace-cli/main/install.sh | bash
```

Installs the **vibespace** binary to `~/.local/bin/vibespace`.

Uninstall:

```bash
curl -fsSL https://raw.githubusercontent.com/bigboggy/vibespace-cli/main/install.sh | bash -s -- --uninstall
```

---

## Quick start

```bash
go run .
```

Requires Go 1.24+ and a terminal at least 80×22. Catppuccin Mocha palette throughout.

The intro animation plays once on startup — `VIBESPACE` boots, holds, collapses to a single
character, then drops you in `#lobby`. Press any key to skip.

---

## Slash commands

Type `/` and Tab to cycle suggestions.

```
/join #name    join or switch channel
/leave         return to #lobby
/list          list channels
/who           list users
/me <action>   third-person action
/clear         clear scrollback
/help          show all commands
/quit          exit vibespace
```

Aliases: `/exit` / `/bye` → `/quit`, `/part` → `/leave`, `/channels` → `/list`,
`/users` → `/who`, `/?` → `/help`.

---

## Keyboard

- `enter` — send / run slash command
- `tab` / `shift+tab` — cycle autocomplete
- `↑` / `↓` — recall message history (or move palette selection when open)
- `pgup` / `pgdn` — scroll the scrollback
- `esc` — clear input / dismiss palette
- `ctrl+c` — quit

---

## Project layout

```
vibespace/
├── main.go                 # entrypoint, wires app to bubbletea
└── internal/
    ├── theme/              # Catppuccin palette, shared styles, logo
    ├── ui/                 # layout + text + chat helpers
    ├── screens/
    │   ├── screen.go       # Screen interface + Navigate messages
    │   ├── intro/          # boot animation
    │   └── lobby/          # chat, slash commands, autocomplete
    └── app/                # router, header, footer, top-level View
```

Each screen implements `screens.Screen`. Navigation flows through
`screens.Navigate(target)` messages caught by `internal/app/router.go`.

---

## License

MIT. Do whatever; please don't ship an "AI-powered" fork.
