package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"kittypaper/internal/domain/kitty"
	"kittypaper/internal/platform/filex"
	"kittypaper/internal/platform/pathx"
)

const (
	DefaultBackgroundTint    = 0.95
	DefaultBackgroundOpacity = 1.0
)

type Settings struct {
	KittyConfPath     string
	GeneratedConfPath string
	ReloadMethod      kitty.ReloadMethod
	WallpaperDirs     []string
	CacheDir          string
	StateFile         string
	ConfigPath        string
	BackgroundTint    float64
	BackgroundOpacity float64
}

func DefaultSettings() Settings {
	return Settings{
		KittyConfPath:     "~/.config/kitty/kitty.conf",
		GeneratedConfPath: "~/.config/kitty/kittypaper-background.conf",
		ReloadMethod:      kitty.ReloadAuto,
		WallpaperDirs:     []string{"~/Pictures/Wallpapers", "~/wallpaper"},
		CacheDir:          "~/.cache/kittypaper",
		StateFile:         "~/.local/state/kittypaper/state.json",
		BackgroundTint:    DefaultBackgroundTint,
		BackgroundOpacity: DefaultBackgroundOpacity,
	}
}

func Load(explicitPath string) (Settings, error) {
	settings := DefaultSettings()
	path, err := resolveConfigPath(explicitPath)
	if err != nil {
		return Settings{}, err
	}
	if path == "" {
		return ExpandSettings(settings)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return Settings{}, fmt.Errorf("read config %s: %w", path, err)
	}
	if err := applyYAML(raw, &settings); err != nil {
		return Settings{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	settings.ConfigPath = path
	return ExpandSettings(settings)
}

func ExpandSettings(settings Settings) (Settings, error) {
	var err error
	if settings.KittyConfPath, err = pathx.Abs(settings.KittyConfPath); err != nil {
		return Settings{}, err
	}
	if settings.GeneratedConfPath, err = pathx.Abs(settings.GeneratedConfPath); err != nil {
		return Settings{}, err
	}
	if settings.CacheDir, err = pathx.Abs(settings.CacheDir); err != nil {
		return Settings{}, err
	}
	if settings.StateFile, err = pathx.Abs(settings.StateFile); err != nil {
		return Settings{}, err
	}
	dirs := make([]string, 0, len(settings.WallpaperDirs))
	for _, dir := range settings.WallpaperDirs {
		abs, err := pathx.Abs(dir)
		if err != nil {
			return Settings{}, err
		}
		if abs != "" {
			dirs = append(dirs, abs)
		}
	}
	settings.WallpaperDirs = dirs
	if settings.ReloadMethod == "" {
		settings.ReloadMethod = kitty.ReloadAuto
	}
	settings.BackgroundTint = clampUnit(settings.BackgroundTint, DefaultBackgroundTint)
	settings.BackgroundOpacity = clampUnit(settings.BackgroundOpacity, DefaultBackgroundOpacity)
	return settings, nil
}

func DefaultConfigFilePath() (string, error) {
	return pathx.Abs(filepath.Join("~", ".config", "kittypaper", "config.yaml"))
}

func Save(settings Settings) error {
	path := settings.ConfigPath
	if path == "" {
		var err error
		path, err = DefaultConfigFilePath()
		if err != nil {
			return err
		}
		settings.ConfigPath = path
	}

	expanded, err := ExpandSettings(settings)
	if err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString("kitty_conf_path: " + strconv.Quote(expanded.KittyConfPath) + "\n")
	b.WriteString("generated_conf_path: " + strconv.Quote(expanded.GeneratedConfPath) + "\n")
	b.WriteString("reload_method: " + strconv.Quote(string(expanded.ReloadMethod)) + "\n")
	b.WriteString("background_tint: " + formatFloat(expanded.BackgroundTint) + "\n")
	b.WriteString("background_opacity: " + formatFloat(expanded.BackgroundOpacity) + "\n")
	b.WriteString("cache_dir: " + strconv.Quote(expanded.CacheDir) + "\n")
	b.WriteString("state_file: " + strconv.Quote(expanded.StateFile) + "\n")
	b.WriteString("wallpaper_dirs:\n")
	for _, dir := range expanded.WallpaperDirs {
		b.WriteString("  - " + strconv.Quote(dir) + "\n")
	}
	return filex.WriteFileAtomic(path, []byte(b.String()), 0o644)
}

func resolveConfigPath(explicitPath string) (string, error) {
	if explicitPath != "" {
		return pathx.Abs(explicitPath)
	}
	if env := os.Getenv("KITTYPAPER_CONFIG"); env != "" {
		return pathx.Abs(env)
	}
	candidate, err := DefaultConfigFilePath()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(candidate); err == nil {
		return candidate, nil
	}
	return "", nil
}

func applyYAML(raw []byte, settings *Settings) error {
	scanner := bufio.NewScanner(strings.NewReader(string(raw)))
	inDirs := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "- ") && inDirs {
			settings.WallpaperDirs = append(settings.WallpaperDirs, unquote(strings.TrimSpace(line[2:])))
			continue
		}
		inDirs = false
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return fmt.Errorf("invalid line: %s", line)
		}
		key = strings.TrimSpace(key)
		value = unquote(strings.TrimSpace(value))
		switch key {
		case "kitty_conf_path":
			if value != "" {
				settings.KittyConfPath = value
			}
		case "generated_conf_path":
			if value != "" {
				settings.GeneratedConfPath = value
			}
		case "reload_method":
			if value != "" {
				settings.ReloadMethod = kitty.ReloadMethod(value)
			}
		case "cache_dir":
			if value != "" {
				settings.CacheDir = value
			}
		case "state_file":
			if value != "" {
				settings.StateFile = value
			}
		case "background_tint":
			if value != "" {
				f, err := strconv.ParseFloat(value, 64)
				if err != nil {
					return fmt.Errorf("invalid background_tint: %s", value)
				}
				settings.BackgroundTint = f
			}
		case "background_opacity":
			if value != "" {
				f, err := strconv.ParseFloat(value, 64)
				if err != nil {
					return fmt.Errorf("invalid background_opacity: %s", value)
				}
				settings.BackgroundOpacity = f
			}
		case "wallpaper_dirs":
			settings.WallpaperDirs = nil
			inDirs = true
		default:
			return fmt.Errorf("unknown key: %s", key)
		}
	}
	return scanner.Err()
}

func unquote(value string) string {
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}

func clampUnit(v, fallback float64) float64 {
	if v < 0 || v > 1 {
		return fallback
	}
	return v
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
