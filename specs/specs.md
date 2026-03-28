# Stashcut — Product Specification

**Version:** 0.4  
**Date:** 2026-03-19  
**Status:** Ready for web implementation

### Changelog

| Version | Date | Changes |
|---|---|---|
| 0.1 | 2026-03-19 | Initial draft |
| 0.2 | 2026-03-19 | Resolved all open questions. Redesigned shortcut data model to eliminate duplicate entries for cross-OS shortcuts. Added user-configurable file path per platform. Clarified tags as v2 with suggested-tags dropdown. |
| 0.3 | 2026-03-19 | Renamed application to Stashcut. |
| 0.4 | 2026-03-19 | Added web-specific implementation decisions: first launch UX, key entry method, icon sourcing, OS filter default, and UI layout. Added Section 10: Web Implementation Guide. |

---

## 1. Overview

A personal keyboard shortcut reference tool that lets users browse, organize, and favorite shortcuts by application. The user manually enters shortcuts they want to remember or learn. The tool is not a key remapper — it is a reference library.

### 1.1 Problem statement

Users who work across multiple applications (and across Windows and macOS) frequently forget keyboard shortcuts. Current solutions either require an internet search every time, or show all shortcuts for an app with no way to prioritize the ones that matter most to the individual user.

### 1.2 Goals

- Browse shortcuts by application
- Mark favorites and reorder by personal priority
- Work identically across macOS, Windows, and web
- Store all data as a single local JSON file, path configurable by the user per platform
- Be fast to open and query while another application is in use

### 1.3 Non-goals

- Key remapping or rebinding (use PowerToys or BetterTouchTool for this)
- Automatic detection of running apps
- Team or shared shortcut libraries (v1 scope)
- Cloud sync (future scope)
- Menu bar / tray widget (deferred to v2)

---

## 2. Platform targets

| Platform | Delivery | Notes |
|---|---|---|
| macOS | Native app (Swift / SwiftUI) | Menu bar icon, global hotkey |
| Windows | Native app (WinUI 3 / .NET) | System tray icon, global hotkey |
| Web | React SPA | File System Access API (Chrome/Edge); manual import/export fallback for Safari |

All three platforms share the same JSON data model and schema. Because each platform can point to a user-specified file path, a single `shortcuts.json` stored in a shared folder (e.g. Dropbox, iCloud Drive) can serve as a manual sync mechanism across devices without a dedicated sync feature.

---

## 3. Features

### 3.1 Must-have (v1)

| Feature | Description |
|---|---|
| App library | Sidebar listing all apps the user has added. Click to filter shortcuts to that app. |
| Shortcut list | Per-app list of shortcuts showing key combos per OS + description. |
| Manual entry | User adds shortcuts themselves via a simple form. |
| Favorites | Star any shortcut. Starred shortcuts float to the top of the list. Favorite state is shared across OS variants of the same shortcut. |
| Custom ordering | Drag and drop to reorder shortcuts within an app. Order is shared across OS variants. |
| Global search | Search across all apps and descriptions simultaneously. |
| Quick-access hotkey | Global keyboard shortcut to bring the app to front (macOS: `Cmd+Option+K`, Windows: `Ctrl+Alt+K`). |
| OS filter | Toggle to show shortcuts for Mac, Windows, or All. Defaults to **All** on web. Native apps default to their own OS. |
| Configurable file path | Each platform lets the user point to any `.json` file on their filesystem as the data file. |
| File System Access API (web) | Web app saves directly to the user's chosen file on Chrome/Edge. Falls back to download/upload on Safari. |
| Import / export | Export full library as a `.json` file. Import merges using last-write-wins on `updated_at` per shortcut `id`. |

### 3.2 Nice-to-have (v2)

| Feature | Description |
|---|---|
| Preset library | Pre-seeded shortcut lists for common apps (Chrome, VS Code, Slack, Figma). User selects which presets to import. |
| Tags | Tag shortcuts with categories. Suggested-tags dropdown prevents free-form duplication. Filter by tag across apps. |
| Menu bar / tray widget | Show shortcuts for the currently active app from menu bar (macOS) or system tray (Windows). |
| Favorites-only view | Toggle to show only starred shortcuts for a focused cheat sheet. |
| Keyboard navigation | Full keyboard-only navigation within the app for power users. |

### 3.3 Future (v3+)

| Feature | Description |
|---|---|
| Cloud sync | Optional sync via iCloud (macOS) or OneDrive (Windows). |
| Shared presets | Community-contributed preset packs. |
| Learning mode | Quiz mode to test whether shortcuts have been memorized. |

