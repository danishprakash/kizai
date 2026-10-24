// ABOUTME: Integration tests for kizai build: runs Build() against a
// ABOUTME: temporary site and checks generated OG images and meta data.
package command

import (
	"bytes"
	"image"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/danishprakash/kizai/og"
	"github.com/sirupsen/logrus"
)

const testTemplate = `OG={{ .OGImage }}
URL={{ .CanonicalURL }}
`

type imageServer struct {
	*httptest.Server
	mu    sync.Mutex
	paths []string
}

func newImageServer(t *testing.T) *imageServer {
	t.Helper()
	fixture, err := os.ReadFile(filepath.Join("..", "og", "testdata", "landscape.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	s := &imageServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.paths = append(s.paths, r.URL.Path)
		s.mu.Unlock()
		if strings.Contains(r.URL.Path, "missing") {
			http.NotFound(w, r)
			return
		}
		w.Write(fixture)
	}))
	t.Cleanup(s.Close)
	return s
}

func (s *imageServer) requested(path string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.paths {
		if p == path {
			return true
		}
	}
	return false
}

func writeSiteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// buildSite creates a site in a temp dir, runs Build() in it and
// returns the site root.
func buildSite(t *testing.T, srv *imageServer, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	base := map[string]string{
		"meta.yml":                   "name: Test\nurl: https://example.com\n",
		"static/.keep":               "",
		"templates/photos.html":      testTemplate,
		"templates/post.html":        testTemplate,
		"templates/photography.html": testTemplate,
		"templates/home.html":        testTemplate,
		"templates/feed.html":        "",
		"pages/index.md":             "---\nlayout: home\n---\n",
		"pages/feed/feed.md":         "---\nlayout: feed\n---\n",
		"pages/posts/placeholder.md": "---\nlayout: post\ntitle: P\ndate: 2020-01-01\n---\n",
	}
	for k, v := range files {
		base[k] = v
	}
	for path, content := range base {
		writeSiteFile(t, filepath.Join(root, path), strings.ReplaceAll(content, "SRV", srv.URL))
	}

	prevGen := ogGen
	ogGen = &og.Generator{Client: srv.Client(), CacheDir: t.TempDir()}
	t.Cleanup(func() { ogGen = prevGen })

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(wd) })

	Build()
	return root
}

func readBuilt(t *testing.T, root, path string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "build", path))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func assertOGFile(t *testing.T, root, path string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "build", path))
	if err != nil {
		t.Fatalf("og image not generated: %v", err)
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if format != "jpeg" || cfg.Width != og.Width || cfg.Height != og.Height {
		t.Fatalf("got %s %dx%d", format, cfg.Width, cfg.Height)
	}
}

func assertNoOGFile(t *testing.T, root, path string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(root, "build", path)); !os.IsNotExist(err) {
		t.Fatalf("expected no og image at %s, stat err = %v", path, err)
	}
}

func TestBuildPhotoAlbumUsesIndexCover(t *testing.T) {
	srv := newImageServer(t)
	root := buildSite(t, srv, map[string]string{
		"pages/photos/index.yaml": "---\nlayout: photography\ntitle: Photos\nphotos:\n  - title: Kasauli\n    slug: kasauli\n    cover: SRV/kasauli/cover.webp\n---\n",
		"pages/photos/kasauli.md": "---\nlayout: photos\ntitle: Kasauli\nimages:\n  - SRV/kasauli/first.webp\n  - SRV/kasauli/cover.webp\n---\n",
	})

	assertOGFile(t, root, "photos/kasauli/og.jpg")
	html := readBuilt(t, root, "photos/kasauli/index.html")
	if !strings.Contains(html, "OG=https://example.com/photos/kasauli/og.jpg\n") {
		t.Fatalf("missing og image url:\n%s", html)
	}
	if !strings.Contains(html, "URL=https://example.com/photos/kasauli/\n") {
		t.Fatalf("missing canonical url:\n%s", html)
	}
	if !srv.requested("/kasauli/cover.webp") || srv.requested("/kasauli/first.webp") {
		t.Fatalf("expected only the cover to be fetched, got %v", srv.paths)
	}
}

