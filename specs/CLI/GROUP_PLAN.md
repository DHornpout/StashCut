# Plan: Group Support for StashCut CLI

## Context

The product spec (v0.5) defines a nested data model: `App → Groups → Shortcuts`. The CLI currently uses a flat model (`ShortcutFile.Shortcuts []Shortcut` with `Shortcut.AppID` as a foreign key). This plan migrates the CLI to the nested model and adds group management to the TUI.

**Decisions confirmed with user:**
- `g` to add group is active only when shortcut panel is focused
- Deleting a group (`d` on a group header) is **blocked** if the group has shortcuts
- Group headers are **always shown** even when only "Uncategorized" exists

---

## Files to Modify

1. `CLI/model/data.go`
2. `CLI/store/store.go`
3. `CLI/keybindings/keys.go`
4. `CLI/ui/form.go`
5. `CLI/ui/shortcutlist.go`
6. `CLI/ui/styles.go`
7. `CLI/ui/app.go`

> All changes must land in one commit because intermediate states won't compile (removing `ShortcutFile.Shortcuts` and `Shortcut.AppID` breaks store + UI simultaneously).

---

## Step 1 — `model/data.go`: Migrate to Nested Schema

**Add `Group` struct:**
```go
type Group struct {
    Name      string     `json:"name"`
    Shortcuts []Shortcut `json:"shortcuts"`
}
```

**Update `App`**: add `Groups []Group` and `UpdatedAt time.Time`. Keep `SortOrder` for migration compatibility.

**Update `Shortcut`**: remove `AppID string` field. Keep `SortOrder` (now relative to its group).

**Update `ShortcutFile`**: remove `Shortcuts []Shortcut` top-level field.

---

## Step 2 — `store/store.go`: Migration, New Helpers, Updated Merge

### Migration on load

Add private `legacyShortcutFile` + `legacyApp` + `legacyShortcut` structs (with `AppID` and top-level `Shortcuts`).

Update `Load()`:
1. Unmarshal into `legacyShortcutFile`.
2. If `len(legacy.Shortcuts) > 0` → call `migrateFromFlat()` which:
   - Creates each `App` with a single `Groups: []Group{{Name: "Uncategorized", Shortcuts: []Shortcut{}}}`.
   - Appends each legacy shortcut (without `AppID`) to the matching app's "Uncategorized" group, sorted by `SortOrder`.
3. Otherwise unmarshal into `model.ShortcutFile` normally, then call `ensureUncategorized()` on each app.

`ensureUncategorized(app *model.App)` — prepends an "Uncategorized" group if none exists.

### `New()` — remove `Shortcuts` field init

### Replace `ShortcutsForApp` with new helpers

```go
// FindShortcut returns (appIdx, groupIdx, scIdx, found) for a shortcut ID.
func FindShortcut(sf *model.ShortcutFile, id string) (int, int, int, bool)

// FindGroup returns the index of the group with the given name in an app, or -1.
func FindGroup(app *model.App, name string) int

// TotalShortcuts counts all shortcuts across all apps and groups.
func TotalShortcuts(sf *model.ShortcutFile) int
```

### Updated `Merge()`

For each incoming app:
- If not in base: append it (with `ensureUncategorized`).
- If in base: merge groups by name. For each incoming group, find matching group by name in base app; if absent, append. Within matched groups, merge shortcuts by ID using last-write-wins on `UpdatedAt`.

---

## Step 3 — `keybindings/keys.go`: Add `NewGroup` Binding

Add `NewGroup key.Binding` to `AppKeyMap` and `Keys`:
```go
NewGroup: key.NewBinding(
    key.WithKeys("g"),
    key.WithHelp("g", "new group"),
),
```

---

## Step 4 — `ui/form.go`: Group Forms

**Add `FormModeAddGroup` constant.**

**Update `FormSubmitMsg`**:
```go
type FormSubmitMsg struct {
    Mode      FormMode
    App       *model.App
    Shortcut  *model.Shortcut
    GroupName string // used for AddGroup and AddShortcut
    AppID     string // which app the group/shortcut belongs to
}
```

**Add `NewAddGroupForm(appID string) Form`** — single "Group Name" text field. `submit()` emits `FormSubmitMsg{Mode: FormModeAddGroup, GroupName: name, AppID: appID}`.

**Update `NewAddShortcutForm(appID string, groupNames []string) Form`**:
- If `len(groupNames) <= 1`: 4 fields (no group selector); `GroupName` in submit defaults to "Uncategorized".
- If `len(groupNames) > 1`: 5 fields — Description, Group (free-text input with hint showing existing names), Mac Keys, Windows Keys, Tags. If user enters a name not in `groupNames`, `app.go` auto-creates the group.

**Update `fieldLabels()`** to handle `FormModeAddGroup` ("Group Name") and the 5-field shortcut form.

---

## Step 5 — `ui/shortcutlist.go`: `ListRow` Type + Group Header Navigation

