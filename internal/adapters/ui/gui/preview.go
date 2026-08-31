package gui

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "golang.org/x/image/webp"
)

type previewInfo struct {
	Width    int
	Height   int
	Modified time.Time
}

func loadPreviewInfo(path string) previewInfo {
	info := previewInfo{}
	if stat, err := os.Stat(path); err == nil {
		info.Modified = stat.ModTime()
	}
	file, err := os.Open(path)
	if err != nil {
		return info
	}
	defer file.Close()
	cfg, _, err := image.DecodeConfig(file)
	if err != nil {
		return info
	}
	info.Width = cfg.Width
	info.Height = cfg.Height
	return info
}

func formatResolution(info previewInfo) string {
	if info.Width <= 0 || info.Height <= 0 {
		return "unknown"
	}
	return fmt.Sprintf("%dx%d", info.Width, info.Height)
}

func formatModified(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	return t.Format("2006-01-02")
}

func matchesFilter(itemPath, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}
	base := strings.ToLower(filepath.Base(itemPath))
	full := strings.ToLower(itemPath)
	return strings.Contains(base, query) || strings.Contains(full, query)
}
