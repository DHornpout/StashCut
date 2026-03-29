package model

import "time"

// ShortcutFile is the root JSON object shared with the web app.
type ShortcutFile struct {
	Version string `json:"version"`
	Meta    Meta   `json:"meta"`
	Apps    []App  `json:"apps"`
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
	UpdatedAt time.Time `json:"updated_at"`
	Groups    []Group   `json:"groups"`
}

type Group struct {
	Name      string     `json:"name"`
	Shortcuts []Shortcut `json:"shortcuts"`
}

type KeysForOS struct {
	Keys        string `json:"keys"`
	KeysDisplay string `json:"keys_display"`
}

type Shortcut struct {
	ID          string               `json:"id"`
	Description string               `json:"description"`
	KeysByOS    map[string]KeysForOS `json:"keys_by_os"`
	IsFavorite  bool                 `json:"is_favorite"`
	SortOrder   int                  `json:"sort_order"`
	Tags        []string             `json:"tags"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
}
