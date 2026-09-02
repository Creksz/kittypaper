package gui

import (
	"time"

	"fyne.io/fyne/v2"
)

func (b *browser) scheduleFilter() {
	b.filterMu.Lock()
	defer b.filterMu.Unlock()
	if b.filterTimer != nil {
		b.filterTimer.Stop()
	}
	b.filterTimer = time.AfterFunc(250*time.Millisecond, func() {
		fyne.Do(b.applyFilter)
	})
}
