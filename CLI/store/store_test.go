package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stashcut/cli/model"
)

// --- helpers ---

func makeShortcut(id string, updatedAt time.Time) model.Shortcut {
	return model.Shortcut{
		ID:          id,
		Description: "desc-" + id,
		UpdatedAt:   updatedAt,
		KeysByOS:    map[string]model.KeysForOS{},
		Tags:        []string{},
	}
}

func makeApp(id string, shortcuts ...model.Shortcut) model.App {
	return model.App{
		ID:   id,
		Name: "App-" + id,
		Groups: []model.Group{
			{Name: "Uncategorized", Shortcuts: shortcuts},
		},
	}
}

func makeFile(apps ...model.App) *model.ShortcutFile {
	return &model.ShortcutFile{
		Version: "1",
		Meta:    model.Meta{},
		Apps:    apps,
	}
}

// writeTempJSON marshals v to a temp file and returns its path.
func writeTempJSON(t *testing.T, v any) string {
	t.Helper()
	f, err := os.CreateTemp("", "stashcut_test_*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(v); err != nil {
		t.Fatal(err)
	}
	return f.Name()
}

// --- Load ---

func TestLoad_FileNotFound(t *testing.T) {
	sf, err := Load("/tmp/does_not_exist_stashcut_xyzzy.json")
	if err != nil {
		t.Fatalf("expected nil error for missing file, got %v", err)
	}
	if sf != nil {
		t.Fatal("expected nil ShortcutFile for missing file")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	f, err := os.CreateTemp("", "stashcut_test_*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString("not valid json{{{")
	f.Close()

	_, err = Load(f.Name())
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestLoad_NestedFormat(t *testing.T) {
	sf := makeFile(makeApp("a1", makeShortcut("s1", time.Now())))
	tmpFile := writeTempJSON(t, sf)
	defer os.Remove(tmpFile)

	loaded, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected non-nil ShortcutFile")
	}
	if len(loaded.Apps) != 1 {
		t.Errorf("expected 1 app, got %d", len(loaded.Apps))
	}
	if loaded.Apps[0].ID != "a1" {
		t.Errorf("expected app ID a1, got %q", loaded.Apps[0].ID)
	}
}

func TestLoad_EnsuresUncategorizedOnLoad(t *testing.T) {
	// File with an app that has only a custom group — Load should prepend Uncategorized.
	sf := &model.ShortcutFile{
		Version: "1",
		Apps: []model.App{
			{
				ID:     "a1",
				Name:   "App1",
				Groups: []model.Group{{Name: "Custom", Shortcuts: []model.Shortcut{}}},
			},
		},
	}
	tmpFile := writeTempJSON(t, sf)
	defer os.Remove(tmpFile)

	loaded, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded.Apps[0].Groups[0].Name != "Uncategorized" {
		t.Errorf("expected Uncategorized prepended, got %q", loaded.Apps[0].Groups[0].Name)
	}
	if len(loaded.Apps[0].Groups) != 2 {
		t.Errorf("expected 2 groups after prepend, got %d", len(loaded.Apps[0].Groups))
	}
}

func TestLoad_LegacyFormat(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	legacy := legacyShortcutFile{
		Version: "1",
		Apps: []legacyApp{
			{ID: "app1", Name: "App One", SortOrder: 0, CreatedAt: now},
		},
		Shortcuts: []legacyShortcut{
			{
				ID:          "sc1",
				AppID:       "app1",
				Description: "Shortcut One",
				KeysByOS:    map[string]model.KeysForOS{"mac": {Keys: "cmd+t", KeysDisplay: "⌘T"}},
				SortOrder:   0,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		},
	}
	tmpFile := writeTempJSON(t, legacy)
	defer os.Remove(tmpFile)

	sf, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sf.Apps) != 1 {
		t.Fatalf("expected 1 app, got %d", len(sf.Apps))
	}
	if sf.Apps[0].Groups[0].Name != "Uncategorized" {
		t.Errorf("expected Uncategorized group, got %q", sf.Apps[0].Groups[0].Name)
	}
	if len(sf.Apps[0].Groups[0].Shortcuts) != 1 {
		t.Errorf("expected 1 shortcut, got %d", len(sf.Apps[0].Groups[0].Shortcuts))
	}
	if sf.Apps[0].Groups[0].Shortcuts[0].ID != "sc1" {
		t.Errorf("expected shortcut ID sc1, got %q", sf.Apps[0].Groups[0].Shortcuts[0].ID)
	}
}

func TestLoad_LegacyFormat_IgnoresUnknownApp(t *testing.T) {
	now := time.Now()
	legacy := legacyShortcutFile{
		Version: "1",
		Apps:    []legacyApp{{ID: "app1", Name: "App One", CreatedAt: now}},
		Shortcuts: []legacyShortcut{
			{ID: "sc1", AppID: "unknown-app", Description: "orphan", CreatedAt: now, UpdatedAt: now},
		},
	}
	tmpFile := writeTempJSON(t, legacy)
	defer os.Remove(tmpFile)

	sf, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sf.Apps[0].Groups[0].Shortcuts) != 0 {
		t.Errorf("expected orphan shortcut to be dropped, got %d", len(sf.Apps[0].Groups[0].Shortcuts))
	}
}

// --- Save ---

func TestSave_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shortcuts.json")

	sc := makeShortcut("s1", time.Now())
	original := makeFile(makeApp("a1", sc))

	if err := Save(path, original); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load after Save failed: %v", err)
	}
	if len(loaded.Apps) != 1 {
		t.Errorf("expected 1 app, got %d", len(loaded.Apps))
	}
	if loaded.Apps[0].Groups[0].Shortcuts[0].ID != "s1" {
		t.Errorf("expected shortcut ID s1, got %q", loaded.Apps[0].Groups[0].Shortcuts[0].ID)
	}
}

func TestSave_CreatesIntermediateDirectories(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "deep", "shortcuts.json")

	if err := Save(path, makeFile()); err != nil {
		t.Fatalf("expected Save to create intermediate directories: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file to exist after Save: %v", err)
	}
}

func TestSave_UpdatesUpdatedAt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shortcuts.json")

	sf := makeFile()
	before := time.Now()
	if err := Save(path, sf); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if sf.Meta.UpdatedAt.Before(before) {
		t.Error("expected Meta.UpdatedAt to be set to current time by Save")
	}
}

