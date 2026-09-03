package gui

import (
	"image/color"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"kittypaper/internal/version"
)

type searchTheme struct {
	fyne.Theme
}

func (t searchTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 16
	case theme.SizeNameInnerPadding:
		return 12
	case theme.SizeNamePadding:
		return 8
	default:
		return t.Theme.Size(name)
	}
}

func (b *browser) buildHeader() fyne.CanvasObject {
	logo := canvas.NewImageFromResource(logoResource())
	logo.FillMode = canvas.ImageFillContain
	logo.SetMinSize(fyne.NewSize(220, 52))

	badge := canvas.NewRectangle(color.NRGBA{R: 245, G: 245, B: 247, A: 255})
	badge.CornerRadius = 8
	badge.SetMinSize(fyne.NewSize(236, 56))
	brand := container.NewPadded(container.NewMax(badge, container.NewPadded(logo)))

	b.globalSearch = widget.NewEntry()
	b.globalSearch.SetPlaceHolder("Search wallpapers...")
	b.globalSearch.OnChanged = func(q string) {
		if b.activeSection == sectionOnline {
			return
		}
		if b.filterEntry != nil && b.filterEntry.Text != q {
			b.filterEntry.SetText(q)
		}
	}
	b.globalSearch.OnSubmitted = func(q string) {
		if b.activeSection == sectionOnline {
			b.onlineQuery = q
			b.onlinePanel.search(q, 1)
			return
		}
		if b.filterEntry != nil && b.filterEntry.Text != q {
			b.filterEntry.SetText(q)
		}
		b.applyFilter()
	}

	searchSlot := canvas.NewRectangle(color.Transparent)
	searchSlot.SetMinSize(fyne.NewSize(360, 40))
	searchBox := container.NewThemeOverride(
		container.NewMax(searchSlot, b.globalSearch),
		searchTheme{Theme: theme.DarkTheme()},
	)

	settingsBtn := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
		b.showSettingsDialog()
	})

	right := container.NewBorder(nil, nil, nil, settingsBtn, searchBox)
	return container.NewPadded(container.NewBorder(nil, nil, brand, right))
}

func (b *browser) buildToolbar() fyne.CanvasObject {
	return container.NewHBox(
		widget.NewButton("Random", b.onRandom),
		widget.NewButton("Refresh", b.onRefresh),
		widget.NewButton("Download", b.onDownloadCurrent),
		widget.NewButton("Collections", func() { b.switchSection(sectionCollections) }),
		widget.NewButton("Paths", b.onPaths),
	)
}

func (b *browser) buildSidebar() fyne.CanvasObject {
	libraryTitle := widget.NewLabelWithStyle("LIBRARY", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	sections := container.NewVBox()
	for _, sec := range []librarySection{sectionAll, sectionFavorites, sectionRecent, sectionOnline, sectionCollections} {
		s := sec
		btn := widget.NewButton(sec.label(), func() { b.switchSection(s) })
		b.sectionButtons[s] = btn
		sections.Add(btn)
	}

	b.filterEntry = widget.NewEntry()
	b.filterEntry.SetPlaceHolder("Filter...")
	b.filterEntry.OnChanged = func(string) { b.scheduleFilter() }

	b.list = widget.NewList(
		func() int { return len(b.filtered) },
		func() fyne.CanvasObject {
			img := canvas.NewImageFromResource(nil)
			img.FillMode = canvas.ImageFillContain
			img.ScaleMode = canvas.ImageScaleFastest
			img.SetMinSize(fyne.NewSize(thumbListW, thumbListH))
			label := widget.NewLabel("")
			label.Truncation = fyne.TextTruncateEllipsis
			return container.NewHBox(img, label)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			row := o.(*fyne.Container)
			img := row.Objects[0].(*canvas.Image)
			label := row.Objects[1].(*widget.Label)
			if i < 0 || i >= len(b.filtered) {
				img.Resource = nil
				img.File = ""
				label.SetText("")
				return
			}
			item := b.filtered[i]
			label.SetText(filepath.Base(item.Path))
			if int(i) == b.selected {
				label.TextStyle = fyne.TextStyle{Bold: true}
			} else {
				label.TextStyle = fyne.TextStyle{}
			}
			img.Resource = nil
			img.File = ""
			path := item.Path
			b.thumbs.listThumb(path, func(res fyne.Resource) {
				if i < 0 || i >= len(b.filtered) || b.filtered[i].Path != path {
					return
				}
				img.Resource = res
				img.Refresh()
			})
		},
	)
	b.list.OnSelected = func(id widget.ListItemID) {
		b.selected = int(id)
		b.refreshSelection()
	}
	b.list.OnUnselected = func(id widget.ListItemID) {
		_ = id
	}

	top := container.NewVBox(
		libraryTitle,
		sections,
		widget.NewSeparator(),
		b.filterEntry,
		widget.NewLabelWithStyle("v"+version.String(), fyne.TextAlignLeading, fyne.TextStyle{}),
	)

	return container.NewBorder(top, nil, nil, nil, b.list)
}

func (b *browser) buildStatusBar() fyne.CanvasObject {
	b.statusLeft = widget.NewLabel("")
	b.statusLeft.Importance = widget.LowImportance
	b.statusRight = widget.NewLabel("Ready")
	b.statusRight.Importance = widget.LowImportance
	return container.NewBorder(nil, nil, b.statusLeft, b.statusRight)
}

func (b *browser) showSettingsDialog() {
	tint := widget.NewSlider(0, 1)
	tint.Step = 0.01
	tint.SetValue(b.container.Settings.BackgroundTint)
	opacity := widget.NewSlider(0, 1)
	opacity.Step = 0.01
	opacity.SetValue(b.container.Settings.BackgroundOpacity)

	apiKey := widget.NewEntry()
	apiKey.SetText(b.container.Settings.WallhavenAPIKey)
	apiKey.SetPlaceHolder("Optional Wallhaven API key")

	form := container.NewVBox(
		widget.NewLabel("Appearance defaults (applied on Apply)"),
		widget.NewLabel("background_tint"),
		tint,
		widget.NewLabel("background_opacity"),
		opacity,
		widget.NewSeparator(),
		widget.NewLabel("Wallhaven API key (optional, improves rate limits)"),
		apiKey,
	)

	var popup *widget.PopUp
	popup = widget.NewModalPopUp(container.NewVBox(
		widget.NewLabelWithStyle("Settings", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		form,
		container.NewHBox(
			widget.NewButton("Save", func() {
				b.container.Settings.WallhavenAPIKey = apiKey.Text
				if b.container.OnlineService != nil {
					b.container.OnlineService.Wallhaven.APIKey = apiKey.Text
				}
				_ = b.container.SetAppearance(tint.Value, opacity.Value)
				_ = b.container.SaveConfig()
				b.previewPanel.syncSliders()
				popup.Hide()
				b.notifySuccess("Settings saved")
			}),
			widget.NewButton("Cancel", func() { popup.Hide() }),
		),
	), b.window.Canvas())
	popup.Resize(fyne.NewSize(420, 320))
	popup.Show()
}
