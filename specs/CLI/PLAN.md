# StashCut CLI — Implementation Plan

**Version:** 1.0
**Date:** 2026-03-28
**Stack:** Go + Bubble Tea

---

## 1. Prerequisites — What to Install

### Go toolchain
- **Go 1.22+** — download from https://go.dev/dl/
  - Installs: `go` compiler, `gofmt`, `go test`, `go build`
  - After install, verify: `go version`
  - Sets `GOPATH` automatically (default: `~/go`)

### Editor support
- **VS Code** extension: `golang.go` (Go Team at Google)
  - Auto-installs: `gopls` (language server), `dlv` (debugger), `staticcheck` (linter)
  - Or use any editor with LSP support (Zed, Neovim, GoLand)

### Optional but recommended
- **`golangci-lint`** — multi-linter runner: `brew install golangci-lint`
- **`air`** — live reload during development: `go install github.com/air-verse/air@latest`

### Runtime requirements for end users
- **None.** `go build` produces a single self-contained binary.
- No Go installation required on the user's machine.
- Distribute the binary directly or via a Homebrew formula.

---

## 2. Project Structure

```
StashCut/CLI/
├── main.go                  # Entry point
├── go.mod                   # Module definition
├── go.sum                   # Dependency lock file
│
├── model/
│   └── data.go              # Go structs mirroring shortcuts.json schema
│
├── store/
│   └── store.go             # Load / save / merge JSON file
│
├── ui/
│   ├── app.go               # Root Bubble Tea model (wires panels together)
│   ├── sidebar.go           # Left panel: app list
│   ├── shortcutlist.go      # Right panel: shortcut list
│   ├── form.go              # Add / edit shortcut form
│   ├── search.go            # Search bar component
│   ├── osfilter.go          # OS filter toggle (Mac | Windows | All)
│   └── styles.go            # Lip Gloss style definitions
│
├── keybindings/
│   └── keys.go              # All key mappings in one place
│
└── config/
    └── config.go            # File path config (UserDefaults equivalent via prefs file)
```

---

## 3. External Dependencies

| Package | Purpose |
|---|---|
| `github.com/charmbracelet/bubbletea` | TUI event loop and state management |
| `github.com/charmbracelet/bubbles` | Prebuilt components: list, textinput, textarea |
| `github.com/charmbracelet/lipgloss` | Layout and styling |
| `github.com/google/uuid` | UUID v4 generation for new apps/shortcuts |

All fetched automatically via `go get` — no manual downloads.

---

## 4. Data Model

Go structs that map 1:1 to `shortcuts.json` (shared with web app):

```go
// ShortcutFile is the root JSON object
type ShortcutFile struct {
    Version   string     `json:"version"`
    Meta      Meta       `json:"meta"`
    Apps      []App      `json:"apps"`
    Shortcuts []Shortcut `json:"shortcuts"`
}

type Meta struct {
    CreatedAt  time.Time `json:"created_at"`
    UpdatedAt  time.Time `json:"updated_at"`
    AppVersion string    `json:"app_version"`
}

type App struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    Icon      string    `json:"icon"`
    SortOrder int       `json:"sort_order"`
    CreatedAt time.Time `json:"created_at"`
}

type KeysForOS struct {
    Keys        string `json:"keys"`
    KeysDisplay string `json:"keys_display"`
}

type Shortcut struct {
    ID          string               `json:"id"`
    AppID       string               `json:"app_id"`
    Description string               `json:"description"`
    KeysByOS    map[string]KeysForOS `json:"keys_by_os"`
    IsFavorite  bool                 `json:"is_favorite"`
    SortOrder   int                  `json:"sort_order"`
    Tags        []string             `json:"tags"`
    CreatedAt   time.Time            `json:"created_at"`
    UpdatedAt   time.Time            `json:"updated_at"`
}
```

---

## 5. Implementation Phases

### Phase 1 — Foundation
- [ ] Init Go module (`go mod init`)
- [ ] Add dependencies (`go get`)
- [ ] Implement `model/data.go` — structs
- [ ] Implement `store/store.go` — load, save, create new file
- [ ] Implement `config/config.go` — read/write data file path from `~/.config/stashcut/config.json`

### Phase 2 — TUI Shell
- [ ] Implement root `ui/app.go` — two-panel layout (sidebar | shortcut list)
- [ ] Implement `ui/styles.go` — colors, borders, layout constants
- [ ] Implement `ui/sidebar.go` — scrollable app list, selected state
- [ ] Implement `ui/shortcutlist.go` — list of shortcuts for selected app
- [ ] Wire keyboard navigation: arrow keys move focus, Tab switches panels

### Phase 3 — Core Features
- [ ] Implement `ui/search.go` — `/` to activate, filters across all apps
- [ ] Implement `ui/osfilter.go` — toggle Mac / Windows / All with `m`, `w`, `a`
- [ ] Favorite toggle — `f` on a shortcut row toggles star, saves immediately
- [ ] Add App — `n` in sidebar opens inline form
- [ ] Add Shortcut — `n` in shortcut list opens `ui/form.go`
- [ ] Delete App / Shortcut — `d` with confirmation prompt
- [ ] Edit Shortcut — `e` opens prefilled form

### Phase 4 — Polish
- [ ] Reorder shortcuts — `J` / `K` (shift+j/k) moves item up/down
- [ ] Import / merge — `:import <path>` command, last-write-wins on `updated_at`
- [ ] Help overlay — `?` shows all keybindings
- [ ] Status bar — shows file path, dirty state (`[unsaved]`), current OS filter
- [ ] First-run experience — if no file found, prompt to create or open

### Phase 5 — Distribution
- [ ] Cross-compile for macOS arm64 + amd64: `GOOS=darwin GOARCH=arm64 go build`
- [ ] Write a Homebrew formula (optional)
- [ ] GitHub Actions CI: build + test on push

---

## 6. Key Bindings (planned)

| Key | Action |
|---|---|
| `Tab` / `Shift+Tab` | Switch focus between sidebar and shortcut list |
| `↑` / `↓` | Navigate items in focused panel |
| `Enter` | Select app (sidebar) / Edit shortcut (list) |
| `n` | New app (sidebar focused) / New shortcut (list focused) |
| `e` | Edit selected shortcut |
| `d` | Delete selected item (with confirmation) |
| `f` | Toggle favorite on selected shortcut |
| `J` / `K` | Move shortcut down / up (reorder) |
| `/` | Activate search |
| `m` / `w` / `a` | OS filter: Mac / Windows / All |
| `?` | Toggle help overlay |
| `q` / `Ctrl+C` | Quit |

---

## 7. File Path Config

Stored at `~/.config/stashcut/config.json`:

```json
{
  "data_file_path": "~/Library/Application Support/Stashcut/shortcuts.json"
}
```

Default created on first run. User can override via `:set-path` command or a `--file` CLI flag.

---

## 8. Build & Run

```bash
# Development
go run .

# Production build (current platform)
go build -o stashcut .

# Release builds
GOOS=darwin GOARCH=arm64  go build -o stashcut-macos-arm64 .
GOOS=darwin GOARCH=amd64  go build -o stashcut-macos-amd64 .

# Install to PATH
go install .
```

---

## 9. Out of Scope for CLI (v1)

- Global hotkey (`Cmd+Option+K`) — not applicable for CLI
- App icons — text/emoji only in terminal
- Drag-and-drop reorder — replaced by `J`/`K` key reorder
- File System Access API — not applicable
