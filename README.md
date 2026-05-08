# lazyrun

Interactive terminal UI for **GitHub Actions workflow runs**, inspired by [lazygit](https://github.com/jesseduffield/lazygit) but for **`gh run`**.  

---

## Prerequisites

| Requirement | Purpose |
|-------------|---------|
| **Go ≥ 1.22** | Build lazyrun |
| [**GitHub CLI `gh`**](https://cli.github.com/) | Authenticated REST access via `gh auth token` |

Run `gh auth login` once; lazyrun uses the resolved token automatically.

---

## Quick start

```bash
cd /path/to/lazyrun
go build -o lazyrun ./cmd/lazyrun

# Repo from git remote `origin`
./lazyrun

# Or pin a repo explicitly
./lazyrun owner/repo
```

---

## Full feature checklist

Every capability below ships in this Bubble Tea rewrite.

### Repository & authentication

| Feature | Behavior |
|---------|----------|
| `lazyrun` — auto-resolve `owner/repo` | Parses `git remote get-url origin` (GitHub SSH/HTTPS) |
| `lazyrun owner/repo` — explicit slug | Optional `.git` suffix stripped |
| `gh auth token` | Bearer token fetched once via GitHub CLI (required) |

### Main layout — three panels

| Feature | Notes |
|---------|-------|
| **Branches** pane | Loads `/repos/{o}/{r}/branches` (pagination via `rel=next`) |
| Branch markers | **`○`** default branch • **`🔒`** protected branches |
| **Runs** pane | Lists Actions runs filtered by selected branch (`/actions/runs?branch=` …) |
| Run glyphs | Icons for queued / running / successes / failures / skips / cancellations / timeouts … |
| **Details** pane | Selected run headline (event, actor, URLs, statuses) plus **jobs** and nested **steps** with durations |

### Focus & navigation

| Keys | Behavior |
|------|-----------|
| `Tab` | Cycle pane focus → Branches → Runs → Details (wraps) |
| `1` / `2` / `3` | Jump to pane (same order as `Tab`) |
| `↑` `↓` or `j` `k` | Move selection inside focused list |
| `g` | Jump **first** item (Branches or Runs depending on pane) |
| `G` | Jump **last** item (Branches or Runs) |
| `Enter` | Runs/Details pane → open aggregated logs (see Logs view) |

### Logs view

| Keys | Behavior |
|------|-----------|
| `l` (`L` unaffected) when Runs or Details focused | Same as Enter — aggregated job logs |
| `↑↓` `jk` scroll | Wrapped **viewport** of combined job logs |
| `PgUp` / `PgDn` | Faster scroll |
| `Esc` | Return to dashboard (clears in-memory logs) |

### Refresh & actions on runs

| Keys | Behavior |
|------|-----------|
| `r` | Refresh branch runs + rerun job metadata for highlighted run |
| `c` | Confirm dialog → **cancel** `/actions/runs/{id}/cancel` |
| `R` | Confirm dialog → `/actions/runs/{id}/rerun` |
| `F` | Confirm dialog → `/actions/runs/{id}/rerun-failed-jobs` (only when failing conclusions) |

### Workflow dispatch

| Keys | Behavior |
|------|-----------|
| `W` | Fetch workflows → pick **first** active YAML workflow → **`workflow_dispatch`** on **currently highlighted branch** (matches prototype behavior) |

### Clipboard integrations

| Keys | Behavior |
|------|-----------|
| `y` | Copy run **browser URL**: tries **OSC 52**, then **`pbcopy`** / **`xclip -selection clipboard`** / **`wl-copy`** |
| `Y` | Copy aggregated logs (requires logs fetched). Strips `{}` tokens like the JS UI |

### Overlays & system keys

| Keys | Behavior |
|------|-----------|
| `?` | Toggle **Help** cheat-sheet overlay |
| `Esc` | Dismiss Help, cancel confirm dialogs, exit Logs view sequence |
| `q` **or** `Ctrl+C` | Quit Bubble Tea (AltScreen teardown) |

### Background polling & status line

| Feature | Timing |
|---------|-------|
| Default poll cadence | **20 s** on active branch slice |
| **LIVE badge** acceleration | Shrinks poll to **3 s** whenever any fetched run lists `queued` / `in_progress` / `pending` / `waiting` / `requested` statuses |
| Status bar footer | Pane legend + ephemeral API / clipboard / clipboard status messages |

---

## Project layout

```
cmd/lazyrun/main.go        # Bubble Tea bootstrap + CLI parsing
internal/github/actions.go # Actions REST helpers + pagination
internal/clipboard/copy.go # Clipboard fallbacks / OSC‑52
internal/repo/detect.go   # Infer owner/repo via git CLI
internal/tui/*.go           # Panels, viewport, dialogs, polling
```

---

## Cross-compilation

Go produces static-ish binaries trivially:

```bash
GOOS=linux GOARCH=amd64 go build -trimpath -o lazyrun-linux-amd64 ./cmd/lazyrun
GOOS=darwin GOARCH=arm64 go build -trimpath -o lazyrun-darwin-arm64 ./cmd/lazyrun
```
