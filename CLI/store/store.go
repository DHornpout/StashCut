package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/stashcut/cli/config"
	"github.com/stashcut/cli/model"
)

// Load reads the shortcuts file from the given path.
func Load(path string) (*model.ShortcutFile, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var sf model.ShortcutFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, err
	}
	return &sf, nil
}

// Save writes the shortcuts file to the given path.
func Save(path string, sf *model.ShortcutFile) error {
	sf.Meta.UpdatedAt = time.Now()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// New creates an empty ShortcutFile with initialized defaults.
func New() *model.ShortcutFile {
	now := time.Now()
	return &model.ShortcutFile{
		Version: "1",
		Meta: model.Meta{
			CreatedAt:  now,
			UpdatedAt:  now,
			AppVersion: config.AppVersion,
		},
		Apps:      []model.App{},
		Shortcuts: []model.Shortcut{},
	}
}

// Merge combines incoming into base using last-write-wins on UpdatedAt for shortcuts.
// Apps in incoming that are not in base are appended. Shortcuts in incoming that are
// newer (by UpdatedAt) than the matching base shortcut replace it; new shortcuts are added.
func Merge(base, incoming *model.ShortcutFile) {
	// Index existing apps by ID
	appIdx := make(map[string]struct{}, len(base.Apps))
	for _, a := range base.Apps {
		appIdx[a.ID] = struct{}{}
	}
	for _, a := range incoming.Apps {
		if _, exists := appIdx[a.ID]; !exists {
			a.SortOrder = len(base.Apps)
			base.Apps = append(base.Apps, a)
		}
	}

	// Index existing shortcuts by ID
	scIdx := make(map[string]int, len(base.Shortcuts))
	for i, s := range base.Shortcuts {
		scIdx[s.ID] = i
	}
	for _, s := range incoming.Shortcuts {
		if idx, exists := scIdx[s.ID]; exists {
			if s.UpdatedAt.After(base.Shortcuts[idx].UpdatedAt) {
				base.Shortcuts[idx] = s
			}
		} else {
			base.Shortcuts = append(base.Shortcuts, s)
		}
	}
}

// ShortcutsForApp returns shortcuts belonging to the given app, sorted by SortOrder.
func ShortcutsForApp(sf *model.ShortcutFile, appID string) []model.Shortcut {
	var result []model.Shortcut
	for _, s := range sf.Shortcuts {
		if s.AppID == appID {
			result = append(result, s)
		}
	}
	return result
}
