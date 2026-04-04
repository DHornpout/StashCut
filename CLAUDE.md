# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

StashCut is a keyboard shortcut manager. This repo contains:
- **`CLI/`** — Go + Bubble Tea TUI app (active development)
- **`WebApp/`** — Web app (not yet implemented; only Playwright test scaffolding exists)
- **`specs/`** — Product and technical specifications

## CLI Commands

All commands run from `CLI/`:

```bash
# Run in development
go run .

# Build binary (output goes to bin/)
go build -o bin/stashcut .

# Release builds
GOOS=darwin GOARCH=arm64 go build -o bin/stashcut-macos-arm64 .
GOOS=darwin GOARCH=amd64 go build -o bin/stashcut-macos-amd64 .

# Install to PATH
go install .

# Run tests
go test ./...

# Run a single test
go test ./store/ -run TestMerge

# Format
gofmt -w .

# Lint (if golangci-lint installed)
golangci-lint run
```

Optional dev tools: `golangci-lint` (`brew install golangci-lint`), `air` for live reload (`go install github.com/air-verse/air@latest`).

## Architecture

The CLI follows the [Bubble Tea](https://github.com/charmbracelet/bubbletea) Model-Update-View pattern. Data flows: `store` loads JSON → `model` holds structs → `ui` renders and mutates → `store` saves JSON.

### Layers

**`model/data.go`** — Go structs mapping 1:1 to `shortcuts.json`. The schema is nested: `ShortcutFile → App → Group → Shortcut`. Each `Shortcut` has `KeysByOS map[string]KeysForOS` for cross-platform keys (keys stored normalized, e.g. `cmd+shift+t`; displayed as `⌘⇧T`).

**`store/store.go`** — Load/save JSON, create new files, and `Merge()` (last-write-wins on `updated_at`). Handles `~` path expansion and directory creation.

**`config/config.go`** — Reads/writes `~/.config/stashcut/config.json` which stores the active data file path. Default data path: `~/Library/Application Support/Stashcut/shortcuts.json`.

**`keybindings/keys.go`** — All key mappings in one place. Reference here before adding new bindings.

**`ui/`** — Bubble Tea components:
- `app.go` — Root model; owns focus state, orchestrates all panels, handles saves
- `sidebar.go` — Left panel: scrollable app list
- `shortcutlist.go` — Right panel: shortcuts for selected app, multi-column layout, favorites float to top
- `form.go` — Add/edit forms for both apps and shortcuts; Tab navigates between fields
- `search.go` — `/` activates search, filters across all apps simultaneously
- `firstrun.go` — Shown when no data file found; prompts to create or open a file
- `styles.go` — All Lip Gloss styles and color constants (purple/gray theme)

### Startup Flow

`main.go` parses `--file` flag → loads config → attempts to load shortcuts file → if missing, shows `firstrun.go` → otherwise creates `AppModel` and starts Bubble Tea with alt-screen.

### Key Bindings Summary

| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | Switch focus between sidebar and shortcut list |
| `↑↓` / `hjkl` | Navigate items |
| `n` | New app (sidebar) / New shortcut (list) |
| `g` | New group (shortcut panel only) |
| `e` | Edit selected shortcut |
| `d` | Delete shortcut, or delete empty group (header row) |
| `f` | Toggle favorite |
| `J` / `K` | Reorder shortcuts down/up |
| `/` | Activate search |
| `m` / `w` / `a` | OS filter: Mac / Windows / All |
| `?` | Help overlay |
| `:import <path>` | Merge another shortcuts file |
| `:set-path <path>` | Change active data file |
| `q` / `Ctrl+C` | Quit |

## Specs

- `specs/specs.md` — Product spec (data model v0.5); describes the canonical nested schema
- `specs/CLI/PLAN.md` — Implementation roadmap; phases 1–3 complete, phase 4 in progress, phase 5 (distribution) not started
- `specs/CLI/GROUP_PLAN.md` — Design plan for group support feature (historical reference)
