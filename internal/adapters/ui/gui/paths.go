package gui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"kittypaper/internal/platform/pathx"
)

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
		b.notifySuccess(fmt.Sprintf("Saved %d wallpaper folder(s)", len(dirs)))
	}, b.window)
	d.Resize(fyne.NewSize(640, 420))
	d.Show()
}

func (b *browser) showProperties(path string) {
	info := loadPreviewInfo(path)
	rows := []string{
		"Resolution: " + formatResolution(info),
		"Aspect Ratio: " + formatAspectRatio(info),
		"File Size: " + formatFileSize(info.FileSize),
		"Modified: " + formatModified(info.Modified),
		"Path: " + path,
	}
	dialog.ShowInformation("Properties", strings.Join(rows, "\n"), b.window)
}

func (b *browser) openFolderAction(path string) {
	dir := filepath.Dir(path)
	_ = exec.Command("xdg-open", dir).Start()
	b.notifySuccess("Opened folder")
}
