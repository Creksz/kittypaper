package gui

import (
	"context"
	"fmt"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"kittypaper/internal/app/dto"
	"kittypaper/internal/domain/online"
	"kittypaper/internal/domain/wallpaper"
)

func (b *browser) applyFilter() {
	query := b.filterEntry.Text
	if b.globalSearch != nil && b.globalSearch.Text != "" && query == "" {
		query = b.globalSearch.Text
	}
	prevPath := ""
	if item, ok := b.currentItem(); ok {
		prevPath = item.Path
	}
	filtered := make([]wallpaper.Item, 0, len(b.allItems))
	for _, item := range b.allItems {
		if matchesFilter(item.Path, query) {
			filtered = append(filtered, item)
		}
	}
	b.filtered = filtered
	b.selected = 0
	if prevPath != "" {
		for i, item := range b.filtered {
			if item.Path == prevPath {
				b.selected = i
				break
			}
		}
	}
	b.list.UnselectAll()
	if len(b.filtered) > 0 {
		b.list.Select(b.selected)
	}
	b.list.Refresh()
	b.refreshSelection()
}

func (b *browser) currentItem() (wallpaper.Item, bool) {
	if b.selected < 0 || b.selected >= len(b.filtered) {
		return wallpaper.Item{}, false
	}
	return b.filtered[b.selected], true
}

func (b *browser) refreshSelection() {
	if b.activeSection == sectionOnline {
		return
	}
	item, ok := b.currentItem()
	if !ok {
		b.previewPanel.showLocal("")
		return
	}
	b.previewPanel.showLocal(item.Path)
}

func (b *browser) persistAppearance() {
	tint := b.previewPanel.tintSlider.Value
	opacity := b.previewPanel.opacitySlider.Value
	go func() {
		if err := b.container.SetAppearance(tint, opacity); err != nil {
			fyne.Do(func() { b.showError("Appearance", err) })
			return
		}
		active, err := b.container.WallpaperService.Status(context.Background())
		if err != nil || active.ActivePath == "" {
			fyne.Do(func() {
				b.notifySuccess(fmt.Sprintf("Appearance saved tint=%.2f opacity=%.2f", tint, opacity))
			})
			return
		}
		_, applyErr := b.container.WallpaperService.SetWallpaper(context.Background(), dto.SetWallpaperRequest{
			Path:         active.ActivePath,
			ReloadMethod: b.container.Settings.ReloadMethod,
		})
		fyne.Do(func() {
			if applyErr != nil {
				b.notifyError("Appearance saved, re-apply failed: " + applyErr.Error())
				return
			}
			b.notifySuccess(fmt.Sprintf("Appearance applied tint=%.2f opacity=%.2f", tint, opacity))
		})
	}()
}

func (b *browser) onApply() {
	item, ok := b.currentItem()
	if !ok {
		dialog.ShowInformation("Apply", "Select a wallpaper first.", b.window)
		return
	}
	tint := b.previewPanel.tintSlider.Value
	opacity := b.previewPanel.opacitySlider.Value
	if err := b.container.SetAppearance(tint, opacity); err != nil {
		b.showError("Apply", err)
		return
	}
	b.setStatus("Applying " + filepath.Base(item.Path) + "...")
	go b.applyWallpaper(item.Path)
}

func (b *browser) applyWallpaper(path string) {
	result, err := b.container.WallpaperService.SetWallpaper(context.Background(), dto.SetWallpaperRequest{
		Path:         path,
		ReloadMethod: b.container.Settings.ReloadMethod,
	})
	fyne.Do(func() {
		if err != nil {
			b.showError("Apply", err)
			return
		}
		if b.container.Library != nil {
			_ = b.container.Library.AddRecent(path)
		}
		msg := "Wallpaper Applied"
		if result.Warning != "" {
			msg += " (warning: " + result.Warning + ")"
		}
		b.notifySuccess(msg)
		b.updateStatusBar()
	})
}

func (b *browser) onRandom() {
	b.setStatus("Picking random wallpaper...")
	go func() {
		result, err := b.container.WallpaperService.SetRandom(context.Background(), b.container.Settings.ReloadMethod)
		fyne.Do(func() {
			if err != nil {
				b.showError("Random", err)
				return
			}
			if b.container.Library != nil {
				_ = b.container.Library.AddRecent(result.WallpaperPath)
			}
			b.notifySuccess("Applied " + filepath.Base(result.WallpaperPath))
			b.selectByPath(result.WallpaperPath)
			b.updateStatusBar()
		})
	}()
}

