package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// pngBytes is a real one-pixel PNG: the cache must be tested with bytes the
// decoder accepts, or "decoding failed" would look like a cache miss.
func pngBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 200, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestImageCacheStartsEachDownloadOnce(t *testing.T) {
	c := newImageCache()
	if c.want("a", "") {
		t.Fatal("an empty URL started a download")
	}
	if !c.want("a", "https://example.test/a.png") {
		t.Fatal("a new image was not requested")
	}
	if c.want("a", "https://example.test/a.png") {
		t.Fatal("an in-flight image was requested twice")
	}
	c.finishCover("a", pngBytes(t))
	if c.want("a", "https://example.test/a.png") {
		t.Fatal("a downloaded image was requested again")
	}
	// A failure is remembered as well: a broken cover must not be fetched on
	// every frame.
	c.finishCover("b", nil)
	if c.want("b", "https://example.test/b.png") {
		t.Fatal("a failed image was requested again")
	}
}

func TestImageCacheDecodesOnceAndDropsTheDownloadReference(t *testing.T) {
	c := newImageCache()
	c.finishCover("a", pngBytes(t))
	if c.decoded("a") == nil {
		t.Fatal("a valid PNG did not decode")
	}
	covers, bitmaps := c.entries()
	if bitmaps != 1 {
		t.Fatalf("bitmaps = %d, want 1", bitmaps)
	}
	if data, ok := c.covers["a"]; !ok || data != nil {
		t.Fatalf("decoded download still holds bytes: %v, %v", data, ok)
	}
	if covers != 1 {
		t.Logf("covers = %d after decoding (a nil marker is expected)", covers)
	}
	if c.decoded("a") == nil {
		t.Fatal("the decoded bitmap was not reused")
	}
}

func TestImageCacheRemembersACorruptImageAsFailed(t *testing.T) {
	c := newImageCache()
	c.finishCover("bad", []byte("not an image"))
	if c.decoded("bad") != nil {
		t.Fatal("garbage decoded as an image")
	}
	if data, ok := c.covers["bad"]; !ok || data != nil {
		t.Fatalf("corrupt image was not marked as failed: %v, %v", data, ok)
	}
}

func TestImageCacheStaysWithinItsBounds(t *testing.T) {
	c := newImageCache()
	data := pngBytes(t)
	for i := 0; i < maxBitmaps+20; i++ {
		name := string(rune('a'+i%26)) + "/" + itoa(i)
		c.finishCover(name, data)
		if c.decoded(name) == nil {
			t.Fatalf("%s did not decode", name)
		}
	}
	_, bitmaps := c.entries()
	if bitmaps > maxBitmaps {
		t.Fatalf("bitmaps = %d, above the bound %d", bitmaps, maxBitmaps)
	}
	if bitmaps < maxBitmaps/2 {
		t.Fatalf("bitmaps = %d: the bound evicted far too aggressively", bitmaps)
	}
	// The oldest entries are gone and will be fetched again when shown.
	if c.decoded("a/0") != nil {
		t.Fatal("the oldest bitmap survived the entry bound")
	}
	if !c.want("a/0", "https://example.test/again.png") {
		t.Fatal("an evicted bitmap was not requested again")
	}
}

func TestImageCacheBoundsDownloadedBytes(t *testing.T) {
	c := newImageCache()
	data := pngBytes(t)
	for i := 0; i < maxCovers+20; i++ {
		c.finishCover(itoa(i), data)
	}
	covers, _ := c.entries()
	if covers > maxCovers {
		t.Fatalf("covers = %d, above the bound %d", covers, maxCovers)
	}
}
