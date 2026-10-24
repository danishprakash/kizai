// ABOUTME: Tests for the build command helpers.
// ABOUTME: Covers deriving photo album image counts from album pages.
package command

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	md "github.com/danishprakash/kizai/markdown"
	"github.com/sirupsen/logrus"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func parseIndex(t *testing.T, dir string) []interface{} {
	t.Helper()
	page := md.Page{}
	if err := page.ParseYAML(filepath.Join(dir, "index.yaml")); err != nil {
		t.Fatal(err)
	}
	setPhotoCounts(page.Yaml, dir)
	photos, ok := page.Yaml["photos"].([]interface{})
	if !ok {
		t.Fatalf("photos is not a list: %T", page.Yaml["photos"])
	}
	return photos
}

func TestSetPhotoCounts(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "index.yaml"), `---
layout: photography
photos:
  - title: Kyoto, Japan
    slug: kyoto-japan-2024
  - title: Paris, France
    slug: paris-france-2025
    count: 99
---
`)
	writeFile(t, filepath.Join(dir, "kyoto-japan-2024.md"), `---
layout: photos
images:
  - https://example.com/a.webp
  - https://example.com/b.webp
  - https://example.com/c.webp
---
`)
	writeFile(t, filepath.Join(dir, "paris-france-2025.md"), `---
layout: photos
images:
  - https://example.com/a.webp
---
`)

	photos := parseIndex(t, dir)
	want := []int{3, 1}
	for i, p := range photos {
		entry := p.(map[interface{}]interface{})
		if entry["count"] != want[i] {
			t.Errorf("%v: count = %v, want %d", entry["slug"], entry["count"], want[i])
		}
	}
}

func TestSetPhotoCountsMissingAlbum(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "index.yaml"), `---
layout: photography
photos:
  - title: Nowhere
    slug: does-not-exist
---
`)

	var logs bytes.Buffer
	logrus.SetOutput(&logs)
	defer logrus.SetOutput(os.Stderr)

	photos := parseIndex(t, dir)
	if !strings.Contains(logs.String(), "does-not-exist.md: no such file or directory") {
		t.Errorf("missing error log, got: %q", logs.String())
	}
	entry := photos[0].(map[interface{}]interface{})
	if _, ok := entry["count"]; ok {
		t.Errorf("count = %v, want no count", entry["count"])
	}
}
