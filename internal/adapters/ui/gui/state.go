package gui

type librarySection int

const (
	sectionAll librarySection = iota
	sectionFavorites
	sectionRecent
	sectionOnline
	sectionCollections
)

func (s librarySection) label() string {
	switch s {
	case sectionAll:
		return "All"
	case sectionFavorites:
		return "Favorites"
	case sectionRecent:
		return "Recent"
	case sectionOnline:
		return "Online"
	case sectionCollections:
		return "Collections"
	default:
		return "All"
	}
}
