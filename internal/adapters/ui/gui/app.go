package gui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"kittypaper/internal/app/dto"
	"kittypaper/internal/bootstrap"
	"kittypaper/internal/domain/wallpaper"
	"kittypaper/internal/platform/pathx"
	"kittypaper/internal/version"
)

type browser struct {
	container *bootstrap.Container

	app      fyne.App
	window   fyne.Window
	allItems []wallpaper.Item
	filtered []wallpaper.Item
	selected int

	list          *widget.List
	filter        *widget.Entry
	preview       *canvas.Image
	resolution    *widget.Label
	modified      *widget.Label
	status        *widget.Label
	tintSlider    *widget.Slider
	opacitySlider *widget.Slider
	tintValue     *widget.Label
	opacityValue  *widget.Label
}

func Run(c *bootstrap.Container) error {
	if c == nil || c.WallpaperService == nil || c.Settings == nil {
		return fmt.Errorf("gui container is not configured")
	}

	items, err := c.WallpaperService.ListWallpapers(context.Background())
	if err != nil {
		return err
	}

	b := &browser{
		container: c,
		allItems:  items,
		filtered:  append([]wallpaper.Item(nil), items...),
		selected:  0,
	}
	b.build()
	b.window.ShowAndRun()
	return nil
}

func (b *browser) build() {
	b.app = app.NewWithID("dev.kittypaper.app")
	b.app.Settings().SetTheme(theme.DarkTheme())

	b.window = b.app.NewWindow("Kittypaper")
	b.window.Resize(fyne.NewSize(980, 680))
	b.window.SetContent(b.layout())
	if len(b.filtered) > 0 {
		b.list.Select(0)
	}
	b.refreshSelection()
	if len(b.allItems) == 0 {
		b.setStatus("No wallpapers found — open Paths to add a folder")
	}
}

func (b *browser) layout() fyne.CanvasObject {
	return container.NewBorder(
		container.NewVBox(b.header(), b.toolbar()),
		b.statusBar(),
		nil, nil,
		b.body(),
	)
}

func (b *browser) header() fyne.CanvasObject {
	title := canvas.NewText("KITTYPAPER", theme.ForegroundColor())
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 22

	subtitle := widget.NewLabel("Wallpaper browser for Kitty terminal")
	subtitle.Importance = widget.LowImportance

	versionLabel := widget.NewLabel(version.String())
	versionLabel.Importance = widget.LowImportance

	return container.NewVBox(
		container.NewHBox(title, widget.NewLabel(" "), versionLabel),
		subtitle,
	)
}

func (b *browser) toolbar() fyne.CanvasObject {
	return container.NewHBox(
		widget.NewButton("Random", b.onRandom),
		widget.NewButton("Refresh", b.onRefresh),
		widget.NewButton("Paths", b.onPaths),
		widget.NewButton("Help", b.onHelp),
	)
}

