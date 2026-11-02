package ui

import (
	"bytes"
	"context"
	"image"
	"slices"
	"sync"

	"github.com/egoist/mygo/ui"
)

const (
	maxCovers         = 400
	maxBitmaps        = 160
	maxCoverBytes     = 32 << 20
	maxBitmapBytes    = 128 << 20
	maxImageDownloads = 4
)

// imageDownloads limits source image work independently of search. Queued work
// observes its timeout too, without opening a connection after cancellation.
// The zero value is ready to use, including in App values made by tests.
type imageDownloads struct {
	once  sync.Once
	slots chan struct{}
}

func (d *imageDownloads) do(ctx context.Context, fetch func(context.Context) ([]byte, string, error)) ([]byte, string, error) {
	d.once.Do(func() { d.slots = make(chan struct{}, maxImageDownloads) })
	select {
	case <-ctx.Done():
		return nil, "", ctx.Err()
	case d.slots <- struct{}{}:
	}
	defer func() { <-d.slots }()
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	return fetch(ctx)
}

// imageCache retains downloaded bytes and decoded bitmaps under separate byte
// and entry budgets. The bitmap budget reserves RGBA pixels, the full mipmap
// pyramid, and the encoded data MyGo retains for re-decoding. It is a cache
// estimate, not a bound on renderer textures or the whole process's memory.
//
// A nil cover remembers a failed or already-decoded download. App.mu protects
// downloads; decoding runs on the window thread, as MyGo requires.
type imageCache struct {
	covers      map[string][]byte
	pending     map[string]bool
	bitmaps     map[string]*ui.Bitmap
	bitmapCosts map[string]int64

	coverOrder  []string
	bitmapOrder []string
	coverBytes  int64
	bitmapBytes int64
	coverLimit  int64
	bitmapLimit int64
}

func newImageCache() imageCache {
	return imageCache{
		covers:      map[string][]byte{},
		pending:     map[string]bool{},
		bitmaps:     map[string]*ui.Bitmap{},
		bitmapCosts: map[string]int64{},
		coverLimit:  maxCoverBytes,
		bitmapLimit: maxBitmapBytes,
	}
}

func (c *imageCache) want(name, rawURL string) bool {
	if rawURL == "" {
		return false
	}
	if _, ok := c.bitmaps[name]; ok {
		return false
	}
	if _, ok := c.covers[name]; ok {
		return false
	}
	if c.pending[name] {
		return false
	}
	c.pending[name] = true
	return true
}

func (c *imageCache) finishCover(name string, data []byte) {
	delete(c.pending, name)
	if _, ok := c.covers[name]; !ok {
		c.coverOrder = append(c.coverOrder, name)
	}
	c.coverBytes -= int64(cap(c.covers[name]))
	// A single over-budget image must not evict everything and then restart
	// its download every frame. Remember it as unavailable for this cache entry.
	if int64(cap(data)) > c.coverLimit {
		data = nil
	}
	c.covers[name] = data
	c.coverBytes += int64(cap(data))
	c.evictCovers()
}

// bitmapCost reserves each possible mip level before decoding. It saturates at
// limit+1 so even a malicious image header cannot overflow the size arithmetic.
func bitmapCost(width, height int, encoded, limit int64) int64 {
	if width <= 0 || height <= 0 || encoded > limit {
		return limit + 1
	}
	cost := encoded
	w, h := int64(width), int64(height)
	for {
		if w > (limit-cost)/4/h {
			return limit + 1
		}
		cost += w * h * 4
		if w == 1 && h == 1 {
			return cost
		}
		w, h = max(1, w/2+w%2), max(1, h/2+h%2)
	}
}

func (c *imageCache) discardCoverBytes(name string) {
	c.coverBytes -= int64(cap(c.covers[name]))
	c.covers[name] = nil
}

func (c *imageCache) decoded(name string) ui.ImageSource {
	if b, ok := c.bitmaps[name]; ok {
		return b
	}
	data, ok := c.covers[name]
	if !ok || data == nil {
		return nil
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		c.discardCoverBytes(name)
		return nil
	}
	cost := bitmapCost(config.Width, config.Height, int64(cap(data)), c.bitmapLimit)
	if cost > c.bitmapLimit {
		c.discardCoverBytes(name)
		return nil
	}
	b, err := ui.DecodeBitmap(data)
	if err != nil {
		c.discardCoverBytes(name)
		return nil
	}
	c.bitmapOrder = append(c.bitmapOrder, name)
	c.bitmaps[name] = b
	c.bitmapCosts[name] = cost
	c.bitmapBytes += cost
	// MyGo keeps data in its decode closure. Move its cost to the bitmap stage;
	// clearing this map entry alone does not free those encoded bytes.
	c.discardCoverBytes(name)
	c.evictBitmaps()
	return b
}

func (c *imageCache) evictCovers() {
	for len(c.coverOrder) > maxCovers || c.coverBytes > c.coverLimit {
		oldest := c.coverOrder[0]
		c.coverOrder = c.coverOrder[1:]
		c.coverBytes -= int64(cap(c.covers[oldest]))
		delete(c.covers, oldest)
	}
}

func (c *imageCache) evictBitmaps() {
	for len(c.bitmapOrder) > maxBitmaps || c.bitmapBytes > c.bitmapLimit {
		oldest := c.bitmapOrder[0]
		c.bitmapOrder = c.bitmapOrder[1:]
		delete(c.bitmaps, oldest)
		c.bitmapBytes -= c.bitmapCosts[oldest]
		delete(c.bitmapCosts, oldest)
		if data, ok := c.covers[oldest]; ok && data == nil {
			delete(c.covers, oldest)
			// Remove the order entry too. A later re-download of this same key
			// must not inherit an obsolete position and evict the new result.
			if i := slices.Index(c.coverOrder, oldest); i >= 0 {
				c.coverOrder = slices.Delete(c.coverOrder, i, i+1)
			}
		}
	}
}

func (c *imageCache) entries() (covers, bitmaps int) {
	return len(c.covers), len(c.bitmaps)
}