---

## 4. Data model

All data is stored in a single JSON file (default name `shortcuts.json`) on the user's local filesystem. The file path is configurable per platform — see section 6.

### 4.1 File structure

```json
{
  "version": "1.0",
  "meta": {
    "created_at": "2026-03-19T10:00:00Z",
    "updated_at": "2026-03-19T14:30:00Z",
    "app_version": "1.0.0"
  },
  "apps": [...],
  "shortcuts": [...]
}
```

| Field | Type | Description |
|---|---|---|
| `version` | string | Schema version. Used to trigger migrations when the format changes. |
| `meta.created_at` | ISO 8601 string | When the file was first created. |
| `meta.updated_at` | ISO 8601 string | Last time any change was saved. |
| `meta.app_version` | string | Version of the app that last wrote this file. |
| `apps` | array | List of app definitions. |
| `shortcuts` | array | List of all shortcut entries across all apps. |

---

### 4.2 App object

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Chrome",
  "icon": "chrome",
  "sort_order": 1,
  "created_at": "2026-03-19T10:00:00Z"
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `id` | UUID v4 string | Yes | Unique identifier. Generated as UUID v4 on creation. |
| `name` | string | Yes | Display name shown in the sidebar. |
| App icons | Built-in icon slugs for common apps (Chrome, VS Code, Slack, Figma, Terminal, etc.). If no built-in icon matches, user can upload a custom PNG. Falls back to a generated initial avatar if neither is set. |
| `sort_order` | integer | Yes | Position in the app sidebar list. Lower = higher in list. |
| `created_at` | ISO 8601 string | Yes | When the app entry was created. |

---

### 4.3 Shortcut object

**Core design principle: a shortcut entry represents one action, not one OS.** If the same action has different key combinations on Mac vs Windows, both are stored within the same entry under `keys_by_os`. Shared metadata — `description`, `is_favorite`, `sort_order`, `tags` — is defined once at the entry level and applies to all OS variants. Only the actual key combinations differ per OS.

