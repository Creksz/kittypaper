package bootstrap

import (
	"context"
	"errors"
	"fmt"

	"kittypaper/internal/adapters/infra/config"
	"kittypaper/internal/app/dto"
	domainerr "kittypaper/internal/domain/errors"
)

type SetupResult struct {
	ConfigPath      string
	ConfigCreated   bool
	GeneratedConfPath string
	IncludeWritten  bool
	KittyConfPath   string
	WallpaperDirs   []string
}

func Setup(ctx context.Context, explicitConfig string) (SetupResult, error) {
	var (
		settings config.Settings
		created  bool
		err      error
	)
	if explicitConfig != "" {
		settings, err = config.Load(explicitConfig)
		if err != nil {
			return SetupResult{}, err
		}
		if err := config.EnsureRuntimeDirs(settings); err != nil {
			return SetupResult{}, err
		}
	} else {
		settings, created, err = config.EnsureDefaults()
		if err != nil {
			return SetupResult{}, err
		}
	}

	container := NewContainer(settings)
	initResult, err := container.WallpaperService.Init(ctx, dto.InitRequest{WriteInclude: true})
	if err != nil {
		if errors.Is(err, domainerr.ErrKittyConfMissing) {
			return SetupResult{}, fmt.Errorf("kitty config not found — install Kitty first, then run: kittypaper setup")
		}
		return SetupResult{}, err
	}

	return SetupResult{
		ConfigPath:      settings.ConfigPath,
		ConfigCreated:   created,
		GeneratedConfPath: initResult.GeneratedConfPath,
		IncludeWritten:  initResult.IncludeWritten,
		KittyConfPath:   initResult.KittyConfPath,
		WallpaperDirs:   append([]string(nil), settings.WallpaperDirs...),
	}, nil
}
