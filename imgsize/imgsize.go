// ABOUTME: Looks up the pixel dimensions of remote images by reading only
// ABOUTME: their headers, with an on-disk cache across builds.
package imgsize

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/sirupsen/logrus"
	_ "golang.org/x/image/webp"
)

const (
	// some hosts (e.g. imgur) reject Go's default User-Agent
	userAgent = "kizai"
	// image headers live at the start of the file, don't fetch the rest
	maxHeaderBytes = 64 << 10
	concurrency    = 8
)

type Size struct {
	Width  int
	Height int
}

// Prober fetches image headers and caches the decoded dimensions in
// CacheDir so that rebuilds don't hit the network again.
type Prober struct {
	Client   *http.Client
	CacheDir string
}

func NewProber() *Prober {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return &Prober{
		Client:   &http.Client{Timeout: 15 * time.Second},
		CacheDir: filepath.Join(dir, "kizai", "imgsize"),
	}
}

func (p *Prober) cacheFile(url string) string {
	sum := sha256.Sum256([]byte(url))
	return filepath.Join(p.CacheDir, hex.EncodeToString(sum[:]))
}

// Get returns the dimensions of the image at url.
func (p *Prober) Get(url string) (Size, error) {
	var s Size
	cached := p.cacheFile(url)
	if data, err := os.ReadFile(cached); err == nil {
		if _, err := fmt.Sscanf(string(data), "%d %d", &s.Width, &s.Height); err == nil {
			return s, nil
		}
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return s, fmt.Errorf("fetch %s: %w", url, err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Range", fmt.Sprintf("bytes=0-%d", maxHeaderBytes-1))
	resp, err := p.Client.Do(req)
	if err != nil {
		return s, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return s, fmt.Errorf("fetch %s: %s", url, resp.Status)
	}

	cfg, _, err := image.DecodeConfig(io.LimitReader(resp.Body, maxHeaderBytes))
	if err != nil {
		return s, fmt.Errorf("decode %s: %w", url, err)
	}
	s = Size{Width: cfg.Width, Height: cfg.Height}

	if err := os.MkdirAll(p.CacheDir, 0755); err != nil {
		logrus.Warnf("imgsize: cache dir: %v", err)
	} else if err := os.WriteFile(cached, []byte(fmt.Sprintf("%d %d", s.Width, s.Height)), 0644); err != nil {
		logrus.Warnf("imgsize: cache write: %v", err)
	}
	return s, nil
}

// GetAll looks up the dimensions of urls concurrently. Failures are
// logged and yield a zero Size so that the build carries on.
func (p *Prober) GetAll(urls []string) []Size {
	sizes := make([]Size, len(urls))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	for i, url := range urls {
		wg.Add(1)
		go func(i int, url string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			s, err := p.Get(url)
			if err != nil {
				logrus.Warnf("imgsize: %v", err)
				return
			}
			sizes[i] = s
		}(i, url)
	}
	wg.Wait()
	return sizes
}