**Add types:**
```go
type RowKind int

const (
    RowKindHeader   RowKind = iota
    RowKindShortcut
)

type ListRow struct {
    Kind        RowKind
    GroupName   string
    Shortcut    model.Shortcut // valid when Kind == RowKindShortcut
    GroupIndex  int
    ShortcutIdx int
}
```

**Replace `Shortcuts []model.Shortcut` with `Rows []ListRow`** in `ShortcutList`. Keep `Selected int` (now indexes into `Rows`, always pointing to a `RowKindShortcut`).

**Add `GroupNames map[string]string`** (shortcutID → groupName) for search mode group column.

**Rename `SetShortcuts` → `SetRows(rows []ListRow)`**: clamps `Selected` to first `RowKindShortcut` index.

**Update `MoveUp()` / `MoveDown()`**: scan past `RowKindHeader` rows.

**Update `SelectedShortcut() *model.Shortcut`**: returns nil if row is a header.

**Add `SelectedRow() *ListRow`**: returns the current `ListRow` pointer.

**Update `View()`**: render `RowKindHeader` rows as a non-selectable section divider:
```
── Uncategorized ─────────────────────────────
```
Use `StyleGroupHeader` (added in `styles.go`). Selected highlight only applies to `RowKindShortcut` rows.

**Update `colLayout`**: add `groupW int` (shown in search mode only, analogous to `appW`).

**Add `StyleGroupHeader` in `styles.go`**:
```go
StyleGroupHeader = lipgloss.NewStyle().Foreground(colorMuted).Italic(true)
```

---

## Step 6 — `ui/app.go`: Wire Everything

### `refreshShortcuts()`

**Normal mode**: iterate `app.Groups`; for each group emit a `RowKindHeader` row then sort-and-emit its shortcuts (favorites first, then `SortOrder`). Call `m.list.SetRows(rows)`.

**Search mode**: iterate all apps × groups × shortcuts; for matches build `RowKindShortcut` rows with `GroupName` populated. Populate `m.list.GroupNames` map. Call `m.list.SetRows(rows)`.

### `handleKey()` changes

- **`kb.New` in shortcut panel**: pass `groupNames` to `NewAddShortcutForm(app.ID, groupNames)`.
- **`kb.NewGroup` in shortcut panel**: open `NewAddGroupForm(app.ID)`.

### `handleFormSubmit()` additions/changes

- **`FormModeAddApp`**: append `Groups: []model.Group{{Name: "Uncategorized", Shortcuts: []model.Shortcut{}}}` when creating the app.
- **`FormModeAddGroup`**: validate non-empty name, call `store.FindGroup` to block duplicates, append group to `m.data.Apps[ai].Groups`.
- **`FormModeAddShortcut`**: resolve group name (default "Uncategorized"), call `store.FindGroup`; auto-create group if not found; append shortcut to the group's `Shortcuts`.
- **`FormModeEditShortcut`**: use `store.FindShortcut(m.data, editOriginalID)` to locate `(ai, gi, si)` and update in place.

### `handleDelete()`

**Sidebar (delete app)**: unchanged except no `m.data.Shortcuts` cleanup needed.

**Shortcut panel**:
- If `SelectedRow().Kind == RowKindHeader`:
  - Block if `GroupName == "Uncategorized"` (status msg: "Cannot delete Uncategorized group").
  - Block if `len(group.Shortcuts) > 0` (status msg: "Group is not empty — delete its shortcuts first").
  - Otherwise delete the group.
- If `RowKindShortcut`: delete shortcut from its group using `store.FindShortcut`.

### `toggleFavorite()`

Use `store.FindShortcut(m.data, sc.ID)` → index directly into `m.data.Apps[ai].Groups[gi].Shortcuts[si]`.

### `moveShortcut()`

Use `store.FindShortcut` to get `(ai, gi, si)`. Swap `SortOrder` between `si` and `si+direction` within the same group. Re-scan `m.list.Rows` after `refreshShortcuts` to restore `Selected` to the moved shortcut.

### `importFile()`

Replace `len(data.Shortcuts)` / `len(data.Apps)` delta with `store.TotalShortcuts(data)` before and after `store.Merge`.

### `helpView()`

Add two lines:
```
g        New group (shortcut panel)
d        Delete group header (if empty) or shortcut
```

---

## Verification

1. `go build -o bin/stashcut .` — must compile cleanly (output to `bin/` folder).
2. Run with `go run .` against existing flat `shortcuts.json` — verify migration: shortcuts appear under "Uncategorized" group with header shown.
3. Create a new app, confirm it gets an "Uncategorized" group header in the shortcut list.
4. Press `g` in shortcut panel → fill form → group header appears.
5. Press `n` in shortcut panel with multiple groups → verify group field appears in form; enter existing and new group names.
6. Press `d` on an occupied group header → confirm blocked with status message.
7. Delete all shortcuts from a group → press `d` on its header → group removed.
8. Press `/` and search → results show app + group columns.
9. Favorites (`f`), reorder (`J`/`K`), edit (`e`) all work correctly with nested structure.
