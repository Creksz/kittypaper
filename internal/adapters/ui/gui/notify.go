package gui

import (
	"fyne.io/fyne/v2/dialog"
)

func (b *browser) notifySuccess(msg string) {
	b.setStatus("✓ " + msg)
}

func (b *browser) notifyError(msg string) {
	b.setStatus("✕ " + msg)
}

func (b *browser) showError(title string, err error) {
	if err == nil {
		return
	}
	b.notifyError(err.Error())
	dialog.ShowError(err, b.window)
}
