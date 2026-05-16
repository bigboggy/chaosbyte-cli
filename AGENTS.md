# AGENTS.md

## Project

**vibespace** — IRC-style TUI chat for devs, built with bubbletea + lipgloss. Two binaries:

| Binary | How to run | Purpose |
|--------|-----------|---------|
| `vibespace` (local) | `go run .` | Single-user TUI, one hub per session |
| `vibespace-server` | `go run ./cmd/server` | Multi-user SSH chat server |

Requires **Go 1.24+**. No test framework, no lint config.

## Architecture

```
main.go                    → tea.NewProgram(app.New(...), tea.WithAltScreen())
internal/app/app.go        → top-level bubbletea Model (intro + lobby screens)
internal/app/router.go     → Update() — global key handling, screen nav, hub event routing
internal/app/header.go     → renderFrame, renderHeader, renderFooter, tooSmall
internal/screens/screen.go → Screen interface (Init/Update/View + Title/Context/Footer/InputFocused)
internal/screens/lobby/    → chat screen: lobby.go + commands.go + completion.go + auth.go + themepicker.go
internal/screens/intro/    → boot animation (intro.go + animation.go)
internal/hub/hub.go        → shared mutable chat state (channels, messages, subscribers)
internal/auth/auth.go      → facade: wires github/device.go + identity/ for /auth
internal/github/device.go  → RFC 8628 device flow: Start → Poll → UserLogin (token discarded)
internal/identity/identity.go → on-disk JSON fingerprint→ghLogin store (atomic writes)
internal/theme/            → 5 themes (tokyonight default, catppuccin, dracula, gruvbox, nord)
internal/ui/               → layout + chat rendering + text helpers
```

**Screen architecture:** screens never import each other. They communicate by emitting messages (`screens.NavigateMsg`, `hub.Event`) that the app router catches. Star dependency graph with `app` at center.

**Identity model:** `fallbackUser` (SSH-derived nick) → `meUser` (active display, tracks `ghLogin` when set). In server mode, if `VIBESPACE_GH_CLIENT_ID` is set and user hasn't authenticated, the lobby is **read-only gated** — only `/auth`, `/help`, `/quit`, `/clear`, `/theme` work.

## Commands

```bash
go build           # binary = vibespace (or gitstatus — go.mod module name hasn't been updated)
go run .           # local TUI
go run ./cmd/server # SSH server
```

### Release

```bash
scripts/release.sh v0.4.0 [--draft] [--notes "..."]
```

Builds darwin/amd64, darwin/arm64, linux/amd64, linux/arm64. Requires `gh` CLI authenticated. Checks dirty tree and main branch.

### Deploy server

```bash
scripts/deploy.sh
```

Builds linux/amd64 from `./cmd/server`, SCPs to `root@vibespace.sh:22022`, `systemctl restart vibespace`.

### Install (end-user)

```bash
curl -fsSL https://raw.githubusercontent.com/bigboggy/vibespace/main/install.sh | bash
# or: bash install.sh --uninstall
```

## Server env vars

| Var | Default | Purpose |
|-----|---------|---------|
| `VIBESPACE_ADDR` | `:2222` | Listen address |
| `VIBESPACE_HOSTKEY` | `.ssh/id_ed25519` | SSH host key path (auto-generated) |
| `VIBESPACE_GH_CLIENT_ID` | — | Enables `/auth` GitHub device flow |
| `VIBESPACE_IDENTITY_PATH` | `./identities.json` | On-disk fingerprint→login store |

## Key conventions

- **Screen interface** (`internal/screens/screen.go`): `Init() tea.Cmd`, `Update(msg) (Screen, tea.Cmd)`, `View(w,h) string`, `Name()`, `Title()`, `HeaderContext()`, `Footer()`, `InputFocused()`. Screens are **value types** — `Update` returns a new screen that replaces the old one in the app's map.
- **Minimum terminal**: 40×10 (`ui.MinWidth`/`ui.MinHeight`). Below that, shows "terminal too small" message. README says 80×22 for the intended experience.
- **Theme**: runtime-swappable via `/theme` or `/theme <id>`. `theme.Styles` embeds `*lipgloss.Renderer` + `Theme` — swapping `Theme` at runtime updates all screens on next render. Per-session renderer handles color depth downgrade.
- **Hub broadcast**: drops on full buffer (non-fatal — subscribers re-read state on every event). 16-event buffer per subscriber.
- **Chat scrollback**: `windowScrollback()` clamps offset and pads to exact height so input stays anchored.
- **Slash commands**: autocomplete palette (`palettePageSize=10`, `commandColWidth=10`). `/auth` and `/logout` are mutually exclusive in the palette based on auth state. Aliases defined in `commands.go`.
- **Intro animation**: 4.6s total, 7 phases (boot → build → hold → shrink → byte → block → fade). Any key skips.
- **Identity store**: flat JSON, atomic write via temp+rename. No access tokens persisted — only (fingerprint, login) mapping.
- `.gitignore` excludes: `.claude`, `.idea`, `.ssh`, `dist`.

## Adding a new screen

1. Create `internal/screens/<name>/` with `name.go` (implements `screens.Screen`).
2. Register in `app.New()` screen map: `screens.<NameID>: <name>.New(...)`.
3. Add `screens.<NameID> = "<name>"` const to `screens/screen.go`.
4. The intro screen can emit `screens.Navigate(screens.LobbyID)` to return.
5. Screens with text input must return `true` from `InputFocused()` to prevent global key interception (e.g. `q` quitting mid-typing).

## Notable gotchas

- `go.mod` module path is `github.com/bchayka/gitstatus` — **not** `vibespace`. The binary name is vibespace but the module name hasn't been updated. All imports use `github.com/bchayka/gitstatus/...`.
- `theme.Styles` is **mutable** — `/theme` swaps the embedded `Theme`. All screens share the same `*Styles` pointer, so the change is instant.
- Hub events arriving during intro still route to lobby (router.go:33-40) — without this, the intro would swallow events and the subscription would stall.
- `cmd/server/main.go` uses `wish` bubbletea middleware. Per-session `tea.WithAltScreen()` is set in the handler.
- `identity.Store.Unlink` and `Link` clone the entire map before writing — fine for small stores, not designed for scale.
- `install.sh` downloads from `bigboggy/vibespace` releases, not `bchayka/vibespace`. Verify the repo URL matches your target before using.
