package fs

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	domainerr "kittypaper/internal/domain/errors"
	"kittypaper/internal/domain/wallpaper"
	"kittypaper/internal/platform/filex"
	"kittypaper/internal/platform/pathx"
)

var imageExts = map[string]struct{}{
	".png":  {},
	".jpg":  {},
	".jpeg": {},
	".webp": {},
	".gif":  {},
	".bmp":  {},
}

type stateFile struct {
	Path      string    `json:"path"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Repository struct {
	WallpaperDirs []string
	StateFile     string
}

func (r Repository) GetByPath(ctx context.Context, path string) (wallpaper.Item, error) {
	_ = ctx
	abs, err := pathx.Abs(path)
	if err != nil {
		return wallpaper.Item{}, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return wallpaper.Item{}, fmt.Errorf("%w: %s", domainerr.ErrWallpaperNotFound, abs)
		}
		return wallpaper.Item{}, err
	}
	if info.IsDir() {
		return wallpaper.Item{}, fmt.Errorf("%w: %s is a directory", domainerr.ErrInvalidWallpaper, abs)
	}
	if !IsImagePath(abs) {
		return wallpaper.Item{}, fmt.Errorf("%w: unsupported image type %s", domainerr.ErrInvalidWallpaper, abs)
	}
	return wallpaper.Item{
		ID:        wallpaper.ID(abs),
		Path:      abs,
		UpdatedAt: info.ModTime(),
	}, nil
}

func (r Repository) List(ctx context.Context) ([]wallpaper.Item, error) {
	_ = ctx
	seen := make(map[string]struct{})
	var items []wallpaper.Item
	for _, dir := range r.WallpaperDirs {
		_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if !IsImagePath(path) {
				return nil
			}
			abs, err := filepath.Abs(path)
			if err != nil {
				return nil
			}
			if _, ok := seen[abs]; ok {
				return nil
			}
			seen[abs] = struct{}{}
			info, err := d.Info()
			mod := time.Time{}
			if err == nil {
				mod = info.ModTime()
			}
			items = append(items, wallpaper.Item{
				ID:        wallpaper.ID(abs),
				Path:      abs,
				UpdatedAt: mod,
			})
			return nil
		})
	}
	return items, nil
}

func (r Repository) SaveActive(ctx context.Context, item wallpaper.Item) error {
	_ = ctx
	payload, err := json.MarshalIndent(stateFile{Path: item.Path, UpdatedAt: time.Now()}, "", "  ")
	if err != nil {
		return err
	}
	return filex.WriteFileAtomic(r.StateFile, append(payload, '\n'), 0o644)
}

func (r Repository) GetActive(ctx context.Context) (wallpaper.Item, error) {
	_ = ctx
	raw, err := os.ReadFile(r.StateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return wallpaper.Item{}, fmt.Errorf("%w: no active wallpaper", domainerr.ErrWallpaperNotFound)
		}
		return wallpaper.Item{}, err
	}
	var st stateFile
	if err := json.Unmarshal(raw, &st); err != nil {
		return wallpaper.Item{}, err
	}
	if st.Path == "" {
		return wallpaper.Item{}, fmt.Errorf("%w: no active wallpaper", domainerr.ErrWallpaperNotFound)
	}
	return wallpaper.Item{ID: wallpaper.ID(st.Path), Path: st.Path, UpdatedAt: st.UpdatedAt}, nil
}

func IsImagePath(path string) bool {
	_, ok := imageExts[strings.ToLower(filepath.Ext(path))]
	return ok
}
