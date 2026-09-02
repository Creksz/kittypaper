package gui

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"kittypaper/internal/domain/online"
)

type onlinePanel struct {
	browser *browser
	root    fyne.CanvasObject

	searchEntry *widget.Entry
	grid        *widget.GridWrap
	status      *widget.Label
	loadMoreBtn *widget.Button
	progress    *widget.ProgressBar
	progressLbl *widget.Label
	cancelBtn   *widget.Button
	progressBox *fyne.Container
	bodyStack   *fyne.Container
	browseView  fyne.CanvasObject
}

func newOnlinePanel(b *browser) *onlinePanel {
	o := &onlinePanel{browser: b}
	o.build()
	return o
}

func (o *onlinePanel) build() {
	title := widget.NewLabelWithStyle("ONLINE", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	o.searchEntry = widget.NewEntry()
	o.searchEntry.SetPlaceHolder("Search wallpapers...")
	o.searchEntry.OnSubmitted = func(q string) {
		o.browser.onlineQuery = q
		o.search(q, 1)
	}

	searchBtn := widget.NewButton("Search", func() {
		o.browser.onlineQuery = o.searchEntry.Text
		o.search(o.searchEntry.Text, 1)
	})

	o.status = widget.NewLabel("Search Wallhaven for wallpapers")
	o.status.Importance = widget.LowImportance

	o.grid = widget.NewGridWrap(
		func() int { return len(o.browser.onlineItems) },
		func() fyne.CanvasObject {
			img := canvas.NewImageFromResource(nil)
			img.FillMode = canvas.ImageFillContain
			img.ScaleMode = canvas.ImageScaleFastest
			img.SetMinSize(fyne.NewSize(120, 80))
			return container.NewPadded(img)
		},
		func(i widget.GridWrapItemID, co fyne.CanvasObject) {
			box := co.(*fyne.Container)
			img := box.Objects[0].(*canvas.Image)
			if i < 0 || i >= len(o.browser.onlineItems) {
				img.Resource = nil
				return
			}
			item := o.browser.onlineItems[i]
			img.Resource = nil
			img.File = ""
			if item.ThumbURL == "" {
				return
			}
			url := item.ThumbURL
			o.browser.remoteThumbs.load(url, func(res fyne.Resource) {
				if i < 0 || i >= len(o.browser.onlineItems) || o.browser.onlineItems[i].ThumbURL != url {
					return
				}
				img.Resource = res
				img.Refresh()
			})
		},
	)
	o.grid.OnSelected = func(id widget.GridWrapItemID) {
		o.browser.onlineSelected = int(id)
		o.showDetail()
	}

	o.loadMoreBtn = widget.NewButton("Load More", func() {
		if o.browser.onlinePage < o.browser.onlineLastPage {
			o.search(o.browser.onlineQuery, o.browser.onlinePage+1)
		}
	})
	o.loadMoreBtn.Hide()

	o.progress = widget.NewProgressBar()
	o.progressLbl = widget.NewLabel("")
	o.cancelBtn = widget.NewButton("Cancel", func() {
		if o.browser.downloadCancel != nil {
			o.browser.downloadCancel()
		}
	})
	o.progressBox = container.NewVBox(o.progressLbl, o.progress, o.cancelBtn)
	o.progressBox.Hide()

	top := container.NewVBox(
		title,
		container.NewBorder(nil, nil, nil, searchBtn, o.searchEntry),
		o.status,
		o.progressBox,
	)

	o.browseView = container.NewBorder(nil, container.NewVBox(o.loadMoreBtn), nil, nil, o.grid)
	o.bodyStack = container.NewStack(o.browseView)

	o.root = container.NewBorder(
		top,
		nil,
		nil, nil,
		o.bodyStack,
	)
}

func (o *onlinePanel) show() {
	if o.browser.onlineQuery != "" {
		o.searchEntry.SetText(o.browser.onlineQuery)
	}
	o.showBrowse()
}

func (o *onlinePanel) showBrowse() {
	o.bodyStack.Objects = []fyne.CanvasObject{o.browseView}
	o.bodyStack.Refresh()
}

func (o *onlinePanel) search(query string, page int) {
	o.status.SetText("Searching...")
	go func() {
		result, err := o.browser.container.SearchOnline(context.Background(), query, page)
		fyne.Do(func() {
			if err != nil {
				o.status.SetText("Search failed: " + err.Error())
				o.browser.notifyError("Search failed")
				return
			}
			if page <= 1 {
				o.browser.onlineItems = result.Items
			} else {
				o.browser.onlineItems = append(o.browser.onlineItems, result.Items...)
			}
			o.browser.onlinePage = result.Page
			o.browser.onlineLastPage = result.LastPage
			o.grid.Refresh()
			if len(result.Items) == 0 && page <= 1 {
				o.status.SetText("No results — try another search")
			} else {
				o.status.SetText(fmt.Sprintf("%d results (page %d/%d)", result.Total, result.Page, result.LastPage))
			}
			if result.Page < result.LastPage {
				o.loadMoreBtn.Show()
			} else {
				o.loadMoreBtn.Hide()
			}
			o.showBrowse()
		})
	}()
}

func (o *onlinePanel) showDetail() {
	if o.browser.onlineSelected < 0 || o.browser.onlineSelected >= len(o.browser.onlineItems) {
		return
	}
	item := o.browser.onlineItems[o.browser.onlineSelected]
	o.bodyStack.Objects = []fyne.CanvasObject{o.detailPanel(item)}
	o.bodyStack.Refresh()
}

func (o *onlinePanel) detailPanel(item online.Item) fyne.CanvasObject {
	preview := canvas.NewImageFromResource(nil)
	preview.FillMode = canvas.ImageFillContain
	preview.ScaleMode = canvas.ImageScaleFastest
	preview.SetMinSize(fyne.NewSize(480, 360))
	if item.ThumbURL != "" {
		url := item.ThumbURL
		o.browser.remoteThumbs.load(url, func(res fyne.Resource) {
			preview.Resource = res
			preview.Refresh()
		})
	}

	info := widget.NewLabel(fmt.Sprintf("%s\nSource: %s", item.Resolution, item.Source))
	downloadBtn := widget.NewButton("Download", func() { o.browser.downloadOnline(item, false) })
	applyBtn := widget.NewButton("Apply", func() { o.browser.downloadOnline(item, true) })
	applyBtn.Importance = widget.HighImportance
	backBtn := widget.NewButton("Back", func() { o.showBrowse() })

	previewBox := container.NewMax(
		widget.NewLabel("Loading preview..."),
		container.NewPadded(preview),
	)

	return container.NewBorder(
		widget.NewLabelWithStyle("PREVIEW", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewHBox(backBtn, downloadBtn, applyBtn),
		nil, nil,
		container.NewVBox(container.NewPadded(previewBox), info),
	)
}

func (o *onlinePanel) showProgress(label string, ratio float64) {
	o.progressBox.Show()
	o.progressLbl.SetText(label)
	o.progress.SetValue(ratio)
}

func (o *onlinePanel) hideProgress() {
	o.progressBox.Hide()
}
