package gui

import (
	"context"
	"fmt"

	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"kittypaper/internal/bootstrap"
	"kittypaper/internal/domain/online"
	"kittypaper/internal/domain/wallpaper"
)

type browser struct {
	container *bootstrap.Container

	app    fyne.App
	window fyne.Window

	allItems     []wallpaper.Item
	filtered     []wallpaper.Item
	selected     int
	activeSection librarySection

	onlineItems     []online.Item
	onlineSelected  int
	onlinePage      int
	onlineLastPage  int
	onlineQuery     string

	downloadCancel context.CancelFunc

	mainStack *fyne.Container

	list         *widget.List
	filterEntry  *widget.Entry
	globalSearch *widget.Entry

	previewPanel *previewPanel
	onlinePanel  *onlinePanel

	statusLeft  *widget.Label
	statusRight *widget.Label

	sectionButtons map[librarySection]*widget.Button

	thumbs       *thumbnailCache
	remoteThumbs *remoteThumbCache
	filterMu     sync.Mutex
	filterTimer  *time.Timer
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
		container:      c,
		allItems:       items,
		filtered:       append([]wallpaper.Item(nil), items...),
		selected:       0,
		activeSection:  sectionAll,
		sectionButtons: make(map[librarySection]*widget.Button),
	}
	b.build()
	b.window.ShowAndRun()
	return nil
}

func (b *browser) build() {
	b.app = app.NewWithID("dev.kittypaper.app")
	b.app.Settings().SetTheme(theme.DarkTheme())

	b.window = b.app.NewWindow("Kittypaper")
	b.window.Resize(fyne.NewSize(1100, 780))

	b.previewPanel = newPreviewPanel(b)
	b.onlinePanel = newOnlinePanel(b)
	b.thumbs = newThumbnailCache(b.container.Settings.CacheDir)
	b.remoteThumbs = newRemoteThumbCache()
	b.window.SetContent(b.layout())

	b.reloadSectionItems()
	b.updateSectionCounts()
	if len(b.filtered) > 0 {
		b.list.Select(0)
	}
	b.refreshSelection()
	b.updateStatusBar()
	if len(b.allItems) == 0 && b.activeSection == sectionAll {
		b.setStatus("No wallpapers found — open Paths to add a folder")
	}
}

func (b *browser) layout() fyne.CanvasObject {
	sidebar := b.buildSidebar()

	b.mainStack = container.NewStack(b.previewPanel.root)
	body := container.NewHSplit(sidebar, b.mainStack)
	body.SetOffset(0.32)

	return container.NewBorder(
		container.NewVBox(b.buildHeader(), b.buildToolbar()),
		b.buildStatusBar(),
		nil, nil,
		body,
	)
}

func (b *browser) setStatus(msg string) {
	if b.statusRight != nil {
		b.statusRight.SetText(msg)
	}
}

func (b *browser) updateStatusBar() {
	count := len(b.allItems)
	favCount := 0
	if b.container.Library != nil {
		if n, err := b.container.Library.FavoriteCount(); err == nil {
			favCount = n
		}
	}
	if b.statusLeft != nil {
		b.statusLeft.SetText(fmt.Sprintf("%d Wallpapers     %d Favorites", count, favCount))
	}
}

func (b *browser) switchSection(sec librarySection) {
	b.activeSection = sec
	for s, btn := range b.sectionButtons {
		if s == sec {
			btn.Importance = widget.HighImportance
		} else {
			btn.Importance = widget.MediumImportance
		}
	}

	if sec == sectionOnline {
		b.mainStack.Objects = []fyne.CanvasObject{b.onlinePanel.root}
		b.onlinePanel.show()
	} else {
		b.mainStack.Objects = []fyne.CanvasObject{b.previewPanel.root}
		b.onlinePanel.showBrowse()
		b.reloadSectionItems()
	}
	b.mainStack.Refresh()
	b.updateSectionCounts()
}

func (b *browser) reloadSectionItems() {
	ctx := context.Background()
	switch b.activeSection {
	case sectionFavorites:
		paths, err := b.container.Library.Favorites()
		if err != nil {
			b.showError("Favorites", err)
			b.allItems = nil
		} else {
			b.allItems = b.itemsFromPaths(paths)
		}
	case sectionRecent:
		paths, err := b.container.Library.Recent()
		if err != nil {
			b.showError("Recent", err)
			b.allItems = nil
		} else {
			b.allItems = b.itemsFromPaths(paths)
		}
	case sectionCollections:
		if b.container.Library != nil {
			names, _ := b.container.Library.CollectionNames()
			if len(names) == 0 {
				b.allItems = nil
				b.setStatus("No collections yet — use Add to Collection from a wallpaper")
			} else {
				paths, _ := b.container.Library.CollectionPaths(names[0])
				b.allItems = b.itemsFromPaths(paths)
			}
		}
	default:
		items, err := b.container.WallpaperService.ListWallpapers(ctx)
		if err != nil {
			b.showError("Library", err)
			b.allItems = nil
		} else {
			b.allItems = items
		}
	}
	b.applyFilter()
	b.updateStatusBar()
}

func (b *browser) itemsFromPaths(paths []string) []wallpaper.Item {
	ctx := context.Background()
	items := make([]wallpaper.Item, 0, len(paths))
	for _, p := range paths {
		item, err := b.container.Repo.GetByPath(ctx, p)
		if err == nil {
			items = append(items, item)
		}
	}
	return items
}

func (b *browser) updateSectionCounts() {
	allCount := len(b.filtered)
	if items, err := b.container.WallpaperService.ListWallpapers(context.Background()); err == nil {
		allCount = len(items)
	}
	favCount := 0
	recentCount := 0
	if b.container.Library != nil {
		if n, err := b.container.Library.FavoriteCount(); err == nil {
			favCount = n
		}
		if rec, err := b.container.Library.Recent(); err == nil {
			recentCount = len(rec)
		}
	}
	setBtn := func(sec librarySection, count int) {
		if btn, ok := b.sectionButtons[sec]; ok {
			btn.SetText(fmt.Sprintf("%s  %d", sec.label(), count))
		}
	}
	setBtn(sectionAll, allCount)
	setBtn(sectionFavorites, favCount)
	setBtn(sectionRecent, recentCount)
	if btn, ok := b.sectionButtons[sectionOnline]; ok {
		btn.SetText("Online")
	}
	if btn, ok := b.sectionButtons[sectionCollections]; ok {
		btn.SetText("Collections")
	}
}
