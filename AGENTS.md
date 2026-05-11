# AGENTS.md

## Project

**gitstatus** — Go TUI (bubbletea) mock git branch/commit social feed. All code is root-level `package main`. No real git integration; data is seeded fake.

## Commands

```
go build          # binary = gitstatus
go run .          # run
```

No test framework, no lint/config, no README.

## File roles

| File | Purpose |
|------|---------|
| `main.go` | entrypoint, `tea.NewProgram` |
| `model.go` | `model` struct, `Update()`, key handling, mode/focus state |
| `view.go` | `View()`, all render functions, layout math |
| `data.go` | `Branch`/`Commit`/`Comment` types, `seedBranches()` |
| `styles.go` | Catppuccin Mocha colors, `lipgloss` style vars, `paneStyle()` |

## Conventions

- Focus cycles `branches → feed → input` via Tab/Shift-Tab; `i` jumps to input, `Esc` exits input.
- `j`/`k` navigate branches or commits within feed. `l` likes a commit. `c` opens comment mode.
- Commit input: type a message, Enter to push. Esc to cancel.
- All colors are Catppuccin Mocha palette (`#1a1b26` bg).
- `minWidth=80`, `minHeight=22` — terminal too small returns an error-style message.
- Branch pane is fixed at 28 columns wide.
- No tests exist. Adding them should target `model.go` Update logic and `view.go` layout helpers.
