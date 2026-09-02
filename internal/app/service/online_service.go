package service

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"kittypaper/internal/adapters/infra/download"
	"kittypaper/internal/adapters/infra/wallhaven"
	"kittypaper/internal/domain/online"
)

type OnlineService struct {
	Wallhaven  wallhaven.Client
	Downloader download.Downloader
}

func (s OnlineService) Search(ctx context.Context, query string, page int) (online.SearchResult, error) {
	return s.Wallhaven.Search(ctx, query, page)
}

func (s OnlineService) Download(ctx context.Context, item online.Item, destDir string, onProgress download.ProgressFunc) (string, error) {
	if strings.TrimSpace(item.ImageURL) == "" {
		return "", fmt.Errorf("online wallpaper has no download URL")
	}
	if destDir == "" {
		return "", fmt.Errorf("no download directory configured")
	}
	filename := sanitizeFilename(item.ID, item.ImageURL)
	return s.Downloader.Download(ctx, item.ImageURL, destDir, filename, onProgress)
}

func sanitizeFilename(id, imageURL string) string {
	ext := filepath.Ext(strings.Split(imageURL, "?")[0])
	if ext == "" {
		ext = ".jpg"
	}
	if id == "" {
		id = "wallpaper"
	}
	return "wallhaven-" + id + ext
}
