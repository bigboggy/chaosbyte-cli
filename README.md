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
/auth          link your GitHub account
/logout        unlink your GitHub account
/clear         clear scrollback
/help          show all commands
/quit          exit vibespace
```

Aliases: `/exit` / `/bye` → `/quit`, `/part` → `/leave`, `/channels` → `/list`,
`/users` → `/who`, `/?` → `/help`.

`/auth` runs GitHub's [device authorization flow](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps#device-flow):
vibespace shows you a short code, you paste it at https://github.com/login/device,
and on success your SSH public key is linked to your GitHub login. Future
connections from the same key pick up the GitHub handle automatically. Only
the (fingerprint, login) pair is stored — no access tokens are persisted.

When GitHub auth is configured (`VIBESPACE_GH_CLIENT_ID` set), the lobby is
**read-only until you authenticate**: you can see existing messages but can't
send, join channels, or run other commands. The input placeholder nudges you
to `/auth`. After you authenticate, the slash palette swaps `/auth` out for
`/logout`, which unlinks your key from the GitHub identity (next connection
starts as a guest again).

### Enabling `/auth` locally

`/auth` works in local mode too — set `VIBESPACE_GH_CLIENT_ID` to your OAuth
app's client id and run:

```bash
export VIBESPACE_GH_CLIENT_ID=Iv1.xxxxxxxxxxxxxxxx
go run .
```

Without that env var, `/auth` reports it isn't configured and the rest of the
app (chat, profiles, posts, friends) runs against the local SQLite DB without
any GitHub link. The identity store lives next to the DB under
`$XDG_CONFIG_HOME/vibespace/` (macOS: `~/Library/Application Support/vibespace/`).

Local mode has no SSH layer, so the "fingerprint" stored alongside your
GitHub login is synthesized from your OS username (`local:<username>`). It's
stable across runs on the same machine but not portable between machines —
log in on a different laptop and you'll re-run `/auth`.

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
`VIBESPACE_GH_CLIENT_ID`, `VIBESPACE_IDENTITY_PATH`, `VIBESPACE_DATA_PATH`.
Local mode also honors `VIBESPACE_GH_CLIENT_ID` (see above); other env vars
are server-only.

Each screen implements `screens.Screen`. Navigation flows through
`screens.Navigate(target)` messages caught by `internal/app/router.go`.

---

## License

MIT. Do whatever; please don't ship an "AI-powered" fork.
