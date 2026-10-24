// ABOUTME: Build-time Open Graph support: picks the source image for a
// ABOUTME: page, renders it to og.jpg next to the page and computes URLs.
package command

import (
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/adrg/frontmatter"
	cnst "github.com/danishprakash/kizai/constants"
	"github.com/danishprakash/kizai/og"
	"github.com/sirupsen/logrus"
)

const ogFilename = "og.jpg"

var (
	mdImageRe   = regexp.MustCompile(`!\[[^\]]*\]\(\s*<?([^)\s>]+)`)
	htmlImageRe = regexp.MustCompile(`<img[^>]*\ssrc=["']([^"']+)["']`)
)

// loadPhotoCovers reads the photos index and maps album slug to cover.
func loadPhotoCovers(path string) map[string]string {
	covers := map[string]string{}
	f, err := os.Open(path)
	if err != nil {
		return covers
	}
	defer f.Close()

	var index struct {
		Photos []struct {
			Slug  string `yaml:"slug"`
			Cover string `yaml:"cover"`
		} `yaml:"photos"`
	}
	if _, err := frontmatter.Parse(f, &index); err != nil {
		logrus.Warnf("og: failed to parse %s: %v", path, err)
		return covers
	}
	for _, p := range index.Photos {
		if p.Slug != "" && p.Cover != "" {
			covers[p.Slug] = p.Cover
		}
	}
	return covers
}

// ogSource picks the image to use for a page's OG preview:
//   - og_image from frontmatter, if set
//   - photo albums: cover from the photos index, else the first image
//   - everything else: the first image in the markdown body
func ogSource(fm map[string]interface{}, images []string, slug, markdown string) string {
	if s, ok := fm["og_image"].(string); ok && s != "" {
		return s
	}
	if fm["layout"] == "photos" {
		if cover := photoCovers[slug]; cover != "" {
			return cover
		}
		if len(images) > 0 {
			return images[0]
		}
		return ""
	}
	return firstImage(markdown)
}

// firstImage returns the src of the earliest markdown or <img> image
// in md, as written.
func firstImage(md string) string {
	src, pos := "", -1
	for _, re := range []*regexp.Regexp{mdImageRe, htmlImageRe} {
		if m := re.FindStringSubmatchIndex(md); m != nil && (pos == -1 || m[0] < pos) {
			src, pos = md[m[2]:m[3]], m[0]
		}
	}
	return src
}

// resolveImage resolves src against the page URL and returns it as an
// absolute http(s) URL, or "" if src isn't fetchable (e.g. data: URIs).
func resolveImage(src, page string) string {
	if src == "" {
		return ""
	}
	base, err := url.Parse(page)
	if err != nil {
		return ""
	}
	ref, err := url.Parse(src)
	if err != nil {
		return ""
	}
	abs := base.ResolveReference(ref)
	if abs.Scheme != "http" && abs.Scheme != "https" {
		return ""
	}
	return abs.String()
}

// localStaticFile maps an absolute URL under this site's /static/ to
// the source file in the static dir, or "" for anything else.
func localStaticFile(abs string) string {
	u, err := url.Parse(abs)
	if err != nil {
		return ""
	}
	site, err := url.Parse(meta.URL)
	if err != nil || u.Host != site.Host || !strings.HasPrefix(u.Path, "/static/") {
		return ""
	}
	return filepath.Join(cnst.BASE_DIR, filepath.FromSlash(u.Path))
}

// allowedHost reports whether OG images may be fetched from abs, as
// per og_image_hosts in meta.yml. All hosts are allowed if it's unset.
func allowedHost(abs string) bool {
	if len(meta.OGImageHosts) == 0 {
		return true
	}
	u, err := url.Parse(abs)
	if err != nil {
		return false
	}
	for _, h := range meta.OGImageHosts {
		if u.Hostname() == h {
			return true
		}
	}
	return false
}

// generateOGImage renders src into dir/og.jpg and returns its absolute
// URL. Images from this site's static dir are read from disk, others
// are fetched if their host is allowed. Failures are logged and yield
// "" so that the build carries on.
func generateOGImage(src, dir string) string {
	abs := resolveImage(src, pageURL(dir))
	if abs == "" {
		return ""
	}

	var data []byte
	var err error
	if local := localStaticFile(abs); local != "" {
		if data, err = os.ReadFile(local); err == nil {
			data, err = og.Render(data)
		}
	} else if allowedHost(abs) {
		data, err = ogGen.Generate(abs)
	} else {
		logrus.Debugf("og: skipping %s: host not in og_image_hosts: %s", dir, abs)
		return ""
	}
	if err != nil {
		logrus.Warnf("og: skipping %s: %v", dir, err)
		return ""
	}

	if err := os.WriteFile(filepath.Join(dir, ogFilename), data, 0644); err != nil {
		logrus.Warnf("og: skipping %s: %v", dir, err)
		return ""
	}
	return pageURL(dir) + ogFilename
}

// canonicalURL returns the absolute URL a built html file is served at.
func canonicalURL(htmlFile string) string {
	return pageURL(filepath.Dir(htmlFile))
}

// pageURL maps a directory under the build dir to its absolute URL,
// always with a trailing slash: build/photos/x => <url>/photos/x/
func pageURL(dir string) string {
	rel, err := filepath.Rel(cnst.BUILD_DIR, dir)
	if err != nil || rel == "." {
		return meta.URL + "/"
	}
	return meta.URL + "/" + filepath.ToSlash(rel) + "/"
}