func TestBuildPhotoAlbumFallsBackToFirstImage(t *testing.T) {
	srv := newImageServer(t)
	root := buildSite(t, srv, map[string]string{
		"pages/photos/hampi.md": "---\nlayout: photos\ntitle: Hampi\nimages:\n  - SRV/hampi/first.webp\n  - SRV/hampi/second.webp\n---\n",
	})

	assertOGFile(t, root, "photos/hampi/og.jpg")
	if !srv.requested("/hampi/first.webp") {
		t.Fatalf("expected first image to be fetched, got %v", srv.paths)
	}
}

func TestBuildFrontmatterOGImageOverrides(t *testing.T) {
	srv := newImageServer(t)
	root := buildSite(t, srv, map[string]string{
		"pages/photos/index.yaml": "---\nlayout: photography\ntitle: Photos\nphotos:\n  - slug: bhutan\n    cover: SRV/bhutan/cover.webp\n---\n",
		"pages/photos/bhutan.md":  "---\nlayout: photos\ntitle: Bhutan\nog_image: SRV/bhutan/chosen.webp\nimages:\n  - SRV/bhutan/cover.webp\n---\n",
	})

	assertOGFile(t, root, "photos/bhutan/og.jpg")
	if !srv.requested("/bhutan/chosen.webp") || srv.requested("/bhutan/cover.webp") {
		t.Fatalf("expected og_image to be fetched, got %v", srv.paths)
	}
}

func TestBuildPostUsesFirstImage(t *testing.T) {
	srv := newImageServer(t)
	root := buildSite(t, srv, map[string]string{
		"pages/posts/trip.md":  "---\nlayout: post\ntitle: Trip\ndate: 2024-01-01\n---\n\nText.\n\n<img src=\"SRV/post/html.png\">\n\n![a](SRV/post/md.png)\n",
		"pages/posts/plain.md": "---\nlayout: post\ntitle: Plain\ndate: 2024-01-02\n---\n\nNo images here.\n",
	})

	assertOGFile(t, root, "posts/trip/og.jpg")
	if !srv.requested("/post/html.png") || srv.requested("/post/md.png") {
		t.Fatalf("expected the first image in the post to be fetched, got %v", srv.paths)
	}
	html := readBuilt(t, root, "posts/trip/index.html")
	if !strings.Contains(html, "OG=https://example.com/posts/trip/og.jpg\n") {
		t.Fatalf("missing og image url:\n%s", html)
	}

	assertNoOGFile(t, root, "posts/plain/og.jpg")
	html = readBuilt(t, root, "posts/plain/index.html")
	if !strings.Contains(html, "OG=\n") {
		t.Fatalf("expected empty og image:\n%s", html)
	}
}

func TestBuildContinuesWhenImageFetchFails(t *testing.T) {
	var logs bytes.Buffer
	logrus.SetOutput(&logs)
	defer logrus.SetOutput(os.Stderr)

	srv := newImageServer(t)
	root := buildSite(t, srv, map[string]string{
		"pages/photos/gone.md": "---\nlayout: photos\ntitle: Gone\nimages:\n  - SRV/missing/a.webp\n---\n",
	})

	if !strings.Contains(logs.String(), "og: skipping build/photos/gone") || !strings.Contains(logs.String(), "404 Not Found") {
		t.Errorf("missing warning log, got: %q", logs.String())
	}

	assertNoOGFile(t, root, "photos/gone/og.jpg")
	html := readBuilt(t, root, "photos/gone/index.html")
	if !strings.Contains(html, "OG=\n") || !strings.Contains(html, "URL=https://example.com/photos/gone/\n") {
		t.Fatalf("unexpected output:\n%s", html)
	}
}

