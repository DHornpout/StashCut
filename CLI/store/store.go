package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/stashcut/cli/config"
	"github.com/stashcut/cli/model"
)

// legacyShortcutFile is used to detect and migrate the old flat JSON format.
type legacyShortcutFile struct {
	Version   string           `json:"version"`
	Meta      model.Meta       `json:"meta"`
	Apps      []legacyApp      `json:"apps"`
	Shortcuts []legacyShortcut `json:"shortcuts"`
}

type legacyApp struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Icon      string    `json:"icon"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
}

type legacyShortcut struct {
	ID          string                    `json:"id"`
	AppID       string                    `json:"app_id"`
	Description string                    `json:"description"`
	KeysByOS    map[string]model.KeysForOS `json:"keys_by_os"`
	IsFavorite  bool                       `json:"is_favorite"`
	SortOrder   int                        `json:"sort_order"`
	Tags        []string                   `json:"tags"`
	CreatedAt   time.Time                  `json:"created_at"`
	UpdatedAt   time.Time                  `json:"updated_at"`
}

// Load reads the shortcuts file from the given path, migrating from the flat
// format automatically if needed.
func Load(path string) (*model.ShortcutFile, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Try to detect legacy flat format.
	var legacy legacyShortcutFile
	if err := json.Unmarshal(data, &legacy); err != nil {
		return nil, err
	}
	if len(legacy.Shortcuts) > 0 {
		return migrateFromFlat(legacy), nil
	}

	// Current nested format.
	var sf model.ShortcutFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, err
	}
	for i := range sf.Apps {
		ensureUncategorized(&sf.Apps[i])
	}
	return &sf, nil
}

// migrateFromFlat converts a legacy flat ShortcutFile to the nested model.
func migrateFromFlat(legacy legacyShortcutFile) *model.ShortcutFile {
	appMap := make(map[string]int)
	apps := make([]model.App, len(legacy.Apps))
	for i, la := range legacy.Apps {
		apps[i] = model.App{
			ID:        la.ID,
			Name:      la.Name,
			Icon:      la.Icon,
			SortOrder: la.SortOrder,
			CreatedAt: la.CreatedAt,
			UpdatedAt: la.CreatedAt,
			Groups:    []model.Group{{Name: "Uncategorized", Shortcuts: []model.Shortcut{}}},
		}
		appMap[la.ID] = i
	}

	for _, ls := range legacy.Shortcuts {
		ai, ok := appMap[ls.AppID]
		if !ok {
			continue
		}
		sc := model.Shortcut{
			ID:          ls.ID,
			Description: ls.Description,
			KeysByOS:    ls.KeysByOS,
			IsFavorite:  ls.IsFavorite,
			SortOrder:   ls.SortOrder,
			Tags:        ls.Tags,
			CreatedAt:   ls.CreatedAt,
			UpdatedAt:   ls.UpdatedAt,
		}
		apps[ai].Groups[0].Shortcuts = append(apps[ai].Groups[0].Shortcuts, sc)
	}

	// Sort shortcuts within each group by SortOrder.
	for i := range apps {
		for g := range apps[i].Groups {
			sort.Slice(apps[i].Groups[g].Shortcuts, func(a, b int) bool {
				return apps[i].Groups[g].Shortcuts[a].SortOrder < apps[i].Groups[g].Shortcuts[b].SortOrder
			})
		}
	}

	return &model.ShortcutFile{
		Version: legacy.Version,
		Meta:    legacy.Meta,
		Apps:    apps,
	}
}

// ensureUncategorized prepends an "Uncategorized" group to the app if none exists.
func ensureUncategorized(app *model.App) {
	for _, g := range app.Groups {
		if g.Name == "Uncategorized" {
			return
		}
	}
	app.Groups = append([]model.Group{{Name: "Uncategorized", Shortcuts: []model.Shortcut{}}}, app.Groups...)
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
		Apps: []model.App{},
	}
}

// Merge combines incoming into base using last-write-wins on UpdatedAt for shortcuts.
// Apps in incoming that are not in base are appended. Groups are merged by name.
// Shortcuts within groups are merged by ID using last-write-wins on UpdatedAt.
func Merge(base, incoming *model.ShortcutFile) {
	appIdx := make(map[string]int, len(base.Apps))
	for i, a := range base.Apps {
		appIdx[a.ID] = i
	}

	for _, inApp := range incoming.Apps {
		bi, exists := appIdx[inApp.ID]
		if !exists {
			ensureUncategorized(&inApp)
			inApp.SortOrder = len(base.Apps)
			base.Apps = append(base.Apps, inApp)
			appIdx[inApp.ID] = len(base.Apps) - 1
			continue
		}

		// Merge groups by name.
		groupIdx := make(map[string]int, len(base.Apps[bi].Groups))
		for gi, g := range base.Apps[bi].Groups {
			groupIdx[g.Name] = gi
		}

		for _, inGrp := range inApp.Groups {
			gi, gExists := groupIdx[inGrp.Name]
			if !gExists {
				base.Apps[bi].Groups = append(base.Apps[bi].Groups, model.Group{
					Name:      inGrp.Name,
					Shortcuts: inGrp.Shortcuts,
				})
				groupIdx[inGrp.Name] = len(base.Apps[bi].Groups) - 1
				continue
			}

			// Merge shortcuts by ID within matched group.
			scIdx := make(map[string]int, len(base.Apps[bi].Groups[gi].Shortcuts))
			for si, s := range base.Apps[bi].Groups[gi].Shortcuts {
				scIdx[s.ID] = si
			}
			for _, inSc := range inGrp.Shortcuts {
				if si, scExists := scIdx[inSc.ID]; scExists {
					if inSc.UpdatedAt.After(base.Apps[bi].Groups[gi].Shortcuts[si].UpdatedAt) {
						base.Apps[bi].Groups[gi].Shortcuts[si] = inSc
					}
				} else {
					base.Apps[bi].Groups[gi].Shortcuts = append(base.Apps[bi].Groups[gi].Shortcuts, inSc)
				}
			}
		}
	}
}

// FindShortcut locates a shortcut by ID across all apps and groups.
// Returns (appIndex, groupIndex, shortcutIndex, found).
func FindShortcut(sf *model.ShortcutFile, id string) (int, int, int, bool) {
	for ai := range sf.Apps {
		for gi := range sf.Apps[ai].Groups {
			for si := range sf.Apps[ai].Groups[gi].Shortcuts {
				if sf.Apps[ai].Groups[gi].Shortcuts[si].ID == id {
					return ai, gi, si, true
				}
			}
		}
	}
	return 0, 0, 0, false
}

// FindGroup returns the index of the group with the given name in an app, or -1.
func FindGroup(app *model.App, name string) int {
	for i, g := range app.Groups {
		if g.Name == name {
			return i
		}
	}
	return -1
}

// TotalShortcuts counts all shortcuts across all apps and groups.
func TotalShortcuts(sf *model.ShortcutFile) int {
	n := 0
	for _, a := range sf.Apps {
		for _, g := range a.Groups {
			n += len(g.Shortcuts)
		}
	}
	return n
}
