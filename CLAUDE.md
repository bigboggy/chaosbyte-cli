# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

**vibespace** — a TUI IRC-style chat lobby for devs, built with bubbletea / lipgloss / wish. Ships in two modes from one codebase:

- **Local** (`./main.go`): single-process, single-user, in-memory hub. Run via `go run .`.
- **SSH server** (`./cmd/server/main.go`): a wish SSH server fronting one shared hub. Each session gets its own `app.App` but reads/writes the same `*hub.Hub`.

Go 1.24+. No test suite, no linter config — `go vet ./...` and `go build ./...` are the only verifiers in tree.

> Heads-up: `AGENTS.md` is stale (describes a previous project called "gitstatus"). Ignore it; this file supersedes it.

## Common commands

```bash
go run .                   # run local mode
go run ./cmd/server        # run SSH server on :2222 (needs VIBESPACE_GH_CLIENT_ID for /auth)
go build ./...             # verify everything compiles
go vet ./...               # only available lint
scripts/deploy.sh          # cross-compile linux/amd64, scp + systemctl restart on vibespace.sh
scripts/release.sh vX.Y.Z  # tag, build 4 platforms, gh release create
```

The Go module path is `github.com/bigboggy/vibespace`. All internal imports go through this prefix.

## Architecture

### Two entrypoints, one app

`main.go` and `cmd/server/main.go` both build the same `app.App` (`internal/app/app.go`). The difference is what they pass in:

- Local mode wires one `app.App` to a fresh `hub.New()` with no auth service. Identity is `@<os-user>`.
- Server mode (`cmd/server/main.go`) creates one `*hub.Hub` and one `*auth.Service` at startup, then per-SSH-session builds an `app.App` against the shared hub. Identity comes from the SSH pubkey fingerprint (resolved via `auth.Service.Lookup`) or a sanitized `sess.User()`.

### Hub is the only mutable chat state

`internal/hub/hub.go` owns channels, messages, and presence. Sessions read state during `View` and on every `hub.Event` they receive via `Subscribe()`. Sessions never hold their own copy of chat — they own only session-local UI state (input, scroll, history, active channel, identity). This is why server-mode sessions stay consistent without locking on the session side.

`hub.Event` implements `tea.Msg`, so events flow straight through bubbletea. Subscribers re-read the hub on each event rather than trusting the event payload — `broadcast` drops on full channel buffer, which is safe because of this re-read pattern.

### Screen interface + Navigate messages

`internal/screens/screen.go` defines the `Screen` interface (`Init/Update/View/Name/Title/HeaderContext/Footer/InputFocused`). Screens **never import each other**. Cross-screen flow happens by emitting `screens.NavigateMsg{Target: ...}` (via `screens.Navigate(target)`), which `internal/app/router.go` catches and dispatches. The dependency graph is a star: `app` at the center, screens as leaves.

Two screens currently exist:
- `screens/intro` — boot animation, emits `Navigate(LobbyID)` when done
- `screens/lobby` — chat, slash commands, autocomplete, `/auth` modal, `/theme` picker

### App router

`internal/app/router.go` handles three concerns:
1. `tea.WindowSizeMsg` is broadcast to **every** screen (not just the active one) so cached layouts in inactive screens stay correct.
2. `hub.Event` is always routed to the lobby (the screen that owns the subscription), regardless of which screen is currently visible — otherwise events arriving during the intro would be swallowed.
3. Key handling: if the active screen's `InputFocused()` is true, the router forwards every key without interception. Otherwise it applies global bindings (`esc`/`q` → lobby, `ctrl+c` → quit).

### Identity and auth gating

`internal/auth` is a thin facade over `internal/github` (device flow) and `internal/identity` (a JSON file mapping SSH fingerprint → GitHub login). The lobby treats `auth.Service` as optional — pass `nil` and `/auth` disappears.

When `auth != nil` and the session is unauthenticated, the lobby is **gated**: only commands in `allowedWhenGated` (in `internal/screens/lobby/commands.go`) work. Everything else returns a "type /auth" hint. After successful auth, `meUser` flips to `@<ghLogin>`; subsequent connections from the same SSH key skip auth via `authSvc.Lookup(fingerprint)` at session start.

The identity store only persists `(fingerprint, login)` — **never access tokens**.

### Theme system

`internal/theme` holds a registry of palettes (`tokyonight` is default — see `theme.DefaultID`) and a `*Styles` value that pairs a theme with a per-session `lipgloss.Renderer`. Server-mode builds a fresh renderer per SSH session (`bm.MakeRenderer(sess)`) so truecolor/256/16-color clients all get appropriate downgrade. `/theme <id>` mutates `*Styles` in place — every subsequent render across all screens picks up the new theme.

### Slash commands

Defined in `internal/screens/lobby/commands.go`:
- `builtins` slice = canonical commands in autocomplete order
- `aliases` map = alternate names (e.g. `/exit` → `/quit`)
- `allowedWhenGated` map = whitelist for unauthenticated server sessions
- `/auth` and `/logout` are mutually exclusive — palette hides whichever doesn't match current auth state

## Server config (env vars)

| Var | Default | Purpose |
|-----|---------|---------|
| `VIBESPACE_ADDR` | `:2222` | listen addr (use non-22 unless OpenSSH moved) |
| `VIBESPACE_HOSTKEY` | `.ssh/id_ed25519` | SSH host key path (auto-generated) |
| `VIBESPACE_GH_CLIENT_ID` | unset | enables `/auth`; when set, lobby is gated until auth |
| `VIBESPACE_IDENTITY_PATH` | `./identities.json` | fingerprint → GH login store |

Without `VIBESPACE_GH_CLIENT_ID`, the server runs but `/auth` is disabled and no gating is applied.

## Conventions

- Min terminal: 80×22 (`ui.MinWidth` / `ui.MinHeight`). Below that, the app renders a "too small" message.
- All chat state mutations go through `hub` methods. Don't mutate channel/message slices from outside the hub.
- Per-session resources (hub subscription) are released via `App.Cleanup()` — called from the SSH context-done goroutine in `cmd/server/main.go`.
- No tests exist. If adding them, target `hub` (concurrent subscribe/broadcast), `identity` (file IO), and `lobby` command parsing.