func (b *browser) body() fyne.CanvasObject {
	libraryTitle := widget.NewLabelWithStyle("LIBRARY", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	b.filter = widget.NewEntry()
	b.filter.SetPlaceHolder("Filter by name or path")
	b.filter.OnChanged = func(string) { b.applyFilter() }

	b.list = widget.NewList(
		func() int { return len(b.filtered) },
		func() fyne.CanvasObject { return widget.NewLabel("template") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			label := o.(*widget.Label)
			if i < 0 || i >= len(b.filtered) {
				label.SetText("")
				return
			}
			label.SetText(filepath.Base(b.filtered[i].Path))
		},
	)
	b.list.OnSelected = func(id widget.ListItemID) {
		b.selected = int(id)
		b.refreshSelection()
	}

	left := container.NewBorder(
		container.NewVBox(libraryTitle, widget.NewLabel("ALL"), b.filter),
		nil, nil, nil,
		b.list,
	)

	previewTitle := widget.NewLabelWithStyle("PREVIEW", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	b.preview = canvas.NewImageFromFile("")
	b.preview.FillMode = canvas.ImageFillContain
	b.preview.SetMinSize(fyne.NewSize(360, 240))

	b.resolution = widget.NewLabel("RESOLUTION —")
	b.modified = widget.NewLabel("MODIFIED —")

	settingsTitle := widget.NewLabelWithStyle("APPEARANCE", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	b.tintValue = widget.NewLabel("")
	b.opacityValue = widget.NewLabel("")
	b.tintSlider = widget.NewSlider(0, 1)
	b.opacitySlider = widget.NewSlider(0, 1)
	b.tintSlider.Step = 0.01
	b.opacitySlider.Step = 0.01
	b.tintSlider.SetValue(b.container.Settings.BackgroundTint)
	b.opacitySlider.SetValue(b.container.Settings.BackgroundOpacity)
	b.updateAppearanceLabels()

	b.tintSlider.OnChanged = func(v float64) {
		b.tintValue.SetText(fmt.Sprintf("background_tint  %.2f", v))
	}
	b.opacitySlider.OnChanged = func(v float64) {
		b.opacityValue.SetText(fmt.Sprintf("background_opacity  %.2f", v))
	}
	b.tintSlider.OnChangeEnded = func(float64) { b.persistAppearance() }
	b.opacitySlider.OnChangeEnded = func(float64) { b.persistAppearance() }

	applyBtn := widget.NewButton("Apply", b.onApply)
	applyBtn.Importance = widget.HighImportance

	meta := container.NewVBox(
		b.resolution,
		b.modified,
		widget.NewSeparator(),
		settingsTitle,
		b.tintValue,
		b.tintSlider,
		b.opacityValue,
		b.opacitySlider,
		widget.NewSeparator(),
		applyBtn,
	)

	right := container.NewBorder(
		previewTitle,
		meta,
		nil, nil,
		container.NewPadded(b.preview),
	)

	split := container.NewHSplit(left, right)
	split.SetOffset(0.42)
	return split
}

func (b *browser) statusBar() fyne.CanvasObject {
	b.status = widget.NewLabel("Ready")
	b.status.Importance = widget.LowImportance
	return container.NewPadded(b.status)
}

func (b *browser) updateAppearanceLabels() {
	b.tintValue.SetText(fmt.Sprintf("background_tint  %.2f", b.tintSlider.Value))
	b.opacityValue.SetText(fmt.Sprintf("background_opacity  %.2f", b.opacitySlider.Value))
}

func (b *browser) persistAppearance() {
	tint := b.tintSlider.Value
	opacity := b.opacitySlider.Value
	if err := b.container.SetAppearance(tint, opacity); err != nil {
		dialog.ShowError(err, b.window)
		return
	}
	b.setStatus(fmt.Sprintf("Saved appearance tint=%.2f opacity=%.2f", tint, opacity))

	go func() {
		active, err := b.container.WallpaperService.Status(context.Background())
		if err != nil || active.ActivePath == "" {
			return
		}
		_, applyErr := b.container.WallpaperService.SetWallpaper(context.Background(), dto.SetWallpaperRequest{
			Path:         active.ActivePath,
			ReloadMethod: b.container.Settings.ReloadMethod,
		})
		fyne.Do(func() {
			if applyErr != nil {
				b.setStatus("Appearance saved, but re-apply failed: " + applyErr.Error())
				return
			}
			b.setStatus(fmt.Sprintf("Appearance applied tint=%.2f opacity=%.2f", tint, opacity))
		})
	}()
}

func (b *browser) applyFilter() {
	query := b.filter.Text
	filtered := make([]wallpaper.Item, 0, len(b.allItems))
	for _, item := range b.allItems {
		if matchesFilter(item.Path, query) {
			filtered = append(filtered, item)
		}
	}
	b.filtered = filtered
	if b.selected >= len(b.filtered) {
		b.selected = 0
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
	item, ok := b.currentItem()
	if !ok {
		b.preview.File = ""
		b.preview.Refresh()
		b.resolution.SetText("RESOLUTION —")
		b.modified.SetText("MODIFIED —")
		return
	}

	b.preview.File = item.Path
	b.preview.Refresh()

	info := loadPreviewInfo(item.Path)
	b.resolution.SetText("RESOLUTION  " + formatResolution(info))
	b.modified.SetText("MODIFIED  " + formatModified(info.Modified))
}

func (b *browser) setStatus(msg string) {
	b.status.SetText(msg)
}

func (b *browser) onApply() {
	item, ok := b.currentItem()
	if !ok {
		dialog.ShowInformation("Apply", "Select a wallpaper first, or add a folder in Paths.", b.window)
		return
	}
	b.setStatus("Applying " + filepath.Base(item.Path) + "...")
	go b.applyWallpaper(item.Path)
}

func (b *browser) onRandom() {
	b.setStatus("Picking random wallpaper...")
	go func() {
		result, err := b.container.WallpaperService.SetRandom(context.Background(), b.container.Settings.ReloadMethod)
		fyne.Do(func() {
			if err != nil {
				b.setStatus("Error: " + err.Error())
				dialog.ShowError(err, b.window)
				return
			}
			msg := "Applied " + filepath.Base(result.WallpaperPath)
			if result.Warning != "" {
				msg += " (warning: " + result.Warning + ")"
			}
			b.setStatus(msg)
			b.selectByPath(result.WallpaperPath)
		})
	}()
}

func (b *browser) applyWallpaper(path string) {
	result, err := b.container.WallpaperService.SetWallpaper(context.Background(), dto.SetWallpaperRequest{
		Path:         path,
		ReloadMethod: b.container.Settings.ReloadMethod,
	})
	fyne.Do(func() {
		if err != nil {
			b.setStatus("Error: " + err.Error())
			dialog.ShowError(err, b.window)
			return
		}
		msg := "Applied " + filepath.Base(result.WallpaperPath)
		if result.Warning != "" {
			msg += " (warning: " + result.Warning + ")"
		}
		b.setStatus(msg)
	})
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
	items, err := b.container.WallpaperService.ListWallpapers(context.Background())
	if err != nil {
		dialog.ShowError(err, b.window)
		return
	}
	b.allItems = items
	b.applyFilter()
	b.setStatus(fmt.Sprintf("Refreshed — %d wallpapers", len(items)))
}

func (b *browser) onPaths() {
	b.showPathsDialog()
}

func (b *browser) showPathsDialog() {
	dirs := append([]string(nil), b.container.Settings.WallpaperDirs...)

	pathEntry := widget.NewEntry()
	pathEntry.SetPlaceHolder("/home/you/Pictures/Wallpapers")

	listBox := container.NewVBox()
	var refreshList func()
	refreshList = func() {
		listBox.Objects = nil
		if len(dirs) == 0 {
			listBox.Add(widget.NewLabel("No wallpaper folders yet."))
		}
		for _, dir := range dirs {
			dirCopy := dir
			row := container.NewBorder(
				nil, nil, nil,
				widget.NewButton("Remove", func() {
					next := make([]string, 0, len(dirs))
					for _, d := range dirs {
						if d != dirCopy {
							next = append(next, d)
						}
					}
					dirs = next
					refreshList()
				}),
				widget.NewLabel(dirCopy),
			)
			listBox.Add(row)
		}
		listBox.Refresh()
	}
	refreshList()

	addTyped := func() {
		raw := strings.TrimSpace(pathEntry.Text)
		if raw == "" {
			return
		}
		abs, err := pathx.Abs(raw)
		if err != nil {
			dialog.ShowError(err, b.window)
			return
		}
		info, err := os.Stat(abs)
		if err != nil {
			dialog.ShowError(fmt.Errorf("path not found: %s", abs), b.window)
			return
		}
		if !info.IsDir() {
			dialog.ShowError(fmt.Errorf("not a directory: %s", abs), b.window)
			return
		}
		for _, d := range dirs {
			if d == abs {
				pathEntry.SetText("")
				return
			}
		}
		dirs = append(dirs, abs)
		pathEntry.SetText("")
		refreshList()
	}
	pathEntry.OnSubmitted = func(string) { addTyped() }

	addFolderBtn := widget.NewButton("Add Folder", func() {
		fd := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, b.window)
				return
			}
			if uri == nil {
				return
			}
			path := uri.Path()
			for _, d := range dirs {
				if d == path {
					return
				}
			}
			dirs = append(dirs, path)
			refreshList()
		}, b.window)
		fd.Resize(fyne.NewSize(700, 500))
		if home, err := os.UserHomeDir(); err == nil {
			if lister, err := storage.ListerForURI(storage.NewFileURI(home)); err == nil {
				fd.SetLocation(lister)
			}
		}
		fd.Show()
	})

	scroll := container.NewVScroll(listBox)
	scroll.SetMinSize(fyne.NewSize(560, 180))

	form := container.NewVBox(
		widget.NewLabel("Add a wallpaper folder with the system picker or by typing a path."),
		addFolderBtn,
		container.NewBorder(nil, nil, nil, widget.NewButton("Add Typed Path", addTyped), pathEntry),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Current folders", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		scroll,
	)

	d := dialog.NewCustomConfirm("Wallpaper Paths", "Save", "Cancel", form, func(ok bool) {
		if !ok {
			return
		}
		if err := b.container.SetWallpaperDirs(dirs); err != nil {
			dialog.ShowError(err, b.window)
			return
		}
		b.onRefresh()
		b.setStatus(fmt.Sprintf("Saved %d wallpaper folder(s)", len(dirs)))
	}, b.window)
	d.Resize(fyne.NewSize(640, 420))
	d.Show()
}

func (b *browser) onHelp() {
	help := `Kittypaper GUI

• Select a wallpaper from the library
• Adjust background_tint / background_opacity under PREVIEW
• Defaults: tint 0.95, opacity 1.0
• Paths: add folders with picker or typed path (like Walt)
• Apply sets the Kitty terminal background

Kittypaper writes kittypaper-background.conf and reloads Kitty automatically.`
	dialog.ShowInformation("Help", help, b.window)
}
