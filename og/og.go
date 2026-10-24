// ABOUTME: Renders Open Graph preview images (1200x630 JPEG, size-capped)
// ABOUTME: from remote source images, with an on-disk cache across builds.
package og

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	_ "image/gif"
	_ "image/png"

	"github.com/sirupsen/logrus"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const (
	Width  = 1200
	Height = 630
	// WhatsApp drops previews above roughly 300KB, keep a margin.
	MaxBytes = 250 * 1024
	// some hosts (e.g. imgur) reject Go's default User-Agent
	UserAgent = "kizai"

	maxSourceBytes = 50 << 20
	// bump to invalidate cached renders when rendering changes
	cacheVersion = "v1"
)

// Render decodes an image (jpeg, png, gif, webp), center-crops it to
// the OG aspect ratio, scales it to Width x Height and encodes it as
// JPEG, lowering quality until the output fits in MaxBytes.
func Render(data []byte) ([]byte, error) {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	dst := image.NewRGBA(image.Rect(0, 0, Width, Height))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, centerCrop(src.Bounds()), draw.Src, nil)

	for q := 85; q >= 25; q -= 10 {
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: q}); err != nil {
			return nil, fmt.Errorf("encode: %w", err)
		}
		if buf.Len() <= MaxBytes {
			return buf.Bytes(), nil
		}
	}
	return nil, fmt.Errorf("cannot encode under %d bytes", MaxBytes)
}

// centerCrop returns the largest Width:Height rectangle centered in b.
func centerCrop(b image.Rectangle) image.Rectangle {
	bw, bh := b.Dx(), b.Dy()
	if bw*Height > bh*Width {
		cw := bh * Width / Height
		x := b.Min.X + (bw-cw)/2
		return image.Rect(x, b.Min.Y, x+cw, b.Max.Y)
	}
	ch := bw * Height / Width
	y := b.Min.Y + (bh-ch)/2
	return image.Rect(b.Min.X, y, b.Max.X, y+ch)
}

// Generator fetches remote images and renders them, caching rendered
// output in CacheDir so that rebuilds don't hit the network again.
type Generator struct {
	Client   *http.Client
	CacheDir string
}

func NewGenerator() *Generator {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return &Generator{
		Client:   &http.Client{Timeout: 15 * time.Second},
		CacheDir: filepath.Join(dir, "kizai", "og"),
	}
}

func (g *Generator) Generate(url string) ([]byte, error) {
	sum := sha256.Sum256([]byte(cacheVersion + url))
	cached := filepath.Join(g.CacheDir, hex.EncodeToString(sum[:])+".jpg")
	if data, err := os.ReadFile(cached); err == nil {
		return data, nil
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	req.Header.Set("User-Agent", UserAgent)
	resp, err := g.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: %s", url, resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxSourceBytes))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", url, err)
	}

	out, err := Render(data)
	if err != nil {
		return nil, fmt.Errorf("render %s: %w", url, err)
	}

	if err := os.MkdirAll(g.CacheDir, 0755); err != nil {
		logrus.Warnf("og: cache dir: %v", err)
	} else if err := os.WriteFile(cached, out, 0644); err != nil {
		logrus.Warnf("og: cache write: %v", err)
	}
	return out, nil
}
