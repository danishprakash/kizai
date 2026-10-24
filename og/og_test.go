// ABOUTME: Tests for OG image rendering and the cached remote generator.
// ABOUTME: Uses real fixtures under testdata/ and a local HTTP server.
package og

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func assertOGJPEG(t *testing.T, data []byte) {
	t.Helper()
	if len(data) > MaxBytes {
		t.Fatalf("output is %d bytes, want <= %d", len(data), MaxBytes)
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("output does not decode: %v", err)
	}
	if format != "jpeg" {
		t.Fatalf("format = %s, want jpeg", format)
	}
	if cfg.Width != Width || cfg.Height != Height {
		t.Fatalf("size = %dx%d, want %dx%d", cfg.Width, cfg.Height, Width, Height)
	}
}

func TestRenderPortraitWebP(t *testing.T) {
	out, err := Render(readFixture(t, "portrait.webp"))
	if err != nil {
		t.Fatal(err)
	}
	assertOGJPEG(t, out)
}

func TestRenderLandscapeJPEG(t *testing.T) {
	out, err := Render(readFixture(t, "landscape.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	assertOGJPEG(t, out)
}

func TestRenderCropsCenter(t *testing.T) {
	// 3000x630 image: left and right thirds red, middle green.
	// A center crop to 1.9:1 keeps only green at the center pixel.
	img := image.NewRGBA(image.Rect(0, 0, 3000, 630))
	for y := 0; y < 630; y++ {
		for x := 0; x < 3000; x++ {
			c := color.RGBA{255, 0, 0, 255}
			if x >= 1000 && x < 2000 {
				c = color.RGBA{0, 255, 0, 255}
			}
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	out, err := Render(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	assertOGJPEG(t, out)
	dec, err := jpeg.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	r, g, _, _ := dec.At(Width/2, Height/2).RGBA()
	if g>>8 < 200 || r>>8 > 50 {
		t.Fatalf("center pixel r=%d g=%d, want green", r>>8, g>>8)
	}
	// The crop is 1200 wide from a 3000 wide image scaled 1:1, so
	// the left edge (x=900 in source) is red.
	r, g, _, _ = dec.At(5, Height/2).RGBA()
	if r>>8 < 200 || g>>8 > 50 {
		t.Fatalf("edge pixel r=%d g=%d, want red", r>>8, g>>8)
	}
}

func TestRenderNoisyImageStaysUnderLimit(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	img := image.NewRGBA(image.Rect(0, 0, 2400, 1260))
	rng.Read(img.Pix)
	for i := 3; i < len(img.Pix); i += 4 {
		img.Pix[i] = 255
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	out, err := Render(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	assertOGJPEG(t, out)
}

func TestRenderRejectsGarbage(t *testing.T) {
	if _, err := Render([]byte("not an image")); err == nil {
		t.Fatal("expected error for non-image input")
	}
}

func TestGeneratorFetchesAndCaches(t *testing.T) {
	fixture := readFixture(t, "portrait.webp")
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Write(fixture)
	}))
	defer srv.Close()

	g := &Generator{Client: srv.Client(), CacheDir: t.TempDir()}
	url := srv.URL + "/2026/Kasauli/a.webp"

	first, err := g.Generate(url)
	if err != nil {
		t.Fatal(err)
	}
	assertOGJPEG(t, first)

	srv.Close()
	second, err := g.Generate(url)
	if err != nil {
		t.Fatalf("cached generate failed with server down: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("cached output differs from first output")
	}
	if atomic.LoadInt32(&hits) != 1 {
		t.Fatalf("server hits = %d, want 1", atomic.LoadInt32(&hits))
	}
}

func TestGeneratorHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()

	g := &Generator{Client: srv.Client(), CacheDir: t.TempDir()}
	if _, err := g.Generate(srv.URL + "/missing.webp"); err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestGeneratorSendsUserAgent(t *testing.T) {
	// imgur rejects Go's default User-Agent with 429
	fixture := readFixture(t, "portrait.webp")
	var ua string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua = r.UserAgent()
		w.Write(fixture)
	}))
	defer srv.Close()

	g := &Generator{Client: srv.Client(), CacheDir: t.TempDir()}
	if _, err := g.Generate(srv.URL + "/a.webp"); err != nil {
		t.Fatal(err)
	}
	if ua != UserAgent {
		t.Fatalf("User-Agent = %q, want %q", ua, UserAgent)
	}
}
