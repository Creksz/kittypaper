package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

func (b *browser) showContextMenu(path string, pos fyne.Position) {
	menu := fyne.NewMenu("",
		fyne.NewMenuItem("Apply", func() {
			if i := b.indexOfPath(path); i >= 0 {
				b.selected = i
				b.list.Select(i)
			}
			b.onApply()
		}),
		fyne.NewMenuItem("Favorite", func() {
			if i := b.indexOfPath(path); i >= 0 {
				b.selected = i
				b.list.Select(i)
			}
			b.onToggleFavorite()
		}),
		fyne.NewMenuItem("Add to Collection", func() { b.onAddToCollection(path) }),
		fyne.NewMenuItem("Rename", func() { b.renameWallpaper(path) }),
		fyne.NewMenuItem("Delete", func() { b.deleteWallpaper(path) }),
		fyne.NewMenuItem("Open Folder", func() { b.openFolderAction(path) }),
		fyne.NewMenuItem("Copy Path", func() { b.copyPath(path) }),
		fyne.NewMenuItem("Properties", func() { b.showProperties(path) }),
	)
	if pos.X == 0 && pos.Y == 0 {
		pos = fyne.NewPos(200, 200)
	}
	widget.ShowPopUpMenuAtPosition(menu, b.window.Canvas(), pos)
}

func (b *browser) indexOfPath(path string) int {
	for i, item := range b.filtered {
		if item.Path == path {
			return i
		}
	}
	return -1
}
