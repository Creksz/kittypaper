package pathx

import (
	"os"
	"path/filepath"
	"strings"
)

func Expand(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}
	if path == "~" {
		return homeDir()
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir(), path[2:])
	}
	return path
}

func Abs(path string) (string, error) {
	expanded := Expand(path)
	if expanded == "" {
		return "", nil
	}
	return filepath.Abs(expanded)
}

func homeDir() string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return home
	}
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	return "."
}
