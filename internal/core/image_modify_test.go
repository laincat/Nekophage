package core

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

// A source may attach a modifyImage script to an image: jm reorders the blocks
// of a long strip that way and several sources rotate a page. The script runs
// over the decoded picture, so what comes back must be the script's output, not
// the bytes the server sent.
func TestModifyImageScriptIsApplied(t *testing.T) {
	// A 4x2 picture: red column first, blue column second.
	src := image.NewRGBA(image.Rect(0, 0, 4, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 4; x++ {
			if x < 2 {
				src.Set(x, y, color.RGBA{R: 255, A: 255})
			} else {
				src.Set(x, y, color.RGBA{B: 255, A: 255})
			}
		}
	}
	var body bytes.Buffer
	if err := png.Encode(&body, src); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(body.Bytes())
	}))
	defer srv.Close()

	c, err := Open(Options{DataDir: filepath.Join(t.TempDir(), "data"), AppVersion: "1.6.0", Locale: "zh_CN"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	script := `
class Rot extends ComicSource {
    name = "Rot"; key = "rot"; version = "1.0.0"
    comic = {
        onImageLoad: (url) => ({
            url: url,
            modifyImage: "let modifyImage = (image) => image.copyAndRotate90()",
        })
    }
}
`
	if _, err := c.Install(context.Background(), script, "test", ""); err != nil {
		t.Fatalf("install: %v", err)
	}

	got, ctype, err := c.FetchImage(context.Background(), "rot", srv.URL+"/1.png", "c1", "e1")
	if err != nil {
		t.Fatalf("fetch image: %v", err)
	}
	// The script turned the 4x2 picture into a 2x4 one.
	if ctype != "image/png" {
		t.Errorf("content type = %q, want image/png after re-encoding", ctype)
	}
	out, err := png.Decode(bytes.NewReader(got))
	if err != nil {
		t.Fatalf("the result is not a PNG: %v", err)
	}
	if b := out.Bounds(); b.Dx() != 2 || b.Dy() != 4 {
		t.Fatalf("result is %dx%d, want 2x4", b.Dx(), b.Dy())
	}
	// A clockwise quarter turn puts the red column on top.
	r, _, _, _ := out.At(0, 0).RGBA()
	if r < 0xF000 {
		t.Errorf("the top pixel is %v, want red", out.At(0, 0))
	}
}

// A script that cannot produce an image must fail loudly rather than hand the
// original bytes over as if the source's instruction had been honoured.
func TestModifyImageFailureIsReported(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 2))
	var body bytes.Buffer
	if err := png.Encode(&body, src); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(body.Bytes())
	}))
	defer srv.Close()

	c, err := Open(Options{DataDir: filepath.Join(t.TempDir(), "data"), AppVersion: "1.6.0", Locale: "zh_CN"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	script := `
class Bad extends ComicSource {
    name = "Bad"; key = "bad"; version = "1.0.0"
    comic = {
        onImageLoad: (url) => ({ url: url, modifyImage: "let modifyImage = () => 'nope'" })
    }
}
`
	if _, err := c.Install(context.Background(), script, "test", ""); err != nil {
		t.Fatalf("install: %v", err)
	}
	if _, _, err := c.FetchImage(context.Background(), "bad", srv.URL+"/1.png", "c1", "e1"); err == nil {
		t.Error("a script that returns no image was accepted")
	}
}

// A source may pair a pixel script with a format this build cannot read —
// hitomi serves avif. The script cannot run, but failing the page over it would
// hide an image the reader could show as it arrived.
func TestModifyImagePassesThroughAnUnknownFormat(t *testing.T) {
	avif := append([]byte{0, 0, 0, 0x1C, 'f', 't', 'y', 'p', 'a', 'v', 'i', 'f'}, make([]byte, 40)...)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/avif")
		_, _ = w.Write(avif)
	}))
	defer srv.Close()

	c, err := Open(Options{DataDir: filepath.Join(t.TempDir(), "data"), AppVersion: "1.6.0", Locale: "zh_CN"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	script := `
class Avif extends ComicSource {
    name = "Avif"; key = "avif"; version = "1.0.0"
    comic = {
        onImageLoad: (url) => ({ url: url, modifyImage: "let modifyImage = (i) => i" })
    }
}
`
	if _, err := c.Install(context.Background(), script, "test", ""); err != nil {
		t.Fatalf("install: %v", err)
	}
	got, ctype, err := c.FetchImage(context.Background(), "avif", srv.URL+"/1.avif", "c1", "e1")
	if err != nil {
		t.Fatalf("fetch image: %v", err)
	}
	if !bytes.Equal(got, avif) {
		t.Errorf("the body changed: %d bytes", len(got))
	}
	if ctype != "image/avif" {
		t.Errorf("content type = %q, want the server's own", ctype)
	}
}

// The decoders that must be registered are easy to forget: with only png
// registered, a jpeg page silently took the "cannot decode" path and the
// script never ran. The type of the result is what proves it ran.
func TestModifyImageRunsOnJPEGBytes(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 3, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 3; x++ {
			src.Set(x, y, color.RGBA{G: 200, A: 255})
		}
	}
	var body bytes.Buffer
	if err := jpeg.Encode(&body, src, nil); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write(body.Bytes())
	}))
	defer srv.Close()

	c, err := Open(Options{DataDir: filepath.Join(t.TempDir(), "data"), AppVersion: "1.6.0", Locale: "zh_CN"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	script := `
class Jpg extends ComicSource {
    name = "Jpg"; key = "jpg"; version = "1.0.0"
    comic = {
        onImageLoad: (url) => ({ url: url, modifyImage: "let modifyImage = (image) => image.copyAndRotate90()" })
    }
}
`
	if _, err := c.Install(context.Background(), script, "test", ""); err != nil {
		t.Fatalf("install: %v", err)
	}
	got, ctype, err := c.FetchImage(context.Background(), "jpg", srv.URL+"/1.jpg", "c1", "e1")
	if err != nil {
		t.Fatalf("fetch image: %v", err)
	}
	if ctype != "image/png" {
		t.Fatalf("content type = %q, want image/png: the script did not run", ctype)
	}
	out, err := png.Decode(bytes.NewReader(got))
	if err != nil {
		t.Fatalf("not a PNG: %v", err)
	}
	if b := out.Bounds(); b.Dx() != 2 || b.Dy() != 3 {
		t.Errorf("result is %dx%d, want 2x3", b.Dx(), b.Dy())
	}
}
