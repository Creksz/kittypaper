package bootstrap

import (
	"os"

	"kittypaper/internal/adapters/infra/config"
	"kittypaper/internal/adapters/infra/fs"
	infraKitty "kittypaper/internal/adapters/infra/kitty"
	"kittypaper/internal/app/service"
)

type Container struct {
	Settings         *config.Settings
	Repo             *fs.Repository
	Writer           *infraKitty.Writer
	WallpaperService *service.WallpaperService
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

	return &Container{
		Settings: &settingsCopy,
		Repo:     repo,
		Writer:   writer,
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
