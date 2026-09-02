package service

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"kittypaper/internal/app/dto"
	domainerr "kittypaper/internal/domain/errors"
	"kittypaper/internal/domain/kitty"
	"kittypaper/internal/domain/wallpaper"
	"kittypaper/internal/ports/outbound"
)

type WallpaperDeps struct {
	Repo      outbound.WallpaperRepository
	Writer    outbound.BackgroundConfigWriter
	Inspector outbound.KittyConfigInspector
	Reloader  outbound.KittyReloader
	Live      outbound.KittyLiveApplier
	Init      InitHooks
	RandIntn  func(n int) int
}

type InitHooks struct {
	KittyConfPath     string
	GeneratedConfPath string
	EnsureGenerated   func() error
	WriteInclude      func() error
	HasInclude        func() (bool, error)
}

type WallpaperService struct {
	deps WallpaperDeps
}

func NewWallpaperService(deps WallpaperDeps) *WallpaperService {
	if deps.RandIntn == nil {
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		deps.RandIntn = rng.Intn
	}
	return &WallpaperService{deps: deps}
}

func (s *WallpaperService) SetWallpaper(ctx context.Context, req dto.SetWallpaperRequest) (dto.ApplyResult, error) {
	if req.Path == "" {
		return dto.ApplyResult{}, domainerr.ErrInvalidWallpaper
	}
	if s.deps.Repo == nil || s.deps.Inspector == nil || s.deps.Writer == nil || s.deps.Reloader == nil {
		return dto.ApplyResult{}, domainerr.ErrNotImplemented
	}

	item, err := s.deps.Repo.GetByPath(ctx, req.Path)
	if err != nil {
		return dto.ApplyResult{}, err
	}

	if err := s.ensureIncludeReady(ctx); err != nil {
		return dto.ApplyResult{}, err
	}
	if err := s.deps.Writer.WriteBackgroundConfig(ctx, item.Path); err != nil {
		return dto.ApplyResult{}, err
	}
	if err := s.deps.Repo.SaveActive(ctx, item); err != nil {
		return dto.ApplyResult{}, err
	}

	var warnings []string
	if s.deps.Live != nil {
		if err := s.deps.Live.ApplyImage(ctx, item.Path); err != nil {
			warnings = append(warnings, err.Error())
		}
	}

	method := req.ReloadMethod
	if method == "" {
		method = kitty.ReloadAuto
	}
	if err := s.deps.Reloader.Reload(ctx, method); err != nil {
		warnings = append(warnings, err.Error())
	}
	result := dto.ApplyResult{WallpaperPath: item.Path}
	if len(warnings) > 0 {
		result.Warning = warnings[0]
		if len(warnings) > 1 {
			result.Warning = warnings[0] + "; " + warnings[1]
		}
	}
	return result, nil
}

func (s *WallpaperService) ensureIncludeReady(ctx context.Context) error {
	err := s.deps.Inspector.EnsureInclude(ctx)
	if err == nil {
		return nil
	}
	if !errors.Is(err, domainerr.ErrKittyIncludeMissing) {
		return err
	}
	if s.deps.Init.EnsureGenerated == nil || s.deps.Init.WriteInclude == nil {
		return err
	}
	if genErr := s.deps.Init.EnsureGenerated(); genErr != nil {
		return genErr
	}
	if writeErr := s.deps.Init.WriteInclude(); writeErr != nil {
		return writeErr
	}
	return s.deps.Inspector.EnsureInclude(ctx)
}

func (s *WallpaperService) PreviewBackground(ctx context.Context, req dto.PreviewRequest) error {
	if s.deps.Live == nil {
		return domainerr.ErrNotImplemented
	}
	path := req.Path
	if path == "" {
		if s.deps.Repo == nil {
			return domainerr.ErrNotImplemented
		}
		active, err := s.deps.Repo.GetActive(ctx)
		if err != nil || active.Path == "" {
			return domainerr.ErrInvalidWallpaper
		}
		path = active.Path
	}
	return s.deps.Live.PreviewBackground(ctx, path, req.Tint, req.Opacity)
}

func (s *WallpaperService) ListWallpapers(ctx context.Context) ([]wallpaper.Item, error) {
	if s.deps.Repo == nil {
		return nil, domainerr.ErrNotImplemented
	}
	return s.deps.Repo.List(ctx)
}

