package online

// Item represents a wallpaper from an online source.
type Item struct {
	ID         string
	PageURL    string
	ThumbURL   string
	ImageURL   string
	Resolution string
	Source     string
}

type SearchResult struct {
	Items      []Item
	Page       int
	LastPage   int
	Total      int
}
