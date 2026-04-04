package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultDataPath(t *testing.T) {
	path, err := DefaultDataPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty default data path")
	}
	if filepath.Base(path) != "shortcuts.json" {
		t.Errorf("expected filename shortcuts.json, got %q", filepath.Base(path))
	}
}

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	cfg := &Config{DataFilePath: "/some/test/path.json"}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.DataFilePath != cfg.DataFilePath {
		t.Errorf("expected %q, got %q", cfg.DataFilePath, loaded.DataFilePath)
	}
}

func TestLoad_CreatesDefaultWhenMissing(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.DataFilePath == "" {
		t.Error("expected non-empty DataFilePath in default config")
	}

	// Config file should now exist on disk.
	expectedPath := filepath.Join(tmpHome, ".config", "stashcut", "config.json")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("expected config file to be created at %s", expectedPath)
	}
}

func TestLoad_CreatesDirectories(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Confirm the .config/stashcut dir does not exist yet.
	cfgDir := filepath.Join(tmpHome, ".config", "stashcut")
	if _, err := os.Stat(cfgDir); !os.IsNotExist(err) {
		t.Skip("config dir unexpectedly already exists")
	}

	if _, err := Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if _, err := os.Stat(cfgDir); os.IsNotExist(err) {
		t.Errorf("expected config directory to be created at %s", cfgDir)
	}
}

func TestSave_OverwritesExisting(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	first := &Config{DataFilePath: "/first/path.json"}
	if err := Save(first); err != nil {
		t.Fatalf("first Save failed: %v", err)
	}

	second := &Config{DataFilePath: "/second/path.json"}
	if err := Save(second); err != nil {
		t.Fatalf("second Save failed: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.DataFilePath != second.DataFilePath {
		t.Errorf("expected %q after overwrite, got %q", second.DataFilePath, loaded.DataFilePath)
	}
}
