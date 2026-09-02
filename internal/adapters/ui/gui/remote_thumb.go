package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
)

func loadRemoteResource(url string) fyne.Resource {
	uri, err := storage.ParseURI(url)
	if err != nil {
		return nil
	}
	res, err := storage.LoadResourceFromURI(uri)
	if err != nil {
		return nil
	}
	return res
}
