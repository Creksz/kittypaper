package gui

import (
	"fmt"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type previewPanel struct {
	browser *browser
	root    fyne.CanvasObject

	preview       *canvas.Image
	placeholder   *widget.Label
	infoBox       *fyne.Container
	tintSlider    *widget.Slider
	opacitySlider *widget.Slider
	tintLabel     *widget.Label
	opacityLabel  *widget.Label
	applyBtn      *widget.Button
	downloadBtn   *widget.Button
	favoriteBtn   *widget.Button
}

func newPreviewPanel(b *browser) *previewPanel {
	p := &previewPanel{browser: b}
	p.build()
	return p
}

func (p *previewPanel) build() {
	title := widget.NewLabelWithStyle("PREVIEW", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	p.preview = canvas.NewImageFromResource(nil)
	p.preview.FillMode = canvas.ImageFillContain
	p.preview.ScaleMode = canvas.ImageScaleFastest
	p.placeholder = widget.NewLabel("Select a wallpaper to preview")
	p.placeholder.Alignment = fyne.TextAlignCenter
	p.placeholder.Importance = widget.LowImportance

	previewBox := container.NewMax(
		p.placeholder,
		container.NewPadded(p.preview),
	)

	p.infoBox = container.NewVBox()
	infoScroll := container.NewVScroll(p.infoBox)
	infoScroll.SetMinSize(fyne.NewSize(0, 100))

	appearanceTitle := widget.NewLabelWithStyle("APPEARANCE", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	p.tintSlider = widget.NewSlider(0, 1)
	p.tintSlider.Step = 0.01
	p.tintSlider.SetValue(p.browser.container.Settings.BackgroundTint)
	p.opacitySlider = widget.NewSlider(0, 1)
	p.opacitySlider.Step = 0.01
	p.opacitySlider.SetValue(p.browser.container.Settings.BackgroundOpacity)
	p.tintLabel = widget.NewLabel(fmt.Sprintf("background_tint  %.2f", p.tintSlider.Value))
	p.opacityLabel = widget.NewLabel(fmt.Sprintf("background_opacity  %.2f", p.opacitySlider.Value))

	p.tintSlider.OnChanged = func(v float64) {
		p.tintLabel.SetText(fmt.Sprintf("background_tint  %.2f", v))
	}
	p.opacitySlider.OnChanged = func(v float64) {
		p.opacityLabel.SetText(fmt.Sprintf("background_opacity  %.2f", v))
	}
	p.tintSlider.OnChangeEnded = func(float64) { p.browser.persistAppearance() }
	p.opacitySlider.OnChangeEnded = func(float64) { p.browser.persistAppearance() }

	p.applyBtn = widget.NewButton("Apply", p.browser.onApply)
	p.applyBtn.Importance = widget.HighImportance
	p.downloadBtn = widget.NewButton("Download", p.browser.onDownloadCurrent)
	p.downloadBtn.Hide()
	p.favoriteBtn = widget.NewButton("☆ Favorite", p.browser.onToggleFavorite)
	menuBtn := widget.NewButton("⋯", func() {
		if item, ok := p.browser.currentItem(); ok {
			p.browser.showContextMenu(item.Path, fyne.NewPos(0, 0))
		}
	})

	actions := container.NewHBox(p.downloadBtn, p.favoriteBtn, menuBtn, p.applyBtn)

	controls := container.NewVBox(
		widget.NewLabelWithStyle("INFORMATION", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		infoScroll,
		widget.NewSeparator(),
		appearanceTitle,
		p.tintLabel,
		p.tintSlider,
		p.opacityLabel,
		p.opacitySlider,
		widget.NewSeparator(),
		actions,
	)

	split := container.NewVSplit(
		container.NewPadded(previewBox),
		container.NewPadded(controls),
	)
	split.SetOffset(0.48)

	p.root = container.NewBorder(title, nil, nil, nil, split)
}

func (p *previewPanel) syncSliders() {
	p.tintSlider.SetValue(p.browser.container.Settings.BackgroundTint)
	p.opacitySlider.SetValue(p.browser.container.Settings.BackgroundOpacity)
	p.tintLabel.SetText(fmt.Sprintf("background_tint  %.2f", p.tintSlider.Value))
	p.opacityLabel.SetText(fmt.Sprintf("background_opacity  %.2f", p.opacitySlider.Value))
}

func (p *previewPanel) setInfo(rows [][2]string) {
	p.infoBox.Objects = nil
	for _, row := range rows {
		if row[1] == "" {
			continue
		}
		key := widget.NewLabel(row[0])
		key.Importance = widget.LowImportance
		val := widget.NewLabel(row[1])
		p.infoBox.Add(container.NewGridWithColumns(2, key, val))
	}
	p.infoBox.Refresh()
}

func (p *previewPanel) showLocal(path string) {
	if path == "" {
		p.preview.Resource = nil
		p.preview.File = ""
		p.preview.Hide()
		p.placeholder.Show()
		p.setInfo(nil)
		p.favoriteBtn.SetText("☆ Favorite")
		return
	}
	p.preview.Show()
	p.placeholder.Hide()
	p.preview.Resource = nil
	p.preview.File = ""

	go func() {
		info := loadPreviewInfo(path)
		fyne.Do(func() {
			p.setInfo([][2]string{
				{"Resolution", formatResolution(info)},
				{"Aspect Ratio", formatAspectRatio(info)},
				{"File Size", formatFileSize(info.FileSize)},
				{"Modified", formatModified(info.Modified)},
				{"Source", "Local"},
				{"Path", filepath.Base(path)},
			})
			p.downloadBtn.Hide()
			p.applyBtn.Show()
			p.updateFavoriteButton(path)
		})
	}()

	p.browser.thumbs.previewThumb(path, func(res fyne.Resource) {
		if item, ok := p.browser.currentItem(); !ok || item.Path != path {
			return
		}
		p.preview.Resource = res
		p.preview.Refresh()
	})
}

func (p *previewPanel) updateFavoriteButton(path string) {
	if p.browser.container.Library == nil {
		return
	}
	ok, err := p.browser.container.Library.IsFavorite(path)
	if err != nil {
		return
	}
	if ok {
		p.favoriteBtn.SetText("★ Favorited")
	} else {
		p.favoriteBtn.SetText("☆ Favorite")
	}
}

func (p *previewPanel) showOnline(itemPath string, resolution, source string) {
	p.preview.File = itemPath
	if itemPath != "" {
		p.preview.Show()
		p.placeholder.Hide()
	} else {
		p.preview.Hide()
		p.placeholder.Show()
	}
	p.preview.Refresh()
	p.setInfo([][2]string{
		{"Resolution", resolution},
		{"Source", source},
	})
	p.downloadBtn.Show()
	p.applyBtn.Show()
	p.favoriteBtn.Hide()
}
