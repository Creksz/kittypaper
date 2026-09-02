package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed default_config.yaml
var defaultConfigYAML []byte

// EnsureDefaults creates ~/.config/kittypaper/config.yaml and cache/state dirs if missing.
// Returns settings, whether the config file was newly created, and any error.
func EnsureDefaults() (Settings, bool, error) {
	path, err := DefaultConfigFilePath()
	if err != nil {
		return Settings{}, false, err
	}
	if _, err := os.Stat(path); err == nil {
		settings, err := Load("")
		return settings, false, err
	} else if !os.IsNotExist(err) {
		return Settings{}, false, err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return Settings{}, false, err
	}
	if err := os.WriteFile(path, defaultConfigYAML, 0o644); err != nil {
		return Settings{}, false, fmt.Errorf("write default config %s: %w", path, err)
	}

	settings, err := Load(path)
	if err != nil {
		return Settings{}, false, err
	}
	if err := ensureRuntimeDirs(settings); err != nil {
		return Settings{}, true, err
	}
	return settings, true, nil
}

func EnsureRuntimeDirs(settings Settings) error {
	return ensureRuntimeDirs(settings)
}

func ensureRuntimeDirs(settings Settings) error {
	expanded, err := ExpandSettings(settings)
	if err != nil {
		return err
	}
	for _, dir := range []string{expanded.CacheDir, filepath.Dir(expanded.StateFile)} {
		if dir == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	for _, dir := range expanded.WallpaperDirs {
		if dir == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}
