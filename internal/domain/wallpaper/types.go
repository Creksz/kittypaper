package wallpaper

import "time"

type ID string

type Item struct {
	ID        ID
	Path      string
	Width     int
	Height    int
	UpdatedAt time.Time
}

type SelectionMode string

const (
	SelectionExact  SelectionMode = "exact"
	SelectionRandom SelectionMode = "random"
	SelectionNext   SelectionMode = "next"
	SelectionBack   SelectionMode = "back"
	SelectionProv   SelectionMode = "prov"
	SelectionPrev   SelectionMode = "prev"
)