```json
{
  "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "app_id": "550e8400-e29b-41d4-a716-446655440000",
  "description": "Reopen last closed tab",
  "keys_by_os": {
    "mac": {
      "keys": "Cmd + Shift + T",
      "keys_display": "⌘⇧T"
    },
    "windows": {
      "keys": "Ctrl + Shift + T",
      "keys_display": "Ctrl+Shift+T"
    }
  },
  "is_favorite": true,
  "sort_order": 1,
  "tags": ["tabs", "navigation"],
  "created_at": "2026-03-19T10:05:00Z",
  "updated_at": "2026-03-19T10:05:00Z"
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `id` | UUID v4 string | Yes | Unique identifier. Generated as UUID v4 on creation. |
| `app_id` | UUID v4 string | Yes | Foreign key reference to the parent app's `id`. |
| `description` | string | Yes | What the shortcut does. Shared across all OS variants. Example: `"Reopen last closed tab"`. |
| `keys_by_os` | object | Yes | Map of OS keys to their key definitions. Must have at least one of: `"mac"`, `"windows"`, `"both"`. |
| `keys_by_os.<os>.keys` | string | Yes | Raw canonical key combination string (e.g. `"Ctrl + Shift + T"`). Used for search. |
| `keys_by_os.<os>.keys_display` | string | No | Symbol/display string (e.g. `"⌘⇧T"`). Falls back to `keys` if omitted. |
| `is_favorite` | boolean | Yes | Whether the user has starred this shortcut. Defaults to `false`. Shared across all OS variants. |
| `sort_order` | integer | Yes | Position within the app's shortcut list. Shared across OS variants. Lower = higher. |
| `tags` | array of strings | No | User-defined category tags. Shared across OS variants. Populated via suggested-tags dropdown in v2. |
| `created_at` | ISO 8601 string | Yes | When the shortcut was first added. |
| `updated_at` | ISO 8601 string | Yes | When the shortcut was last modified. Used for last-write-wins merge. |

#### `keys_by_os` key reference

| Key | Meaning |
|---|---|
| `"mac"` | Key combination applies to macOS only. |
| `"windows"` | Key combination applies to Windows only. |
| `"both"` | A single key combination identical on both platforms (e.g. `Ctrl+P` in VS Code). |

An entry may have `"mac"` and `"windows"` simultaneously (different combos), only `"both"` (same combo on both), or just one OS key if the shortcut exists on one platform only.

**Runtime display logic:**

| Current OS | Key shown |
|---|---|
| macOS | `keys_by_os.mac` → fallback to `keys_by_os.both` |
| Windows | `keys_by_os.windows` → fallback to `keys_by_os.both` |
| All (filter override) | Both mac and windows keys shown side by side |

---

### 4.4 Full example file

```json
{
  "version": "1.0",
  "meta": {
    "created_at": "2026-03-19T10:00:00Z",
    "updated_at": "2026-03-19T14:30:00Z",
    "app_version": "1.0.0"
  },
  "apps": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "Chrome",
      "icon": "chrome",
      "sort_order": 1,
      "created_at": "2026-03-19T10:00:00Z"
    },
    {
      "id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
      "name": "VS Code",
      "icon": "vscode",
      "sort_order": 2,
      "created_at": "2026-03-19T10:00:00Z"
    }
  ],
  "shortcuts": [
    {
      "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
      "app_id": "550e8400-e29b-41d4-a716-446655440000",
      "description": "Reopen last closed tab",
      "keys_by_os": {
        "mac": {
          "keys": "Cmd + Shift + T",
          "keys_display": "⌘⇧T"
        },
        "windows": {
          "keys": "Ctrl + Shift + T",
          "keys_display": "Ctrl+Shift+T"
        }
      },
      "is_favorite": true,
      "sort_order": 1,
      "tags": ["tabs", "navigation"],
      "created_at": "2026-03-19T10:05:00Z",
      "updated_at": "2026-03-19T10:05:00Z"
    },
    {
      "id": "a87ff679-a2f3-471d-b2e3-5b1c56c6a1e3",
      "app_id": "550e8400-e29b-41d4-a716-446655440000",
      "description": "Focus the address bar",
      "keys_by_os": {
        "mac": {
          "keys": "Cmd + L",
          "keys_display": "⌘L"
        },
        "windows": {
          "keys": "Ctrl + L",
          "keys_display": "Ctrl+L"
        }
      },
      "is_favorite": false,
      "sort_order": 2,
      "tags": ["navigation"],
      "created_at": "2026-03-19T10:06:00Z",
      "updated_at": "2026-03-19T10:06:00Z"
    },
    {
      "id": "eccbc87e-4b5c-e9fe-862a-b3fc8b9f5e6a",
      "app_id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
      "description": "Quick open file",
      "keys_by_os": {
        "both": {
          "keys": "Ctrl + P",
          "keys_display": "Ctrl+P"
        }
      },
      "is_favorite": true,
      "sort_order": 1,
      "tags": ["files", "navigation"],
      "created_at": "2026-03-19T11:00:00Z",
      "updated_at": "2026-03-19T11:00:00Z"
    },
    {
      "id": "c4ca4238-a0b9-3382-8dcc-509a6f75849b",
      "app_id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
      "description": "Open integrated terminal",
      "keys_by_os": {
        "mac": {
          "keys": "Ctrl + `",
          "keys_display": "Ctrl+`"
        },
        "windows": {
          "keys": "Ctrl + `",
          "keys_display": "Ctrl+`"
        }
      },
      "is_favorite": true,
      "sort_order": 2,
      "tags": ["terminal"],
      "created_at": "2026-03-19T11:05:00Z",
      "updated_at": "2026-03-19T11:05:00Z"
    }
  ]
}
```

---

### 4.5 Data design decisions

**Why `keys_by_os` as a nested object instead of a flat `os` field with duplicate entries?**  
The v0.1 design created two separate shortcut entries for "Reopen last closed tab" — one for Mac, one for Windows — duplicating `description`, `is_favorite`, `sort_order`, and `tags`. The new model stores the action once and nests only the parts that differ (the key combination) under each OS. Starring a shortcut or reordering it applies to the action universally, which is the correct behavior — a user's preference does not change based on which computer they are on.

**Why UUID v4 for `id` fields?**  
UUIDs prevent collisions when merging two independently maintained `shortcuts.json` files (e.g. one from a Mac, one from a Windows machine that both evolved separately). Incrementing integers would collide if both files were edited independently before being merged.

**Why store `sort_order` as an integer instead of relying on array position?**  
Array position is fragile during partial imports and merges. An explicit `sort_order` integer survives reordering operations and makes the intended order unambiguous.

**Why is `tags` included in the schema now if it is a v2 feature?**  
The field is defined in the schema today so that any data entered in v1 (e.g. by a power user who edits the JSON directly) is forward-compatible. Apps in v1 read and write-preserve the field but do not render any tags UI.

**Why support `"both"` as a `keys_by_os` key alongside `"mac"` and `"windows"`?**  
Some shortcuts are identical across platforms (e.g. `Ctrl+P` in VS Code). Using `"both"` avoids writing the same key combination twice while still being explicit that the entry applies on either platform.

---

## 5. UX flows

### 5.1 Add a shortcut
1. User selects an app from the sidebar (or creates a new app).
2. User clicks **+** (Add shortcut). A form slides in on the right panel.
3. The form contains:
   - **Description** — text field (required).
   - **Mac keys** — a key combo input with a **Record** button. Clicking Record captures the next key combo pressed in the browser. A plain text field is always available as a manual fallback.
   - **Windows keys** — same dual input as Mac keys.
   - If the same combo applies to both OS, user fills in either field and checks **Same on both platforms** to mirror it automatically.
4. User saves. The new shortcut appears at the bottom of the list for that app.

### 5.2 Mark a favorite
1. User hovers over a shortcut row.
2. A star icon appears. User clicks it.
3. `is_favorite` is set to `true`. The shortcut moves to the top of the favorites group.
4. The star state applies to both Mac and Windows views of the same shortcut.

### 5.3 Reorder shortcuts
1. User drags a shortcut row up or down within the list.
2. `sort_order` values are updated and saved immediately.
3. Favorites can only be reordered within the favorites group; non-favorites within their group.
4. Reordering in Mac view also reorders in Windows view (order is shared).

### 5.4 Filter by OS
1. A segmented toggle in the toolbar: **Mac | Windows | All**.
2. Web app defaults to **All** on every load.
3. Native macOS app defaults to **Mac**; native Windows app defaults to **Windows**.
4. Mac: shows `keys_by_os.mac` key, falling back to `keys_by_os.both`.
5. Windows: shows `keys_by_os.windows` key, falling back to `keys_by_os.both`.
6. All: shows both Mac and Windows key combos side by side on each shortcut row.

### 5.5 Search
1. A search bar is always visible at the top.
2. Typing filters across all apps by `description` and all `keys` values within `keys_by_os`.
3. Results are grouped by app.

### 5.6 Change the data file path
1. User opens Settings.
2. Under **Data File**, the current file path is shown.
3. User clicks **Change** and picks any `.json` file via the native file picker (or File System Access API on web Chrome/Edge).
4. The app loads the selected file immediately. If the file does not yet exist, the app offers to create a new empty one at that path.
5. The chosen path is persisted in the app's local preferences store (not in the JSON data file itself) so it is remembered across sessions.

### 5.7 Import / merge
1. User clicks Import in settings and selects a `.json` file.
2. The app validates the schema `version` field.
3. Merge logic runs per shortcut `id` using last-write-wins on `updated_at`. Shortcuts only in the imported file are added. Shortcuts only in the local file are kept. Shortcuts in both files are kept at whichever has the later `updated_at`.
4. The merged result is saved immediately to the active data file.

---

## 6. Storage and file path configuration

### 6.1 Default paths

| Platform | Default file location |
|---|---|
| macOS | `~/Library/Application Support/Stashcut/shortcuts.json` |
| Windows | `%APPDATA%\Stashcut\shortcuts.json` |
| Web | No default. User must open an existing file or create a new one on first launch. |

### 6.2 User-configurable path

Each platform stores the chosen file path in its own local preferences store. This setting is per-device and per-app — changing the path on one machine does not affect any other.

| Platform | Preferences store |
|---|---|
| macOS | `UserDefaults` — key: `dataFilePath` |
| Windows | Registry — `HKCU\Software\Stashcut\dataFilePath` |
| Web (Chrome/Edge) | File System Access API file handle persisted in `IndexedDB` |
| Web (Safari) | No persistent handle. User re-selects file on each session. |

### 6.3 Manual sync via shared folder

Because any platform can point to any file path, users can achieve cross-device sync without a cloud backend by placing `shortcuts.json` in a synced folder (iCloud Drive, Dropbox, OneDrive, or a network drive) and pointing each app at that shared path. The app does not manage sync conflicts at this layer — if two devices write simultaneously, the filesystem's last write wins. For intentional controlled merging, use the Import flow (section 5.7).

---

## 7. Schema versioning and migration

The `version` field at the root of the JSON file tracks the schema version. On every file load the app checks this field and runs any pending migrations before rendering.

Migration rules:
- Always migrate forward (e.g. `1.0` → `1.1` → `2.0`), never backward.
- Migrations must be additive or transformative — never destructive.
- Before migrating, the app writes a timestamped backup: `shortcuts.backup.YYYYMMDDHHMMSS.json` in the same directory as the data file.
- After migration completes, the file is saved immediately with the updated `version` value.

---

## 8. Resolved decisions

All open questions from v0.1 are now closed:

| # | Question | Decision |
|---|---|---|
| 1 | Should the web app use the File System Access API? | **Yes.** Supported on Chrome/Edge. Safari falls back to manual download/upload. |
| 2 | UUID or incrementing integer for `id`? | **UUID v4.** Prevents collisions when merging independently maintained files. |
| 3 | Tags: predefined list or free-form? | **Suggested-tags dropdown** in v2. Free-form input with existing-tag suggestions to prevent duplicates. Not in v1 scope. |
| 4 | Conflict resolution on import? | **Last-write-wins** based on `updated_at` per shortcut `id`. |
| 5 | Menu bar / tray widget in v1? | **Deferred to v2.** |

---

## 9. Suggested tech stack

| Platform | Language / Framework | Rationale |
|---|---|---|
| macOS | Swift + SwiftUI | Native look, `UserDefaults` for prefs, global hotkey via `NSEvent` |
| Windows | C# + WinUI 3 | Native look, Registry for prefs, global hotkey via `RegisterHotKey` |
| Web | React + TypeScript | File System Access API, `IndexedDB` for file handle, same JSON model |
| Shared schema | JSON Schema (draft-07) | Single source of truth for validation across all three platforms |

For the web app, no backend is needed. All state lives in memory loaded from the chosen file and is written back to that same file handle on every save.

---

## 10. Web implementation guide

This section captures all web-specific decisions and is the primary reference for building the web version of Stashcut in v1.

### 10.1 UI layout

**Two-panel layout.** Fixed for v1 — no layout toggle.

```
┌──────────────────┬────────────────────────────────────┐
│   App sidebar    │        Shortcut list panel          │
│   (240px fixed)  │        (fills remaining width)      │
│                  │                                     │
│  + Add app       │  [Search...]      [Mac|Windows|All] │
│  ─────────────   │  ─────────────────────────────────  │
│  ★ Chrome        │  ★ Reopen last closed tab           │
│    VS Code       │     ⌘⇧T  /  Ctrl+Shift+T           │
│    Figma         │                                     │
│    Slack         │    Focus address bar                │
│                  │     ⌘L  /  Ctrl+L                   │
│                  │                                     │
│                  │  [+ Add shortcut]                   │
└──────────────────┴────────────────────────────────────┘
```

- The sidebar lists all apps sorted by `sort_order`. Active app is highlighted.
- The right panel shows shortcuts for the selected app, or search results across all apps when a search query is active.
- The **Add shortcut** form opens as an inline panel or modal overlay within the right panel — not a separate route.
- No third detail panel in v1. Editing a shortcut opens the same Add form pre-filled.

### 10.2 First launch experience

When the user opens Stashcut with no file loaded (fresh browser session or no `IndexedDB` handle on Chrome/Edge), show a **welcome screen** instead of the main two-panel layout.

Welcome screen contains:
- Stashcut logo and one-line tagline.
- Two primary action buttons:
  - **Create new file** — opens the File System Access API save picker (Chrome/Edge) or initialises an in-memory library (Safari, file saved on first export).
  - **Open existing file** — opens the File System Access API open picker to load a previously saved `shortcuts.json`.

After either action succeeds, transition directly into the main two-panel layout. Do not show the welcome screen again for that session.

On Safari, where no persistent file handle is possible, the welcome screen persists the in-memory state for the session. A persistent banner is shown reminding the user to export before closing the tab.

### 10.3 Key combo entry (Add / Edit form)

Each OS key combo field (Mac and Windows) has two input modes that coexist in the same field:

**Record mode**
- User clicks the **Record** button (keyboard icon).
- The field enters a listening state ("Press keys now…").
- The app captures `keydown` events using `event.preventDefault()` to suppress browser defaults during recording.
- When the user releases all keys, the combo is formatted into canonical text (`Cmd + Shift + T`) and written into both `keys` and `keys_display`.
- Recording stops automatically after capture, or on `Escape`.

**Manual text mode**
- The text input is always editable directly alongside the Record button.
- The user may type the combo in plain text (e.g. `Ctrl + Shift + T`).
- `keys_display` is set to the same value as `keys` when entered manually, unless the user edits `keys_display` separately (advanced).

**Same on both platforms checkbox**
- A checkbox below both fields: **Same key combo on both platforms**.
- When checked, the Windows field mirrors the Mac field (or vice versa, whichever was filled last) and becomes read-only.
- This writes `keys_by_os.both` and removes any separate `mac`/`windows` entries.

### 10.4 App icons

**Built-in icon slugs (v1 set)**

The following slugs are supported in v1. Each maps to a bundled SVG icon:

| Slug | App |
|---|---|
| `chrome` | Google Chrome |
| `firefox` | Mozilla Firefox |
| `safari` | Apple Safari |
| `vscode` | Visual Studio Code |
| `figma` | Figma |
| `slack` | Slack |
| `notion` | Notion |
| `terminal` | Terminal / Command Prompt |
| `finder` | Finder |
| `excel` | Microsoft Excel |
| `word` | Microsoft Word |
| `gmail` | Gmail |
| `github` | GitHub |

**Custom icon upload fallback**

If the user creates an app whose name does not match any built-in slug, they are offered an **Upload icon** button in the Add App form. Accepted formats: PNG, JPG, SVG. Max size: 256 KB. The image is resized to 32×32px and stored as a base64 string in the `icon` field of the app object.

**Generated avatar fallback**

If neither a slug match nor an uploaded icon is present, display a coloured circle with the first letter of the app name (e.g. "N" for Notepad). Colour is deterministically derived from the app name string so it is stable across sessions.

### 10.5 File handling on web

| Scenario | Chrome / Edge | Safari |
|---|---|---|
| Create new file | File System Access API save picker → writes immediately | In-memory only. Persistent export banner shown. |
| Open existing file | File System Access API open picker → file handle stored in `IndexedDB` | File picker → loaded into memory for session only. |
| Save on change | Writes directly to the open file handle automatically | No auto-save. Export button saves a download. |
| Session restore | Re-requests permission to the stored `IndexedDB` handle on page load | No restore. User must re-open the file each session. |
| Export (all browsers) | Available in Settings → Export as a `.json` download | Primary save mechanism. |

### 10.6 State management

All application state lives in a single in-memory store (React context or Zustand). The store shape mirrors the JSON file structure directly:

```typescript
interface StashcutStore {
  fileHandle: FileSystemFileHandle | null  // Chrome/Edge only
  filePath: string | null                  // display only
  data: StashcutFile                       // the full in-memory JSON
  selectedAppId: string | null
  osFilter: 'mac' | 'windows' | 'all'     // default: 'all'
  searchQuery: string
  isDirty: boolean                         // unsaved changes flag (Safari)
}
```

On every mutation (add, edit, delete, reorder, favorite), the store writes the updated `data` back to the file handle immediately (Chrome/Edge). On Safari, `isDirty` is set to `true` and the export banner prompts the user to save manually.

### 10.7 Tech stack (web)

| Concern | Choice | Notes |
|---|---|---|
| Framework | React 18 + TypeScript | |
| State | Zustand | Lightweight, no boilerplate |
| Styling | Tailwind CSS | Utility-first, consistent spacing |
| Drag and drop | `@dnd-kit/core` | Accessible, works with keyboard |
| File I/O | File System Access API + fallback download | Native browser API |
| Icons (built-in) | Bundled SVGs | No external CDN dependency |
| Schema validation | `zod` | Runtime JSON validation on file load |
| Build tool | Vite | Fast HMR for development |
| Testing | Vitest + React Testing Library | Unit and component tests |

### 10.8 Component map

| Component | Responsibility |
|---|---|
| `WelcomeScreen` | First launch — Create / Open file options |
| `AppSidebar` | Lists apps, handles app selection, Add App button |
| `AppIcon` | Renders built-in slug, uploaded image, or initial avatar |
| `ShortcutListPanel` | Toolbar (search, OS filter), shortcut rows, Add Shortcut button |
| `ShortcutRow` | Single shortcut row with star, key badges, description, drag handle |
| `KeyBadge` | Renders a single key combo (e.g. `⌘⇧T`) as a styled pill |
| `ShortcutForm` | Add / Edit form with Record key input and manual text fallback |
| `KeyRecorder` | Isolated key capture component used inside ShortcutForm |
| `SearchBar` | Controlled input, fires filter against store |
| `OsFilterToggle` | Mac / Windows / All segmented control |
| `SettingsPanel` | File path display, Change file, Export, Import |
| `ExportBanner` | Safari-only persistent banner with Export shortcut |

---

*End of specification v0.4*