// --- New ---

func TestNew(t *testing.T) {
	sf := New()
	if sf == nil {
		t.Fatal("expected non-nil ShortcutFile")
	}
	if sf.Version != "1" {
		t.Errorf("expected version 1, got %q", sf.Version)
	}
	if sf.Apps == nil {
		t.Error("expected non-nil Apps slice")
	}
	if len(sf.Apps) != 0 {
		t.Errorf("expected empty Apps, got %d", len(sf.Apps))
	}
	if sf.Meta.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
	if sf.Meta.UpdatedAt.IsZero() {
		t.Error("expected non-zero UpdatedAt")
	}
}

// --- Merge ---

func TestMerge_NewApp(t *testing.T) {
	base := makeFile(makeApp("a1"))
	incoming := makeFile(makeApp("a2"))

	Merge(base, incoming)

	if len(base.Apps) != 2 {
		t.Errorf("expected 2 apps after merge, got %d", len(base.Apps))
	}
}

func TestMerge_NewShortcutInExistingGroup(t *testing.T) {
	s1 := makeShortcut("s1", time.Now())
	s2 := makeShortcut("s2", time.Now())

	base := makeFile(makeApp("a1", s1))
	incoming := makeFile(makeApp("a1", s2))

	Merge(base, incoming)

	shortcuts := base.Apps[0].Groups[0].Shortcuts
	if len(shortcuts) != 2 {
		t.Errorf("expected 2 shortcuts after merge, got %d", len(shortcuts))
	}
}

func TestMerge_LastWriteWins_IncomingNewer(t *testing.T) {
	old := time.Now().Add(-time.Hour)
	newer := time.Now()

	s := makeShortcut("s1", old)
	s.Description = "old description"

	sNew := makeShortcut("s1", newer)
	sNew.Description = "new description"

	base := makeFile(makeApp("a1", s))
	incoming := makeFile(makeApp("a1", sNew))

	Merge(base, incoming)

	got := base.Apps[0].Groups[0].Shortcuts[0].Description
	if got != "new description" {
		t.Errorf("expected newer incoming shortcut to win, got %q", got)
	}
}

func TestMerge_LastWriteWins_BaseNewer(t *testing.T) {
	old := time.Now().Add(-time.Hour)
	newer := time.Now()

	s := makeShortcut("s1", newer)
	s.Description = "base description"

	sOld := makeShortcut("s1", old)
	sOld.Description = "old incoming"

	base := makeFile(makeApp("a1", s))
	incoming := makeFile(makeApp("a1", sOld))

	Merge(base, incoming)

	got := base.Apps[0].Groups[0].Shortcuts[0].Description
	if got != "base description" {
		t.Errorf("expected base shortcut to win when newer, got %q", got)
	}
}

func TestMerge_NewGroup(t *testing.T) {
	base := makeFile(makeApp("a1"))
	incoming := &model.ShortcutFile{
		Version: "1",
		Apps: []model.App{
			{
				ID:   "a1",
				Name: "App1",
				Groups: []model.Group{
					{Name: "NewGroup", Shortcuts: []model.Shortcut{makeShortcut("s1", time.Now())}},
				},
			},
		},
	}

	Merge(base, incoming)

	found := false
	for _, g := range base.Apps[0].Groups {
		if g.Name == "NewGroup" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected NewGroup to be added to base app after merge")
	}
}

