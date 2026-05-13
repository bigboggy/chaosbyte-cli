```
 ██████╗██╗  ██╗ █████╗  ██████╗ ███████╗██████╗ ██╗   ██╗████████╗███████╗
██╔════╝██║  ██║██╔══██╗██╔═══██╗██╔════╝██╔══██╗╚██╗ ██╔╝╚══██╔══╝██╔════╝
██║     ███████║███████║██║   ██║███████╗██████╔╝ ╚████╔╝    ██║   █████╗
██║     ██╔══██║██╔══██║██║   ██║╚════██║██╔══██╗  ╚██╔╝     ██║   ██╔══╝
╚██████╗██║  ██║██║  ██║╚██████╔╝███████║██████╔╝   ██║      ██║   ███████╗
 ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═════╝    ╚═╝      ╚═╝   ╚══════╝
```

> chaosbyte presents **vibespace**: a TUI chatroom served over SSH.

`chaosbyte` is the studio; `vibespace` is the product. The flagship room runs on **vibespace.sh**. Each connection spawns a per-session bubbletea program, and the in-process broker fans chat out to everyone connected to the same team room.

Built with [bubbletea](https://github.com/charmbracelet/bubbletea), [lipgloss](https://github.com/charmbracelet/lipgloss), and [wish](https://github.com/charmbracelet/wish).

---

## Install

One-liner (Linux / macOS):

```bash
curl -fsSL https://raw.githubusercontent.com/bigboggy/vibespace-cli/main/install.sh | bash
```

Installs the **vibespace** binary to `~/.local/bin/vibespace`.

To uninstall:

```bash
bash install.sh --uninstall
```

---

## Quick start

Local single-user, no SSH, flagship config baked in:

```bash
go run ./cmd/vibespace
```

Stand up the SSH host:

```bash
go run ./cmd/vibespace-server --port 23234
```

Then from another shell:

```bash
ssh -p 23234 vibespace@localhost     # flagship room
ssh -p 23234 acme@localhost          # acme tenant
```

The SSH user is the team slug. Two clients connected with the same slug share a room.

---

## What's in the room

The chaosbyte splash plays once on connect and resolves into the **vibespace lobby**. The lobby is the room: chat, the moderator's voice, a per-line typo engine that animates arrivals, and a value-noise field engine behind the surface.

The other surface in v0.3 is **spotlight** (`/spotlight`), which highlights one project at a time and is reached from the lobby.

A **blitz** (`/blitz`) is a thirty-second cascade-race round that plays inside the chat itself. The moderator cascades a target word in, players race to type it, the first three unique matchers score 3 / 2 / 1, and the winner's nick cascade-settles at the end. The round paints existing chat with a per-row wave offset; there is no separate game grid.

---

## Slash commands

```
/spotlight     open the current spotlit project
/blitz         start a thirty-second cascade-race round
/themes        list color themes, or /themes <name> to switch
/me <text>     third-person action
/who           list who is here
/clear         clear scrollback
/help          show all commands
/leave         leave the room
/quit          exit vibespace
```

Aliases: `/exit` and `/bye` map to `/quit`, `/users` maps to `/who`, `/?` maps to `/help`.

---

## Keyboard

- `ctrl+c` quits
- `esc` returns to the lobby from any sub-screen, or dismisses a popup
- `enter` sends a message or runs a slash command
- `tab` and `shift+tab` cycle autocomplete
- `↑` and `↓` recall message history
- `pgup` and `pgdn` scroll the scrollback
- Per-screen hints update in the footer

---

## Themes

Two palettes ship by default. `/themes` switches between them inside the room without disconnecting.

| Name | Look |
|------|------|
| `boggy` | Tokyo-night-style dark: navy ground, light cool foreground, bright blue and purple accents. The flagship default. |
| `workshop` | Parchment cream on near-black, muted phosphor green for OK marks, muted gold for the moderator. |

Adding a third theme is a single entry in `internal/theme.Themes`. Team configs can also override their default palette inline in the team's `.toml`.

---

## Project layout

```
vibespace-cli/
├── main.go                          # tiny entrypoint shim
├── cmd/
│   ├── vibespace/                   # local single-user dev binary
│   ├── vibespace-server/            # Wish SSH daemon (the deployable)
│   └── vibespace-trace/             # headless event-loop tracer
└── internal/
    ├── theme/                       # palettes (boggy, workshop), logo, styles
    ├── ui/                          # layout + chat message types
    ├── config/                      # RoomConfig + .toml loader
    ├── platform/                    # team registry, slug to (config, broker)
    ├── room/                        # shared chat broker, per-user nicks
    ├── field/                       # value-noise bitmap field engine
    ├── typo/                        # Pretext-style content engine for chat
    ├── mod/                         # moderator event surface
    ├── games/                       # in-chat blitz round
    ├── app/                         # router, header, footer, top-level View
    └── screens/
        ├── screen.go                # Screen interface + Navigate messages
        ├── intro/                   # chaosbyte studio splash
        ├── lobby/                   # the vibespace room
        └── spotlight/               # featured-project surface
```

Each feature screen implements `screens.Screen` and never imports another feature screen. Navigation flows through `screens.Navigate(target)` messages caught by `internal/app/router.go`. The dependency graph is a star.

---

## License

MIT. Do whatever; please don't ship an "AI-powered" fork.
