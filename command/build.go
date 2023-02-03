package command

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	md "github.com/danishprakash/kizai/markdown"
	"github.com/danishprakash/kizai/utils"

	"github.com/sirupsen/logrus"
)

const (
	BASE_DIR  = "/home/danishprakash/code/kizai-site"
	DIR       = BASE_DIR + "/pages"
	BUILD_DIR = BASE_DIR + "/build"
	STATIC    = BASE_DIR + "/static/css"
	TEMPLATES = BASE_DIR + "/templates"
)

func chdir() {
	_ = os.Chdir(DIR)
}

type Blog struct {
	Files []string
	Dirs  []string
	Posts []Post
}

type Post struct {
	Title       string
	Slug        string
	Date        string
	Frontmatter map[string]interface{}
	Body        []byte
}

// handle posts/
func (b *Blog) processDirs(dirCh chan<- bool) {
	for _, dir := range b.Dirs {
		// fmt.Println("dir: ", dir)
		srcDir := filepath.Join(DIR, dir)
		dstDir := filepath.Join(BUILD_DIR, dir)
		os.Mkdir(dstDir, 0755)

		// iterate over all the posts
		files, _ := os.ReadDir(dir)
		for _, file := range files {
			fn := file.Name()
			if filepath.Ext(fn) != ".md" {
				continue
			}

			mdFilepath := filepath.Join(srcDir, file.Name())

			// https://danishpraka.sh/posts/slug/
			slug := strings.TrimSuffix(filepath.Base(file.Name()), filepath.Ext(file.Name()))
			os.MkdirAll(filepath.Join(dstDir, slug), 0755)
			htmlFile := filepath.Join(dstDir, slug, "index.html")

			page := md.Page{}
			if err := page.ParseFrontmatter(mdFilepath); err != nil {
				logrus.Errorf("processDir: %+v", err)
				continue
			}

			err := page.RenderHTML(md.MarkdownToHTML(mdFilepath), htmlFile)
			if err != nil {
				logrus.Errorf("processDir: %+v", err)
			}

			// TODO: sort posts by date
			// Parse frontmatter from the post (title, date)
			// fmt.Println(file.Name(), fm["title"])
			var title string
			if page.FM["title"] != nil {
				title = fmt.Sprintf("%v", page.FM["title"])
			}
			b.Posts = append(b.Posts, Post{
				Slug:        slug,
				Title:       title,
				Frontmatter: page.FM,
				Body:        md.MarkdownToHTML(mdFilepath),
			})
		}

		// I prefer showing posts on the homepage
		// so this won't work or there would be redundancy
		// to retain posts on home, have to handle it separately
		if filepath.Base(srcDir) == "posts" {
			continue
		}

		// parse index pages for
		// directories within pages:
		//     build/books/index.html
		//     build/posts/index.html
		indexHTML := filepath.Join(dstDir, "index.html")
		indexMDFilepath := filepath.Join(srcDir, "index.md")

		page := md.Page{}
		if err := page.ParseFrontmatter(indexMDFilepath); err != nil {
			logrus.Errorf("processDir: failed for file %s: %+v", srcDir, err)
			continue
		}

		err := page.RenderHTML(md.MarkdownToHTML(indexMDFilepath), indexHTML)
		if err != nil {
			logrus.Errorf("processDir: %+v", err)
		}

	}
}

func (p *Blog) processFiles(dirCh <-chan bool) error {
	for _, file := range p.Files {
		// TODO: rm this
		if filepath.Ext(file) != ".md" || strings.Contains(filepath.Base(file), "readme") {
			continue
		}

		var htmlFile string
		if file == "index.md" {
			// pages/index.md => build/index.html (root)
			htmlFile = filepath.Join(BUILD_DIR, "index.html")
		} else {
			// pages/about.md => build/about/index.html
			htmlDir := filepath.Join(BUILD_DIR, strings.TrimSuffix(file, ".md"))
			os.Mkdir(htmlDir, 0755)
			htmlFile = filepath.Join(htmlDir, "index.html")
		}

		mdFilepath := filepath.Join(DIR, file)
		htmlBody := md.MarkdownToHTML(mdFilepath)

		page := md.Page{}
		if err := page.ParseFrontmatter(mdFilepath); err != nil {
			logrus.Errorf("processDir: failed for file %s: %+v", htmlFile, err)
		}

		err := page.RenderHTML(htmlBody, htmlFile)
		if err != nil {
			logrus.Errorf("processDir: %+v", err)
		}

	}
	return nil
}

func Build() {
	logrus.SetReportCaller(true)
	chdir()
	setup()
	blog := &Blog{}
	if err := blog.process(); err != nil {
		panic(err)
	}
}

func (b *Blog) process() error {
	buildStaticDir := filepath.Join(BUILD_DIR, "static")
	os.Mkdir(buildStaticDir, 0755)
	if err := utils.CopyDir(STATIC, buildStaticDir); err != nil {
		fmt.Printf("error copying static directory: %+v", err)
		os.Exit(1)
	}
	files, err := os.ReadDir(".")
	if err != nil {
		return err
	}

	for _, f := range files {
		fmt.Printf("process: %s\n", f.Name())
		if f.IsDir() {
			b.Dirs = append(b.Dirs, f.Name())
		} else {
			b.Files = append(b.Files, f.Name())
		}
	}

	dirCh := make(chan bool, 1)

	b.processDirs(dirCh)
	b.processFiles(dirCh)

	close(dirCh)

	return nil
}

func clearIfDirExists(dir string) {
	err := os.RemoveAll(dir)
	if err != nil {
		panic(err)
	}
}

func setup() {
	clearIfDirExists(BUILD_DIR)
	os.Mkdir(BUILD_DIR, 0755)
}
