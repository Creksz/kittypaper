package dto

import (
	"kittypaper/internal/domain/kitty"
	"kittypaper/internal/domain/wallpaper"
)

type SetWallpaperRequest struct {
	Path         string
	Selection    wallpaper.SelectionMode
	ReloadMethod kitty.ReloadMethod
}

type ApplyResult struct {
	WallpaperPath string
	Warning       string
}

type PreviewRequest struct {
	Path    string
	Tint    float64
	Opacity float64
}

type StatusResult struct {
	ActivePath        string
	GeneratedConfPath string
	KittyConfPath     string
	IncludeOK         bool
	WallpaperCount    int
}

type InitRequest struct {
	WriteInclude bool
}

type InitResult struct {
	KittyConfPath     string
	GeneratedConfPath string
	IncludeWritten    bool
	RestoredPath      string
}
