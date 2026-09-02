package outbound

import (
	"context"

	"kittypaper/internal/domain/kitty"
	"kittypaper/internal/domain/wallpaper"
)

type WallpaperRepository interface {
	GetByPath(ctx context.Context, path string) (wallpaper.Item, error)
	List(ctx context.Context) ([]wallpaper.Item, error)
	SaveActive(ctx context.Context, item wallpaper.Item) error
	GetActive(ctx context.Context) (wallpaper.Item, error)
}

type BackgroundConfigWriter interface {
	WriteBackgroundConfig(ctx context.Context, wallpaperPath string) error
}

type KittyConfigInspector interface {
	EnsureInclude(ctx context.Context) error
}

type KittyReloader interface {
	Reload(ctx context.Context, method kitty.ReloadMethod) error
}

// KittyLiveApplier applies a wallpaper to running Kitty instances immediately.
type KittyLiveApplier interface {
	ApplyImage(ctx context.Context, wallpaperPath string) error
	PreviewBackground(ctx context.Context, wallpaperPath string, tint, opacity float64) error
}

type CacheStore interface {
	Set(ctx context.Context, key string, value []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
}
