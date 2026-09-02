package library

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"kittypaper/internal/platform/filex"
)

const maxRecent = 50

type Data struct {
	Favorites   []string            `json:"favorites"`
	Recent      []string            `json:"recent"`
	Collections map[string][]string `json:"collections"`
}

type Store struct {
	Path string
}

func (s Store) load() (Data, error) {
	data := Data{Collections: map[string][]string{}}
	raw, err := os.ReadFile(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return data, nil
		}
		return data, err
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return Data{Collections: map[string][]string{}}, err
	}
	if data.Collections == nil {
		data.Collections = map[string][]string{}
	}
	return data, nil
}

func (s Store) save(data Data) error {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return err
	}
	if data.Collections == nil {
		data.Collections = map[string][]string{}
	}
	payload, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return filex.WriteFileAtomic(s.Path, append(payload, '\n'), 0o644)
}

func (s Store) IsFavorite(path string) (bool, error) {
	data, err := s.load()
	if err != nil {
		return false, err
	}
	path = filepath.Clean(path)
	for _, p := range data.Favorites {
		if filepath.Clean(p) == path {
			return true, nil
		}
	}
	return false, nil
}

func (s Store) ToggleFavorite(path string) (bool, error) {
	data, err := s.load()
	if err != nil {
		return false, err
	}
	path = filepath.Clean(path)
	for i, p := range data.Favorites {
		if filepath.Clean(p) == path {
			data.Favorites = append(data.Favorites[:i], data.Favorites[i+1:]...)
			return false, s.save(data)
		}
	}
	data.Favorites = append(data.Favorites, path)
	return true, s.save(data)
}

func (s Store) Favorites() ([]string, error) {
	data, err := s.load()
	if err != nil {
		return nil, err
	}
	return append([]string(nil), data.Favorites...), nil
}

func (s Store) AddRecent(path string) error {
	data, err := s.load()
	if err != nil {
		return err
	}
	path = filepath.Clean(path)
	next := []string{path}
	for _, p := range data.Recent {
		if filepath.Clean(p) == path {
			continue
		}
		next = append(next, p)
		if len(next) >= maxRecent {
			break
		}
	}
	data.Recent = next
	return s.save(data)
}

func (s Store) Recent() ([]string, error) {
	data, err := s.load()
	if err != nil {
		return nil, err
	}
	return append([]string(nil), data.Recent...), nil
}

func (s Store) FavoriteCount() (int, error) {
	favs, err := s.Favorites()
	if err != nil {
		return 0, err
	}
	return len(favs), nil
}

func (s Store) CollectionNames() ([]string, error) {
	data, err := s.load()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(data.Collections))
	for name := range data.Collections {
		names = append(names, name)
	}
	return names, nil
}

func (s Store) CollectionPaths(name string) ([]string, error) {
	data, err := s.load()
	if err != nil {
		return nil, err
	}
	paths, ok := data.Collections[name]
	if !ok {
		return nil, fmt.Errorf("collection not found: %s", name)
	}
	return append([]string(nil), paths...), nil
}

func (s Store) AddToCollection(name, path string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("collection name is required")
	}
	data, err := s.load()
	if err != nil {
		return err
	}
	path = filepath.Clean(path)
	for _, p := range data.Collections[name] {
		if filepath.Clean(p) == path {
			return nil
		}
	}
	data.Collections[name] = append(data.Collections[name], path)
	return s.save(data)
}