func (b *browser) selectByPath(path string) {
	for i, item := range b.filtered {
		if item.Path == path {
			b.selected = i
			b.list.Select(i)
			b.refreshSelection()
			return
		}
	}
}

func (b *browser) onRefresh() {
	b.reloadSectionItems()
	b.notifySuccess(fmt.Sprintf("Refreshed — %d wallpapers", len(b.allItems)))
}

func (b *browser) onToggleFavorite() {
	item, ok := b.currentItem()
	if !ok {
		return
	}
	if b.container.Library == nil {
		return
	}
	nowFav, err := b.container.Library.ToggleFavorite(item.Path)
	if err != nil {
		b.showError("Favorite", err)
		return
	}
	if nowFav {
		b.notifySuccess("Added to favorites")
	} else {
		b.notifySuccess("Removed from favorites")
	}
	b.previewPanel.updateFavoriteButton(item.Path)
	b.updateStatusBar()
	if b.activeSection == sectionFavorites {
		b.reloadSectionItems()
	}
}

func (b *browser) onDownloadCurrent() {
	if b.activeSection == sectionOnline {
		if b.onlineSelected >= 0 && b.onlineSelected < len(b.onlineItems) {
			b.downloadOnline(b.onlineItems[b.onlineSelected], false)
		}
		return
	}
	dialog.ShowInformation("Download", "Download is for online wallpapers. Switch to Online and search.", b.window)
}

func (b *browser) downloadOnline(item online.Item, applyAfter bool) {
	dest := b.container.DownloadDir()
	if dest == "" {
		dialog.ShowInformation("Download", "Add a wallpaper folder in Paths first.", b.window)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	b.downloadCancel = cancel

	b.onlinePanel.showProgress("Downloading...", 0)

	go func() {
		path, err := b.container.OnlineService.Download(ctx, item, dest, func(done, total int64) {
			ratio := 0.0
			if total > 0 {
				ratio = float64(done) / float64(total)
			}
			fyne.Do(func() {
				b.onlinePanel.showProgress(fmt.Sprintf("Downloading... %d%%", int(ratio*100)), ratio)
			})
		})
		fyne.Do(func() {
			b.onlinePanel.hideProgress()
			b.downloadCancel = nil
			if err != nil {
				b.showError("Download", err)
				return
			}
			b.notifySuccess("Download Complete")
			b.onRefresh()
			if applyAfter {
				go b.applyWallpaper(path)
			}
		})
	}()
}

func (b *browser) copyPath(path string) {
	b.window.Clipboard().SetContent(path)
	b.notifySuccess("Path copied")
}

func (b *browser) onAddToCollection(path string) {
	if b.container.Library == nil {
		return
	}
	entry := widget.NewEntry()
	entry.SetPlaceHolder("Collection name")
	dialog.ShowForm("Add to Collection", "Save", "Cancel", []*widget.FormItem{
		widget.NewFormItem("Name", entry),
	}, func(ok bool) {
		if !ok || entry.Text == "" {
			return
		}
		if err := b.container.Library.AddToCollection(entry.Text, path); err != nil {
			b.showError("Collection", err)
			return
		}
		b.notifySuccess("Added to collection " + entry.Text)
	}, b.window)
}

func (b *browser) deleteWallpaper(path string) {
	dialog.ShowConfirm("Delete", "Delete "+filepath.Base(path)+"?", func(ok bool) {
		if !ok {
			return
		}
		if err := osRemove(path); err != nil {
			b.showError("Delete", err)
			return
		}
		b.notifySuccess("Deleted " + filepath.Base(path))
		b.onRefresh()
	}, b.window)
}

func (b *browser) renameWallpaper(path string) {
	entry := widget.NewEntry()
	entry.SetText(filepath.Base(path))
	dialog.ShowForm("Rename", "Save", "Cancel", []*widget.FormItem{
		widget.NewFormItem("Name", entry),
	}, func(ok bool) {
		if !ok || entry.Text == "" {
			return
		}
		newPath := filepath.Join(filepath.Dir(path), entry.Text)
		if err := osRename(path, newPath); err != nil {
			b.showError("Rename", err)
			return
		}
		b.notifySuccess("Renamed")
		b.onRefresh()
	}, b.window)
}
