// Package probesite serves one image on loopback, for probes that need a chapter
// with real pages but must not depend on a site.
//
// A source script can hand back an image address, so a fixture pointing at
// 127.0.0.1 lets the reader decode a real picture with no network. The picture is
// drawn at runtime rather than pasted as base64: a hand-typed blob that fails to
// decode looks exactly like a client that cannot render.
package probesite

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net"
	"net/http"
	"strconv"
	"time"
)

// ServeOnePage answers every request on 127.0.0.1:port with a small PNG. The
// returned function stops it.
func ServeOnePage(port int) (func(), error) {
	// Taller than any window on purpose: an 8x8 page cannot scroll, so a check
	// that reads the reading position back would always see zero and prove nothing
	// about scrolling. The width matches a phone-ish column so the reader lays it
	// out like a real page.
	const width, height = 320, 1200
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// A band every 100 rows makes a scrolled position visible in a capture,
			// not just a number in a log.
			shade := uint8(0x90)
			if (y/100)%2 == 0 {
				shade = 0x20
			}
			img.Set(x, y, color.RGBA{R: 0x20, G: shade, B: 0x90, A: 0xFF})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	page := buf.Bytes()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", strconv.Itoa(len(page)))
		_, _ = w.Write(page)
	})
	srv := &http.Server{
		Addr:              "127.0.0.1:" + strconv.Itoa(port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return nil, err
	}
	go func() { _ = srv.Serve(ln) }()
	return func() { _ = srv.Close() }, nil
}