func (s *WallpaperService) SetRandom(ctx context.Context, method kitty.ReloadMethod) (dto.ApplyResult, error) {
	if s.deps.Repo == nil {
		return dto.ApplyResult{}, domainerr.ErrNotImplemented
	}
	items, err := s.deps.Repo.List(ctx)
	if err != nil {
		return dto.ApplyResult{}, err
	}
	if len(items) == 0 {
		return dto.ApplyResult{}, domainerr.ErrNoWallpapers
	}

	choice := items[s.deps.RandIntn(len(items))]
	if len(items) > 1 {
		active, _ := s.deps.Repo.GetActive(ctx)
		if active.Path != "" {
			filtered := make([]wallpaper.Item, 0, len(items))
			for _, item := range items {
				if item.Path != active.Path {
					filtered = append(filtered, item)
				}
			}
			if len(filtered) > 0 {
				choice = filtered[s.deps.RandIntn(len(filtered))]
			}
		}
	}

	return s.SetWallpaper(ctx, dto.SetWallpaperRequest{
		Path:         choice.Path,
		Selection:    wallpaper.SelectionRandom,
		ReloadMethod: method,
	})
}

func (s *WallpaperService) Status(ctx context.Context) (dto.StatusResult, error) {
	result := dto.StatusResult{
		KittyConfPath:     s.deps.Init.KittyConfPath,
		GeneratedConfPath: s.deps.Init.GeneratedConfPath,
	}
	if s.deps.Init.HasInclude != nil {
		ok, err := s.deps.Init.HasInclude()
		if err == nil {
			result.IncludeOK = ok
		}
	}
	if s.deps.Repo != nil {
		if items, err := s.deps.Repo.List(ctx); err == nil {
			result.WallpaperCount = len(items)
		}
		if active, err := s.deps.Repo.GetActive(ctx); err == nil {
			result.ActivePath = active.Path
		}
	}
	return result, nil
}

func (s *WallpaperService) Init(ctx context.Context, req dto.InitRequest) (dto.InitResult, error) {
	_ = ctx
	if s.deps.Init.EnsureGenerated == nil {
		return dto.InitResult{}, domainerr.ErrNotImplemented
	}
	if err := s.deps.Init.EnsureGenerated(); err != nil {
		return dto.InitResult{}, err
	}

	result := dto.InitResult{
		KittyConfPath:     s.deps.Init.KittyConfPath,
		GeneratedConfPath: s.deps.Init.GeneratedConfPath,
	}
	if !req.WriteInclude {
		return result, nil
	}
	if s.deps.Init.WriteInclude == nil {
		return dto.InitResult{}, domainerr.ErrNotImplemented
	}
	if err := s.deps.Init.WriteInclude(); err != nil {
		return dto.InitResult{}, err
	}
	result.IncludeWritten = true
	if restored, err := s.Restore(ctx, kitty.ReloadAuto); err == nil {
		result.RestoredPath = restored.WallpaperPath
	}
	return result, nil
}

// Restore re-applies the last saved wallpaper to the generated Kitty include file.
// Kitty loads that file on every new terminal window, so this also fixes persistence
// after reboot when include is present but the generated file was cleared.
func (s *WallpaperService) Restore(ctx context.Context, method kitty.ReloadMethod) (dto.ApplyResult, error) {
	if s.deps.Repo == nil || s.deps.Inspector == nil || s.deps.Writer == nil {
		return dto.ApplyResult{}, domainerr.ErrNotImplemented
	}

	active, err := s.deps.Repo.GetActive(ctx)
	if err != nil {
		return dto.ApplyResult{}, err
	}

	item, err := s.deps.Repo.GetByPath(ctx, active.Path)
	if err != nil {
		return dto.ApplyResult{}, err
	}

	if err := s.ensureIncludeReady(ctx); err != nil {
		return dto.ApplyResult{}, err
	}
	if err := s.deps.Writer.WriteBackgroundConfig(ctx, item.Path); err != nil {
		return dto.ApplyResult{}, err
	}

	var warnings []string
	if s.deps.Live != nil {
		if err := s.deps.Live.ApplyImage(ctx, item.Path); err != nil {
			warnings = append(warnings, err.Error())
		}
	}
	if method == "" {
		method = kitty.ReloadAuto
	}
	if s.deps.Reloader != nil {
		if err := s.deps.Reloader.Reload(ctx, method); err != nil {
			warnings = append(warnings, err.Error())
		}
	}

	result := dto.ApplyResult{WallpaperPath: item.Path}
	if len(warnings) > 0 {
		result.Warning = warnings[0]
		if len(warnings) > 1 {
			result.Warning = warnings[0] + "; " + warnings[1]
		}
	}
	return result, nil
}
