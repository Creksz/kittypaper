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
	FileSize int64
}

func loadPreviewInfo(path string) previewInfo {
	info := previewInfo{}
	if stat, err := os.Stat(path); err == nil {
		info.Modified = stat.ModTime()
		info.FileSize = stat.Size()
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
		return ""
	}
	return fmt.Sprintf("%d × %d", info.Width, info.Height)
}

func formatAspectRatio(info previewInfo) string {
	if info.Width <= 0 || info.Height <= 0 {
		return ""
	}
	g := gcd(info.Width, info.Height)
	if g == 0 {
		return ""
	}
	return fmt.Sprintf("%d:%d", info.Width/g, info.Height/g)
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func formatFileSize(size int64) string {
	if size <= 0 {
		return ""
	}
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func formatModified(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	now := time.Now()
	diff := now.Sub(t)
	switch {
	case diff < time.Minute:
		return "Just now"
	case diff < time.Hour:
		return fmt.Sprintf("%d min ago", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%d hours ago", int(diff.Hours()))
	case diff < 48*time.Hour:
		return "Yesterday"
	default:
		return t.Format("2006-01-02")
	}
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