func TestBuildOnlyFetchesAllowedHosts(t *testing.T) {
	var logs bytes.Buffer
	logrus.SetOutput(&logs)
	defer logrus.SetOutput(os.Stderr)

	srv := newImageServer(t)
	// same server, reached through a host that isn't allowed
	other := strings.Replace(srv.URL, "127.0.0.1", "localhost", 1)
	fixture, err := os.ReadFile(filepath.Join("..", "og", "testdata", "portrait.webp"))
	if err != nil {
		t.Fatal(err)
	}
	root := buildSite(t, srv, map[string]string{
		"meta.yml":                     "name: Test\nurl: https://example.com\nog_image_hosts:\n  - 127.0.0.1\n",
		"static/img/posts/trip/a.webp": string(fixture),
		"pages/photos/allowed.md":      "---\nlayout: photos\ntitle: Allowed\nimages:\n  - SRV/allowed/a.webp\n---\n",
		"pages/posts/external.md":      "---\nlayout: post\ntitle: External\ndate: 2024-01-04\n---\n\n![x](" + other + "/external/a.png)\n",
		"pages/posts/override.md":      "---\nlayout: post\ntitle: Override\ndate: 2024-01-05\nog_image: " + other + "/override/a.png\n---\n",
		"pages/posts/local.md":         "---\nlayout: post\ntitle: Local\ndate: 2024-01-06\n---\n\n<img src=\"/static/img/posts/trip/a.webp\">\n",
	})

	assertOGFile(t, root, "photos/allowed/og.jpg")
	assertOGFile(t, root, "posts/local/og.jpg")
	assertNoOGFile(t, root, "posts/external/og.jpg")
	assertNoOGFile(t, root, "posts/override/og.jpg")
	if srv.requested("/external/a.png") || srv.requested("/override/a.png") {
		t.Fatalf("disallowed hosts must not be fetched, got %v", srv.paths)
	}
	if strings.Contains(logs.String(), "level=warning") {
		t.Fatalf("disallowed hosts must be skipped quietly, got: %q", logs.String())
	}
}

func TestBuildPostUsesLocalStaticImage(t *testing.T) {
	srv := newImageServer(t)
	fixture, err := os.ReadFile(filepath.Join("..", "og", "testdata", "portrait.webp"))
	if err != nil {
		t.Fatal(err)
	}
	root := buildSite(t, srv, map[string]string{
		"static/img/posts/trip/a.webp": string(fixture),
		"pages/posts/local.md":         "---\nlayout: post\ntitle: Local\ndate: 2024-01-03\n---\n\n<img src=\"./../../static/img/posts/trip/a.webp\"/>\n",
	})

	assertOGFile(t, root, "posts/local/og.jpg")
	if len(srv.paths) != 0 {
		t.Fatalf("local image must not hit the network, got %v", srv.paths)
	}
}

func TestFirstImage(t *testing.T) {
	cases := []struct {
		name, md, want string
	}{
		{"none", "just text", ""},
		{"markdown", "a ![x](https://a.com/1.png) b", "https://a.com/1.png"},
		{"html", `<img alt="x" src="https://a.com/2.jpg">`, "https://a.com/2.jpg"},
		{"earliest wins", "![x](https://a.com/md.png)\n<img src=\"https://a.com/html.png\">", "https://a.com/md.png"},
		{"relative", `<img src="./../../static/a.png"/>`, "./../../static/a.png"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := firstImage(c.md); got != c.want {
				t.Fatalf("firstImage = %q, want %q", got, c.want)
			}
		})
	}
}

func TestResolveImage(t *testing.T) {
	cases := []struct {
		src, want string
	}{
		{"https://a.com/1.png", "https://a.com/1.png"},
		{"http://a.com/1.png", "http://a.com/1.png"},
		{"/static/img/a.png", "https://example.com/static/img/a.png"},
		{"./../../static/img/a.png", "https://example.com/static/img/a.png"},
		{"img/a.png", "https://example.com/posts/trip/img/a.png"},
		{"//cdn.com/a.png", "https://cdn.com/a.png"},
		{"data:image/png;base64,AAAA", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := resolveImage(c.src, "https://example.com/posts/trip/"); got != c.want {
			t.Fatalf("resolveImage(%q) = %q, want %q", c.src, got, c.want)
		}
	}
}