func TestMerge_DuplicateAppNotAdded(t *testing.T) {
	base := makeFile(makeApp("a1"))
	incoming := makeFile(makeApp("a1"))

	Merge(base, incoming)

	if len(base.Apps) != 1 {
		t.Errorf("expected 1 app (no duplicate), got %d", len(base.Apps))
	}
}

// --- FindShortcut ---

func TestFindShortcut_Found(t *testing.T) {
	sc := makeShortcut("target", time.Now())
	sf := makeFile(makeApp("a1", sc))

	ai, gi, si, found := FindShortcut(sf, "target")
	if !found {
		t.Fatal("expected shortcut to be found")
	}
	if ai != 0 || gi != 0 || si != 0 {
		t.Errorf("unexpected indices: ai=%d gi=%d si=%d", ai, gi, si)
	}
}

func TestFindShortcut_NotFound(t *testing.T) {
	sf := makeFile(makeApp("a1", makeShortcut("s1", time.Now())))

	_, _, _, found := FindShortcut(sf, "nonexistent")
	if found {
		t.Error("expected shortcut not to be found")
	}
}

func TestFindShortcut_MultipleAppsAndGroups(t *testing.T) {
	sc1 := makeShortcut("first", time.Now())
	sc2 := makeShortcut("second", time.Now())
	a1 := makeApp("a1", sc1)
	a2 := makeApp("a2", sc2)
	sf := makeFile(a1, a2)

	ai, gi, si, found := FindShortcut(sf, "second")
	if !found {
		t.Fatal("expected second shortcut to be found")
	}
	if ai != 1 {
		t.Errorf("expected app index 1, got %d", ai)
	}
	if gi != 0 || si != 0 {
		t.Errorf("unexpected indices gi=%d si=%d", gi, si)
	}
}

// --- FindGroup ---

func TestFindGroup_Found(t *testing.T) {
	app := makeApp("a1")
	app.Groups = append(app.Groups, model.Group{Name: "MyGroup"})

	idx := FindGroup(&app, "MyGroup")
	if idx < 0 {
		t.Error("expected FindGroup to find MyGroup")
	}
	if app.Groups[idx].Name != "MyGroup" {
		t.Errorf("unexpected group name %q", app.Groups[idx].Name)
	}
}

func TestFindGroup_NotFound(t *testing.T) {
	app := makeApp("a1")

	idx := FindGroup(&app, "Ghost")
	if idx != -1 {
		t.Errorf("expected -1 for missing group, got %d", idx)
	}
}

func TestFindGroup_EmptyApp(t *testing.T) {
	app := model.App{ID: "a1", Name: "Empty", Groups: []model.Group{}}

	if idx := FindGroup(&app, "Anything"); idx != -1 {
		t.Errorf("expected -1 for empty group list, got %d", idx)
	}
}

// --- TotalShortcuts ---

func TestTotalShortcuts(t *testing.T) {
	a1 := makeApp("a1", makeShortcut("s1", time.Now()), makeShortcut("s2", time.Now()))
	a2 := makeApp("a2", makeShortcut("s3", time.Now()))
	sf := makeFile(a1, a2)

	if n := TotalShortcuts(sf); n != 3 {
		t.Errorf("expected 3 total shortcuts, got %d", n)
	}
}

func TestTotalShortcuts_Empty(t *testing.T) {
	if n := TotalShortcuts(makeFile()); n != 0 {
		t.Errorf("expected 0 for empty file, got %d", n)
	}
}

func TestTotalShortcuts_MultipleGroups(t *testing.T) {
	app := model.App{
		ID:   "a1",
		Name: "App1",
		Groups: []model.Group{
			{Name: "Uncategorized", Shortcuts: []model.Shortcut{makeShortcut("s1", time.Now())}},
			{Name: "Group2", Shortcuts: []model.Shortcut{makeShortcut("s2", time.Now()), makeShortcut("s3", time.Now())}},
		},
	}
	sf := makeFile(app)

	if n := TotalShortcuts(sf); n != 3 {
		t.Errorf("expected 3 shortcuts across groups, got %d", n)
	}
}

// --- ensureUncategorized ---

func TestEnsureUncategorized_AddsWhenMissing(t *testing.T) {
	app := model.App{
		ID:     "a1",
		Name:   "App1",
		Groups: []model.Group{{Name: "Custom"}},
	}
	ensureUncategorized(&app)

	if app.Groups[0].Name != "Uncategorized" {
		t.Errorf("expected Uncategorized prepended, got %q", app.Groups[0].Name)
	}
	if len(app.Groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(app.Groups))
	}
}

func TestEnsureUncategorized_NoOpWhenExists(t *testing.T) {
	app := makeApp("a1") // already has Uncategorized
	count := len(app.Groups)
	ensureUncategorized(&app)

	if len(app.Groups) != count {
		t.Errorf("expected group count unchanged (%d), got %d", count, len(app.Groups))
	}
}
