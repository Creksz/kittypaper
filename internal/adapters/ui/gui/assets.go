package gui

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed assets/logo.png
var logoPNG []byte

func logoResource() fyne.Resource {
	return fyne.NewStaticResource("kittypaper-logo.png", logoPNG)
}
