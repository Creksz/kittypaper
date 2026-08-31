package errors

import "errors"

var (
	ErrNotImplemented      = errors.New("not implemented")
	ErrInvalidWallpaper    = errors.New("invalid wallpaper")
	ErrWallpaperNotFound   = errors.New("wallpaper not found")
	ErrNoWallpapers        = errors.New("no wallpapers found")
	ErrKittyIncludeMissing = errors.New("kitty include is missing")
	ErrKittyConfMissing    = errors.New("kitty.conf is missing")
	ErrUnknownReloadMethod = errors.New("unknown reload method")
)
