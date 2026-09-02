package gui

import (
	"os"
)

func osRemove(path string) error {
	return os.Remove(path)
}

func osRename(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}
