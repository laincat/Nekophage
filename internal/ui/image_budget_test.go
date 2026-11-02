package ui

import "testing"

func TestCoverBudgetCountsCapacityAndReplacements(t *testing.T) {
	c := newImageCache()
	c.coverLimit = 10
	c.finishCover("a", make([]byte, 3, 8))
	c.finishCover("b", make([]byte, 6))
	if _, ok := c.covers["a"]; ok {
		t.Fatal("the old download survived the byte budget")
	}
	if c.coverBytes != 6 {
		t.Fatalf("retained download bytes = %d, want 6", c.coverBytes)
	}
	c.finishCover("b", make([]byte, 2, 3))
	if c.coverBytes != 3 {
		t.Fatalf("replacement cost = %d, want 3", c.coverBytes)
	}
	c.finishCover("oversized", make([]byte, 11))
	if c.coverBytes != 3 || c.want("oversized", "https://example.test/large.png") {
		t.Fatal("an oversized entry retained bytes or was retried immediately")
	}
}

func TestBitmapBudgetIncludesEncodedDataAndEvictsOldest(t *testing.T) {
	c := newImageCache()
	data := pngBytes(t)
	cost := int64(cap(data)) + 4 // A one-pixel RGBA image has no smaller mip levels.
	c.bitmapLimit = cost
	for _, name := range []string{"a", "b"} {
		c.finishCover(name, data)
		if c.decoded(name) == nil {
			t.Fatalf("%s did not decode", name)
		}
	}
	if c.bitmapBytes != cost || c.coverBytes != 0 {
		t.Fatalf("bitmap/download costs = %d/%d, want %d/0", c.bitmapBytes, c.coverBytes, cost)
	}
	if c.decoded("a") != nil || !c.want("a", "https://example.test/a.png") {
		t.Fatal("an evicted bitmap was retained or could not be downloaded again")
	}
	if c.decoded("b") == nil || c.want("b", "https://example.test/b.png") {
		t.Fatal("the newest bitmap was lost or downloaded twice")
	}
	if c.bitmapBytes != cost {
		t.Fatal("reusing a bitmap counted its bytes twice")
	}
}

func TestOverBudgetBitmapAndCorruptImageReleaseDownloadedBytes(t *testing.T) {
	c := newImageCache()
	data := pngBytes(t)
	c.bitmapLimit = int64(cap(data)) + 3
	c.finishCover("large", data)
	if c.decoded("large") != nil || c.bitmapBytes != 0 || c.coverBytes != 0 {
		t.Fatal("the over-budget image was decoded or kept its downloaded bytes")
	}
	if c.want("large", "https://example.test/large.png") {
		t.Fatal("the over-budget image restarted on the next frame")
	}
	c.finishCover("bad", []byte("not an image"))
	if c.decoded("bad") != nil || c.coverBytes != 0 {
		t.Fatal("a corrupt image retained its downloaded bytes")
	}
}

func TestBitmapCostIncludesOddMipLevelsWithoutOverflow(t *testing.T) {
	if got := bitmapCost(3, 1, 10, 100); got != 34 { // 3x1, 2x1, 1x1 RGBA + encoded.
		t.Fatalf("odd-width bitmap cost = %d, want 34", got)
	}
	largest := int(^uint(0) >> 1)
	if got := bitmapCost(largest, largest, 0, maxBitmapBytes); got != maxBitmapBytes+1 {
		t.Fatalf("huge header cost = %d, want over budget", got)
	}
}

func TestBitmapIsNotDownloadedAgainAfterItsMarkerIsEvicted(t *testing.T) {
	c := newImageCache()
	c.finishCover("visible", pngBytes(t))
	if c.decoded("visible") == nil {
		t.Fatal("the visible image did not decode")
	}
	for i := 0; i <= maxCovers; i++ {
		c.finishCover("failed/"+itoa(i), nil)
	}
	if c.decoded("visible") == nil || c.want("visible", "https://example.test/visible.png") {
		t.Fatal("evicting a download marker caused a cached bitmap to be downloaded again")
	}
}

func TestBitmapEvictionDoesNotLeaveOldDownloadPositions(t *testing.T) {
	c := newImageCache()
	data := pngBytes(t)
	c.bitmapLimit = int64(cap(data)) + 4
	for i := 0; i < maxCovers+10; i++ {
		name := itoa(i % 2)
		c.finishCover(name, data)
		if c.decoded(name) == nil {
			t.Fatalf("revisited key %s disappeared at iteration %d", name, i)
		}
	}
	if len(c.coverOrder) != 1 || len(c.bitmapOrder) != 1 {
		t.Fatalf("one cached bitmap retained %d download positions and %d bitmap positions", len(c.coverOrder), len(c.bitmapOrder))
	}
}
