// ABOUTME: Tests for remote image dimension lookup and its cache.
// ABOUTME: Reuses the og package fixtures and a local HTTP server.
package imgsize

import (
	"image"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func serveFixture(t *testing.T, name string, hits *int32) *httptest.Server {
	t.Helper()
	path := filepath.Join("..", "og", "testdata", name)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(hits, 1)
		http.ServeFile(w, r, path)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func fixtureSize(t *testing.T, name string) Size {
	t.Helper()
	f, err := os.Open(filepath.Join("..", "og", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		t.Fatal(err)
	}
	return Size{Width: cfg.Width, Height: cfg.Height}
}

func TestGetFetchesAndCaches(t *testing.T) {
	var hits int32
	srv := serveFixture(t, "portrait.webp", &hits)
	p := &Prober{Client: srv.Client(), CacheDir: t.TempDir()}
	url := srv.URL + "/2026/Kasauli/a.webp"
	want := fixtureSize(t, "portrait.webp")

	got, err := p.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("size = %+v, want %+v", got, want)
	}

	srv.Close()
	got, err = p.Get(url)
	if err != nil {
		t.Fatalf("cached get failed with server down: %v", err)
	}
	if got != want {
		t.Fatalf("cached size = %+v, want %+v", got, want)
	}
	if atomic.LoadInt32(&hits) != 1 {
		t.Fatalf("server hits = %d, want 1", atomic.LoadInt32(&hits))
	}
}

func TestGetAllKeepsOrderAndSkipsFailures(t *testing.T) {
	var hits int32
	srv := serveFixture(t, "landscape.jpg", &hits)
	missing := httptest.NewServer(http.NotFoundHandler())
	defer missing.Close()

	p := &Prober{Client: srv.Client(), CacheDir: t.TempDir()}
	want := fixtureSize(t, "landscape.jpg")

	got := p.GetAll([]string{srv.URL + "/a.jpg", missing.URL + "/b.jpg", srv.URL + "/c.jpg"})
	if got[0] != want || got[2] != want {
		t.Fatalf("sizes = %+v, want %+v at 0 and 2", got, want)
	}
	if got[1] != (Size{}) {
		t.Fatalf("failed lookup = %+v, want zero", got[1])
	}
}
