package gui

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"os"
	"path/filepath"
	"sync"

	"fyne.io/fyne/v2"
	"github.com/nfnt/resize"
	_ "golang.org/x/image/webp"
)

const (
	thumbListW      = 96
	thumbListH      = 72
	thumbPreviewMax = 1280
)

type thumbnailCache struct {
	dir      string
	mu       sync.RWMutex
	mem      map[string]fyne.Resource
	inflight map[string][]func(fyne.Resource)
}

func newThumbnailCache(dir string) *thumbnailCache {
	_ = os.MkdirAll(filepath.Join(dir, "thumbs"), 0o755)
	return &thumbnailCache{
		dir:      dir,
		mem:      make(map[string]fyne.Resource),
		inflight: make(map[string][]func(fyne.Resource)),
	}
}

func (c *thumbnailCache) listThumb(path string, cb func(fyne.Resource)) {
	c.load(path, thumbListW, thumbListH, resize.NearestNeighbor, cb)
}

func (c *thumbnailCache) previewThumb(path string, cb func(fyne.Resource)) {
	c.load(path, thumbPreviewMax, thumbPreviewMax, resize.Bilinear, cb)
}

func (c *thumbnailCache) load(path string, maxW, maxH int, filter resize.InterpolationFunction, cb func(fyne.Resource)) {
	if path == "" {
		fyne.Do(func() { cb(nil) })
		return
	}
	key := cacheKey(path, maxW, maxH)
	if res, ok := c.cached(key); ok {
		fyne.Do(func() { cb(res) })
		return
	}

	c.mu.Lock()
	if waits, busy := c.inflight[key]; busy {
		c.inflight[key] = append(waits, cb)
		c.mu.Unlock()
		return
	}
	c.inflight[key] = []func(fyne.Resource){cb}
	c.mu.Unlock()

	go func() {
		res := c.generate(path, maxW, maxH, key, filter)
		c.mu.Lock()
		if res != nil {
			c.mem[key] = res
		}
		waiters := c.inflight[key]
		delete(c.inflight, key)
		c.mu.Unlock()
		fyne.Do(func() {
			for _, w := range waiters {
				w(res)
			}
		})
	}()
}

func (c *thumbnailCache) cached(key string) (fyne.Resource, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	res, ok := c.mem[key]
	return res, ok
}

func cacheKey(path string, maxW, maxH int) string {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Sprintf("%s|%d|%d", path, maxW, maxH)
	}
	return fmt.Sprintf("%s|%d|%d|%d", path, maxW, maxH, info.ModTime().UnixNano())
}

func (c *thumbnailCache) diskPath(key string) string {
	h := sha256.Sum256([]byte(key))
	return filepath.Join(c.dir, "thumbs", hex.EncodeToString(h[:16])+".png")
}

func (c *thumbnailCache) generate(path string, maxW, maxH int, key string, filter resize.InterpolationFunction) fyne.Resource {
	disk := c.diskPath(key)
	if raw, err := os.ReadFile(disk); err == nil {
		return fyne.NewStaticResource(filepath.Base(disk), raw)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	src, _, err := image.Decode(f)
	if err != nil {
		return nil
	}
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= 0 || h <= 0 {
		return nil
	}

	targetW, targetH := fitSize(w, h, maxW, maxH)
	if targetW >= w && targetH >= h {
		targetW, targetH = w, h
	}
	dst := resize.Thumbnail(uint(targetW), uint(targetH), src, filter)

	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return nil
	}
	data := buf.Bytes()
	_ = os.WriteFile(disk, data, 0o644)
	return fyne.NewStaticResource(filepath.Base(path), data)
}

func fitSize(w, h, maxW, maxH int) (int, int) {
	if w <= maxW && h <= maxH {
		return w, h
	}
	wf, hf := float64(w), float64(h)
	scale := wf / float64(maxW)
	if hf/float64(maxH) > scale {
		scale = hf / float64(maxH)
	}
	return int(wf / scale), int(hf / scale)
}

type remoteThumbCache struct {
	mu   sync.RWMutex
	mem  map[string]fyne.Resource
	busy map[string][]func(fyne.Resource)
}

func newRemoteThumbCache() *remoteThumbCache {
	return &remoteThumbCache{
		mem:  make(map[string]fyne.Resource),
		busy: make(map[string][]func(fyne.Resource)),
	}
}

func (c *remoteThumbCache) load(url string, cb func(fyne.Resource)) {
	if url == "" {
		fyne.Do(func() { cb(nil) })
		return
	}
	c.mu.RLock()
	if res, ok := c.mem[url]; ok {
		c.mu.RUnlock()
		fyne.Do(func() { cb(res) })
		return
	}
	c.mu.RUnlock()

	c.mu.Lock()
	if waits, ok := c.busy[url]; ok {
		c.busy[url] = append(waits, cb)
		c.mu.Unlock()
		return
	}
	c.busy[url] = []func(fyne.Resource){cb}
	c.mu.Unlock()

	go func() {
		res := loadRemoteResource(url)
		c.mu.Lock()
		if res != nil {
			c.mem[url] = res
		}
		waiters := c.busy[url]
		delete(c.busy, url)
		c.mu.Unlock()
		fyne.Do(func() {
			for _, w := range waiters {
				w(res)
			}
		})
	}()
}
