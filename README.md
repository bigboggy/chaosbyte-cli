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
/auth github   link a GitHub account (server-side only)
/clear         clear scrollback
/help          show all commands
/quit          exit vibespace
```

Aliases: `/exit` / `/bye` → `/quit`, `/part` → `/leave`, `/channels` → `/list`,
`/users` → `/who`, `/?` → `/help`.

`/auth github` runs GitHub's [device authorization flow](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps#device-flow):
the server shows you a short code, you paste it at https://github.com/login/device,
and on success your SSH public key is linked to your GitHub login. Future
connections from the same key pick up the GitHub handle automatically. Only
the (fingerprint, login) pair is stored — no access tokens are persisted.

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
    ├── hub/                # shared channel/message state (server mode)
    ├── identity/           # SSH fingerprint -> GitHub login store
    ├── github/             # device-flow OAuth client
    ├── auth/               # facade combining identity + github
    ├── screens/
    │   ├── screen.go       # Screen interface + Navigate messages
    │   ├── intro/          # boot animation
    │   └── lobby/          # chat, slash commands, autocomplete, /auth modal
    └── app/                # router, header, footer, top-level View
```

For server mode, `cmd/server/main.go` wires a wish SSH server in front of the
same lobby. Env vars: `VIBESPACE_ADDR`, `VIBESPACE_HOSTKEY`,
`VIBESPACE_MAX_SESS`, `VIBESPACE_GH_CLIENT_ID`, `VIBESPACE_IDENTITY_PATH`.

Each screen implements `screens.Screen`. Navigation flows through
`screens.Navigate(target)` messages caught by `internal/app/router.go`.

---

## License

MIT. Do whatever; please don't ship an "AI-powered" fork.
