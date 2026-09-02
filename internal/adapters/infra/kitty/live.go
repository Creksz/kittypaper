package kitty

import (
	"context"
	"fmt"
	"path/filepath"
)

const defaultPreviewLayout = "scaled"

// PreviewBackground updates running Kitty windows without writing saved config.
func (r Reloader) PreviewBackground(ctx context.Context, wallpaperPath string, tint, opacity float64) error {
	abs := wallpaperPath
	if cleaned, err := filepath.Abs(wallpaperPath); err == nil {
		abs = cleaned
	}

	bin := r.KittyBinary
	if bin == "" {
		bin = "kitty"
	}

	if err := r.runKittyAt(ctx, bin, "@", "set-background-image", abs, "--all", "--layout", defaultPreviewLayout); err != nil {
		return fmt.Errorf("live preview image: %w", err)
	}
	if err := r.runKittyAt(ctx, bin, "@", "load-config",
		"-o", fmt.Sprintf("background_tint=%.2f", tint),
		"-o", fmt.Sprintf("background_opacity=%.2f", opacity),
	); err != nil {
		return fmt.Errorf("live preview appearance: %w", err)
	}
	return nil
}
