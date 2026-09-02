package bootstrap

import (
	"context"
	"os"
	"path/filepath"

	"kittypaper/internal/adapters/infra/config"
	"kittypaper/internal/adapters/infra/download"
	"kittypaper/internal/adapters/infra/fs"
	infraKitty "kittypaper/internal/adapters/infra/kitty"
	"kittypaper/internal/adapters/infra/library"
	"kittypaper/internal/adapters/infra/wallhaven"
	"kittypaper/internal/app/dto"
	"kittypaper/internal/app/service"
	"kittypaper/internal/domain/online"
)

type Container struct {
	Settings         *config.Settings
	Repo             *fs.Repository
	Writer           *infraKitty.Writer
	Library          *library.Store
	WallpaperService *service.WallpaperService
	OnlineService    *service.OnlineService
}

func NewContainer(settings config.Settings) *Container {
	settingsCopy := settings
	repo := &fs.Repository{
		WallpaperDirs: append([]string(nil), settingsCopy.WallpaperDirs...),
		StateFile:     settingsCopy.StateFile,
	}
	inspector := infraKitty.Inspector{
		KittyConfPath:     settingsCopy.KittyConfPath,
		GeneratedConfPath: settingsCopy.GeneratedConfPath,
	}
	writer := &infraKitty.Writer{
		GeneratedConfPath: settingsCopy.GeneratedConfPath,
		Tint:              settingsCopy.BackgroundTint,
		Opacity:           settingsCopy.BackgroundOpacity,
	}
	reloader := infraKitty.Reloader{}

	libPath, _ := settingsCopy.LibraryFilePath()
	if libPath == "" {
		libPath = filepath.Join(filepath.Dir(settingsCopy.StateFile), "library.json")
	}
	libStore := &library.Store{Path: libPath}

	onlineSvc := &service.OnlineService{
		Wallhaven:  wallhaven.Client{APIKey: settingsCopy.WallhavenAPIKey},
		Downloader: download.Downloader{},
	}

	return &Container{
		Settings:      &settingsCopy,
		Repo:          repo,
		Writer:        writer,
		Library:       libStore,
		OnlineService: onlineSvc,
		WallpaperService: service.NewWallpaperService(service.WallpaperDeps{
			Repo:      repo,
			Writer:    writer,
			Inspector: inspector,
			Reloader:  reloader,
			Live:      reloader,
			Init: service.InitHooks{
				KittyConfPath:     settingsCopy.KittyConfPath,
				GeneratedConfPath: settingsCopy.GeneratedConfPath,
				EnsureGenerated: func() error {
					if err := os.MkdirAll(settingsCopy.CacheDir, 0o755); err != nil {
						return err
					}
					return infraKitty.EnsureGenerated(settingsCopy.GeneratedConfPath)
				},
				WriteInclude: func() error {
					return infraKitty.AppendInclude(settingsCopy.KittyConfPath, settingsCopy.GeneratedConfPath)
				},
				HasInclude: func() (bool, error) {
					return infraKitty.HasInclude(settingsCopy.KittyConfPath, settingsCopy.GeneratedConfPath)
				},
			},
		}),
	}
}

func (c *Container) SetWallpaperDirs(dirs []string) error {
	c.Settings.WallpaperDirs = append([]string(nil), dirs...)
	c.Repo.WallpaperDirs = append([]string(nil), dirs...)
	return config.Save(*c.Settings)
}

func (c *Container) SetAppearance(tint, opacity float64) error {
	c.Settings.BackgroundTint = tint
	c.Settings.BackgroundOpacity = opacity
	c.Writer.Tint = tint
	c.Writer.Opacity = opacity
	return config.Save(*c.Settings)
}

func (c *Container) DownloadDir() string {
	if len(c.Settings.WallpaperDirs) > 0 {
		return c.Settings.WallpaperDirs[0]
	}
	return ""
}

func (c *Container) SearchOnline(ctx context.Context, query string, page int) (online.SearchResult, error) {
	if c.OnlineService == nil {
		return online.SearchResult{}, nil
	}
	return c.OnlineService.Search(ctx, query, page)
}

func (c *Container) SaveConfig() error {
	return config.Save(*c.Settings)
}
func (c *Container) PreviewBackground(path string, tint, opacity float64) error {
	return c.WallpaperService.PreviewBackground(context.Background(), dto.PreviewRequest{
		Path:    path,
		Tint:    tint,
		Opacity: opacity,
	})
}
