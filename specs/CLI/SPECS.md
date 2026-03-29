# StashCut CLI — Feature Specification

**Version:** 1.0
**Stack:** Go 1.22+ · Bubble Tea · Bubbles · Lip Gloss
**Binary:** Single self-contained binary (`go build -o stashcut .`)

---

## Table of Contents

1. [Launch & Configuration](#1-launch--configuration)
2. [Data Model](#2-data-model)
3. [First-Run Experience](#3-first-run-experience)
4. [Layout](#4-layout)
5. [Sidebar — App List](#5-sidebar--app-list)
6. [Shortcut List](#6-shortcut-list)
7. [Search](#7-search)
8. [OS Filter](#8-os-filter)
9. [Forms](#9-forms)
10. [Favorites](#10-favorites)
11. [Reordering](#11-reordering)
12. [Deletion](#12-deletion)
13. [Command Mode](#13-command-mode)
14. [Import & Merge](#14-import--merge)
15. [Persistence & Dirty State](#15-persistence--dirty-state)
16. [Help Overlay](#16-help-overlay)
17. [Status Bar](#17-status-bar)
18. [Key Bindings Reference](#18-key-bindings-reference)
19. [Visual Design](#19-visual-design)

---

## 1. Launch & Configuration

### CLI Flag

```
stashcut --file <path>
```

Overrides the configured data file path for this session only.

### Config File

| Item | Value |
|---|---|
| Path | `~/.config/stashcut/config.json` |
| Created | Automatically on first run |
| Field | `data_file_path` — absolute path to `shortcuts.json` |

### Default Data File

```
~/Library/Application Support/Stashcut/shortcuts.json
```

Created automatically when the user selects **Create new file** on first run.

---

## 2. Data Model

The CLI shares the same `shortcuts.json` schema as the web and native apps.

### ShortcutFile (root)

| Field | Type | Description |
|---|---|---|
| `version` | string | Schema version |
| `meta` | Meta | File metadata |
| `apps` | []App | Ordered list of apps (array position = display order) |

### Meta

| Field | Type | Description |
|---|---|---|
| `created_at` | time.Time | File creation time |
| `updated_at` | time.Time | Updated on every save |
| `app_version` | string | App version that last wrote the file |

### App

| Field | Type | Description |
|---|---|---|
| `id` | string | UUID v4 |
| `name` | string | Display name |
| `icon` | string | Optional emoji (1–2 chars) |
| `created_at` | time.Time | Creation time |
| `updated_at` | time.Time | Last modified time |
| `groups` | []Group | Ordered list of groups. Always contains at least `"Uncategorized"`. |

### Group

| Field | Type | Description |
|---|---|---|
| `name` | string | Display name, unique within app. `"Uncategorized"` is the mandatory catch-all. |
| `shortcuts` | []Shortcut | Ordered list of shortcuts (array position = display order) |

### Shortcut

| Field | Type | Description |
|---|---|---|
| `id` | string | UUID v4 |
| `description` | string | Human-readable label |
| `keys_by_os` | map[string]KeysForOS | Keys keyed by `"mac"`, `"windows"`, or `"both"` |
| `is_favorite` | bool | Starred status |
| `tags` | []string | Searchable tags |
| `created_at` | time.Time | Creation time |
| `updated_at` | time.Time | Last edit time |

### KeysForOS

| Field | Type | Description |
|---|---|---|
| `keys` | string | Normalized binding: lowercase, `+` separator, no spaces (e.g. `"cmd+shift+p"`) |
| `keys_display` | string | Formatted display (e.g. `"⌘⇧P"`). Falls back to `keys` if empty. |

---

## 3. First-Run Experience

Shown when no shortcuts file is found at the configured path.

### Choose Screen

```
╭─────────────────────────────────────────╮
│  Welcome to StashCut CLI                │
│                                         │
│  No shortcuts file was found.           │
│                                         │
│  [C]  Create new file at default path   │
│       ~/Library/Application Support/…   │
│                                         │
│  [O]  Open / specify a different path   │
│                                         │
│  Press C or O — q to quit               │
╰─────────────────────────────────────────╯
```

| Key | Action |
|---|---|
| `C` | Create a new empty file at the default path |
| `O` | Open a text field to enter a custom path |
| `q` / `Ctrl+C` | Quit without creating anything |

### Specify Path Screen

A text input replaces the choose options. Pressing `Enter` confirms; `Esc` returns to the choose screen. `~/` paths are expanded to the full home directory. If the file does not exist at the specified path, a new empty file is created there.

The chosen path is persisted to `~/.config/stashcut/config.json` so subsequent launches go directly to the app.

---

## 4. Layout

```
┌─ Search bar (always visible, 1 line) ────────────────────────────────────┐
│  / to search                                                              │
├─ Sidebar ──────────────┬─ Shortcut List ───────────────────────────────┤
│ Apps                   │ App Name                                       │
│                        │    │ Description        │ Mac        │ Windows  │
│  🧑‍💻 VS Code             │ ───┼────────────────────┼────────────┼────────── │
│  🌐 Chrome             │ ★  │ Open Command…      │ ⌘⇧P        │ Ctrl⇧P   │
│  …                     │    │ Quick Open File    │ ⌘P         │ Ctrl+P   │
│                        │                                                │
├─ Status bar ──────────────────────────────────────────────────────────┤
│  ~/…/shortcuts.json  OS: All          Shortcut added  ? help  q quit  │
└────────────────────────────────────────────────────────────────────────┘
```

### Dimensions

| Element | Size |
|---|---|
| Sidebar width | 28 chars (fixed) |
| Sidebar max visible apps | 20 |
| Shortcut list width | Terminal width − sidebar − borders |
| Panel height | Terminal height − 3 (search bar + status bar + newline) |
| Search bar | 1 line, full terminal width |
| Status bar | 1 line, full terminal width |

### Panel Focus

`Tab` / `Shift+Tab` toggles focus between the **Sidebar** and the **Shortcut List**. The focused panel has a purple border; the unfocused panel has a dark gray border.

---

## 5. Sidebar — App List

- Displays all apps in array order.
- Each row: `{icon} {name}` — truncated to 22 chars with `…` if longer.
- Selected app is highlighted:
  - **Focused:** white text on dark-purple background.
  - **Unfocused:** accent-purple bold text.
- Scrolls when app count exceeds visible rows (max 20).
- Panel height adapts to content (does not stretch to fill terminal when fewer than 20 apps).

### Actions in Sidebar

| Key | Action |
|---|---|
| `↑` / `k` | Move selection up |
| `↓` / `j` | Move selection down |
| `n` | Open **Add App** form |
| `d` | Delete selected app and all its shortcuts |

---

## 6. Shortcut List

Displays shortcuts for the selected app in a table with a fixed header row and a horizontal divider.

### Normal Mode Columns

| Column | Width | Content |
|---|---|---|
| ★ | 2 | `★` (amber) if favorite, blank otherwise |
| Description | Dynamic (min 15) | Shortcut description |
| Mac | 18 | macOS key binding (`keys_display` preferred) |
| Windows | 18 | Windows key binding (`keys_display` preferred) |

Columns are separated by ` │ ` (3 chars, muted gray).

### Search Mode Columns

An **App** column (14 chars, muted) is inserted between ★ and Description, showing which app each result belongs to.

### OS Filter Effect on Columns

| Filter | Mac column | Windows column |
|---|---|---|
| All | Visible | Visible |
| macOS | Visible | Hidden |
| Windows | Hidden | Visible |

### Display Order

Shortcuts are displayed grouped by their group name. Within each group:
1. Favorites first (`is_favorite = true`)
2. Then by array position

### Empty State

- No shortcuts: `"No shortcuts yet — press n to add."`
- No search results: `"No matches found."`

### Actions in Shortcut List

| Key | Action |
|---|---|
| `↑` / `k` | Move selection up |
| `↓` / `j` | Move selection down |
| `n` | Open **Add Shortcut** form |
| `e` | Open **Edit Shortcut** form (pre-filled) |
| `d` | Delete selected shortcut |
| `f` | Toggle favorite on selected shortcut |
| `J` | Move shortcut down (swap array position with next) |
| `K` | Move shortcut up (swap array position with previous) |

---

## 7. Search

### Activation

Press `/` — the search bar at the top activates and receives keyboard focus.

### Scope

Searches **all shortcuts across all apps** (not just the selected app).

Matches on:
- `description` (case-insensitive substring)
- `tags` (case-insensitive substring, any tag)

Does **not** match on app name or key bindings.

### Results

- Shown in the Shortcut List panel with the **App** column visible.
- Title changes to `"Search Results ({count})"`.
- Sorted: favorites first, then by array position, grouped by app.
- Results update in real time as you type.

### Deactivation

Press `Esc` or `Enter`. The shortcut list returns to the selected app's shortcuts.

### Search Bar

- Full-width, 1-line bar at the top of the screen.
- Inactive: shows `"  / to search"` hint (muted text).
- Active: shows `/ ` prompt (accent) followed by a live text input.
- Character limit: 100.

---

## 8. OS Filter

Toggles which key-binding columns are shown and which shortcuts' keys are displayed.

| Key | Filter | Status Bar Label |
|---|---|---|
| `a` | All platforms | `All` |
| `m` | macOS only | `macOS` |
| `w` | Windows only | `Windows` |

Default on launch: **All**.

---

## 9. Forms

All forms render as a full-screen overlay with a rounded purple border. Navigation within a form uses `Tab` / `↑` / `↓`. `Enter` on the **last field** submits. `Esc` cancels.

### Add App

| Field | Limit | Required |
|---|---|---|
| Name | 64 chars | Yes |
| Icon | 4 chars (emoji) | No |

On submit: UUID assigned, `"Uncategorized"` group created automatically, `created_at` = `updated_at` = now.

### Add Shortcut

| Field | Limit | Notes |
|---|---|---|
| Description | 128 chars | Required |
| Group | — | Select from existing groups for the app. Defaults to `"Uncategorized"`. |
| Mac Keys | 64 chars | Stored under `keys_by_os["mac"]`. Normalized to lowercase+`+` on save. |
| Windows Keys | 64 chars | Stored under `keys_by_os["windows"]`. Normalized to lowercase+`+` on save. |
| Tags | 128 chars | Comma-separated, trimmed |

On submit: UUID assigned, shortcut appended to the selected group's `shortcuts` array, `created_at` = `updated_at` = now.

### Add Group

| Field | Limit | Notes |
|---|---|---|
| Name | 64 chars | Required. Must be unique within the app. `"Uncategorized"` is reserved. |

On submit: group appended to the app's `groups` array with an empty `shortcuts` list.

### Edit Shortcut

Same fields as Add Shortcut, pre-filled with current values. On submit: `description`, `keys_by_os`, `group`, `tags`, and `updated_at` are updated. `id`, `created_at`, and `is_favorite` are preserved. If the group changed, the shortcut is moved to the end of the new group's array.

---

## 10. Favorites

- **Key:** `f` (shortcut list focused)
- Toggles `is_favorite` on the selected shortcut.
- Starred shortcuts show `★` in amber in the first column.
- Favorites always sort to the top of the list (within the app or within search results).
- Change is saved immediately.

---

## 11. Reordering

- **`J`** — move selected shortcut down (swap position with the next item in the group's array).
- **`K`** — move selected shortcut up (swap position with the previous item in the group's array).
- Only available in the Shortcut List panel.
- Blocked at group boundaries (no wrap-around, no cross-group movement).
- Change is saved immediately.

---

## 12. Deletion

### Delete App (`d` in sidebar)

1. Removes the app from the `apps` array (including all its groups and shortcuts).
2. Selection adjusts if the deleted app was the last item.
3. Status: `"Deleted app: {name}"`.
4. Saved immediately.

### Delete Group (`d` on a group header)

1. Blocked if the group is `"Uncategorized"` — status: `"Cannot delete Uncategorized group"`.
2. Blocked if the group contains shortcuts — status: `"Move or delete shortcuts before deleting group"`.
3. Otherwise removes the group from the app's `groups` array.
4. Status: `"Deleted group: {name}"`.
5. Saved immediately.

### Delete Shortcut (`d` in shortcut list)

1. Removes the shortcut from its group's `shortcuts` array.
2. Status: `"Shortcut deleted"`.
3. Saved immediately.

No confirmation prompt — deletion is immediate.

---

## 13. Command Mode

### Activation

Press `:` — a single-line command bar replaces the search bar at the top.

`Esc` cancels. `Enter` executes.

### Commands

#### `:import <path>`

Merges another `shortcuts.json` file into the current data using last-write-wins conflict resolution (see [Import & Merge](#14-import--merge)).

```
:import ~/Downloads/shortcuts.json
:import /absolute/path/shortcuts.json
```

#### `:set-path <path>`

Changes the active data file path for this session and saves it to `~/.config/stashcut/config.json`.

```
:set-path ~/Dropbox/shortcuts.json
```

Both commands support `~/` home directory expansion.

---

## 14. Import & Merge

### Algorithm

1. Load the incoming file from the specified path.
2. **Apps:** Any app ID not present in the current data is appended to the end of `apps` (with its groups and shortcuts intact).
3. **Groups:** For apps present in both files, any group name not present in the current app is appended to the end of the app's `groups` array.
4. **Shortcuts:** All shortcuts across all groups in the incoming file are matched by `id` against the current file:
   - If the incoming shortcut ID already exists and `incoming.updated_at > existing.updated_at`, the existing shortcut is replaced in-place (last-write-wins).
   - If the incoming shortcut ID does not exist, it is appended to the end of the matching group (matched by name); if no matching group exists, it is appended to `"Uncategorized"`.
5. The merged data is saved immediately.

### Status Messages

| Outcome | Message |
|---|---|
| Success | `"Imported: +{N} apps, +{M} shortcuts"` |
| File not found | `"Import error: file not found: {path}"` |
| Parse error | `"Import error: {error}"` |

---

## 15. Persistence & Dirty State

- Every mutation (add, edit, delete, favorite, reorder, import, set-path) triggers an **async save** via a `tea.Cmd`.
- While the save is in flight, `[unsaved]` appears in red in the status bar.
- `[unsaved]` clears when the save completes (`SavedMsg` received).
- On save, `meta.updated_at` is updated to the current time.
- File is written as indented JSON (2-space) with permissions `0644`.
- Parent directories are created automatically (`0755`).

---

## 16. Help Overlay

**Key:** `?` — toggles the overlay on/off.

Displays a rounded-bordered panel listing all key bindings:

```
  StashCut CLI — Key Bindings

  Navigation
    Tab / Shift+Tab    Switch panel
    ↑ / ↓  or  k / j  Navigate list

  Actions
    n        New app (sidebar) / New shortcut (list)
    e        Edit selected shortcut
    d        Delete selected item
    f        Toggle favorite on shortcut
    J / K    Move shortcut down / up (reorder)

  Filter & Search
    /        Search (description + tags)
    m        Filter: macOS
    w        Filter: Windows
    a        Filter: All platforms

  Commands  (press : to open)
    :import <path>     Merge another shortcuts.json
    :set-path <path>   Change the active file path

  General
    ?        Toggle this help
    q        Quit
```

---

## 17. Status Bar

Single line at the bottom of the screen. Dark background (#111827), muted text (#6B7280).

```
 ~/…/shortcuts.json [unsaved]  OS: macOS          Shortcut added  ? help  q quit
```

| Section | Content |
|---|---|
| Left | Data file path + `[unsaved]` (red) if dirty |
| Left | `OS: {All \| macOS \| Windows}` |
| Right | Last status message |
| Right | `? help  q quit` (always shown) |

---

## 18. Key Bindings Reference

### Global

| Key | Action |
|---|---|
| `?` | Toggle help overlay |
| `q` / `Ctrl+C` | Quit |
| `/` | Activate search |
| `:` | Activate command mode |
| `m` | OS filter: macOS |
| `w` | OS filter: Windows |
| `a` | OS filter: All |
| `Tab` | Switch focus to next panel |
| `Shift+Tab` | Switch focus to previous panel |
| `↑` / `k` | Move selection up |
| `↓` / `j` | Move selection down |

### Sidebar (focused)

| Key | Action |
|---|---|
| `n` | New app |
| `d` | Delete selected app |

### Shortcut List (focused)

| Key | Action |
|---|---|
| `n` | New shortcut |
| `e` | Edit selected shortcut |
| `d` | Delete selected shortcut |
| `f` | Toggle favorite |
| `J` | Move shortcut down |
| `K` | Move shortcut up |

### Search Mode

| Key | Action |
|---|---|
| Any character | Filter results in real time |
| `Esc` / `Enter` | Deactivate search |

### Command Mode

| Key | Action |
|---|---|
| Any character | Build command string |
| `Enter` | Execute command |
| `Esc` | Cancel |

### Forms

| Key | Action |
|---|---|
| `Tab` / `↓` | Next field |
| `Shift+Tab` / `↑` | Previous field |
| `Enter` (last field) | Submit form |
| `Esc` | Cancel form |

---

## 19. Visual Design

### Color Palette

| Name | Hex | Usage |
|---|---|---|
| Primary (purple) | `#7C3AED` | Focused panel border, form border |
| Accent (light purple) | `#A78BFA` | Titles, selected text (unfocused), key bindings |
| Muted (gray) | `#6B7280` | Secondary text, table separators, status bar |
| Selected bg | `#1E1B4B` | Selected row background, search/command bar bg |
| Border | `#374151` | Unfocused panel border |
| Favorite (amber) | `#F59E0B` | Favorite star `★` |
| Error (red) | `#EF4444` | `[unsaved]` indicator, error messages |
| Success (green) | `#10B981` | Reserved |
| Status bg | `#111827` | Status bar background |

### Borders

- **Focused panel:** Rounded border, purple (`#7C3AED`)
- **Unfocused panel:** Rounded border, dark gray (`#374151`)
- **Forms / help overlay:** Rounded border, purple

### Special Characters

| Symbol | Usage |
|---|---|
| `★` | Favorite marker (amber) |
| `…` | Truncation ellipsis |
| ` │ ` | Table column separator |
| `─` / `─┼─` | Table header divider |
| `/` | Search prompt |
| `:` | Command prompt |
