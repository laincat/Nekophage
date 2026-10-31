package ui

import (
	"github.com/egoist/mygo/ui"
)

// Bounds on the image caches. Without them a long browse session keeps every
// cover it ever drew — compressed bytes first, then decoded bitmaps — for the
// life of the window, and a memory limit that only exists in the allocator's
// imagination is not a limit.
const (
	// maxCovers bounds downloaded bytes and "this one failed" markers.
	maxCovers = 400
	// maxBitmaps bounds decoded images, which dominate memory: a cover is a
	// few tens of kilobytes compressed and several megabytes decoded.
	maxBitmaps = 160
)

// imageCache holds the two stages of an image: the downloaded bytes and the
// decoded bitmap the renderer can draw.
//
// Keys are shared between the stages ("source/id" for covers, "page/…" for
// kept pages). Entries are evicted oldest first; a failed download is stored as
// nil so a broken cover is not retried on every frame, and it counts toward the
// bound like any other entry.
//
// Callers hold App.mu for the download side and render on the window's thread
// for the decode side, so this type does not lock: the map is only touched
// where the window is, and the two stages never change key sets concurrently.
type imageCache struct {
	covers  map[string][]byte
	pending map[string]bool
	bitmaps map[string]*ui.Bitmap

	coverOrder  []string
	bitmapOrder []string
}

func newImageCache() imageCache {
	return imageCache{
		covers:  map[string][]byte{},
		pending: map[string]bool{},
		bitmaps: map[string]*ui.Bitmap{},
	}
}

// want records that a download for name should start. It reports false when the
// image is already cached, already downloading, or has no URL.
func (c *imageCache) want(name, rawURL string) bool {
	if rawURL == "" {
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

// finishCover stores a download's result, failed or not.
func (c *imageCache) finishCover(name string, data []byte) {
	delete(c.pending, name)
	if _, ok := c.covers[name]; !ok {
		c.coverOrder = append(c.coverOrder, name)
	}
	c.covers[name] = data
	c.evictCovers()
}

// cover returns the downloaded bytes, and whether the entry exists at all.
// decoded returns the bitmap for name, decoding the downloaded bytes once.
// It returns nil while the download is still running or after a failure.
func (c *imageCache) decoded(name string) ui.ImageSource {
	if b, ok := c.bitmaps[name]; ok {
		return b
	}
	data, ok := c.covers[name]
	if !ok || data == nil {
		return nil
	}
	b, err := ui.DecodeBitmap(data)
	if err != nil {
		// A corrupt image is remembered as a failure instead of being decoded
		// again on every frame.
		c.covers[name] = nil
		return nil
	}
	c.bitmapOrder = append(c.bitmapOrder, name)
	c.bitmaps[name] = b
	// The decoded bitmap is now the cache entry. Keeping its compressed
	// download as well only doubles retention across long browsing sessions.
	c.covers[name] = nil
	c.evictBitmaps()
	return b
}

func (c *imageCache) evictCovers() {
	for len(c.coverOrder) > maxCovers {
		oldest := c.coverOrder[0]
		c.coverOrder = c.coverOrder[1:]
		if _, ok := c.covers[oldest]; ok {
			delete(c.covers, oldest)
		}
	}
}

func (c *imageCache) evictBitmaps() {
	for len(c.bitmapOrder) > maxBitmaps {
		oldest := c.bitmapOrder[0]
		c.bitmapOrder = c.bitmapOrder[1:]
		delete(c.bitmaps, oldest)
		// The download marker goes with it: coming back to this cover should
		// fetch it again rather than show a permanently blank frame.
		if data, ok := c.covers[oldest]; ok && data == nil {
			delete(c.covers, oldest)
		}
	}
}

// entries reports how many of each stage are held, for tests and diagnostics.
func (c *imageCache) entries() (covers, bitmaps int) {
	return len(c.covers), len(c.bitmaps)
}
