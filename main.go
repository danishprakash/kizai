package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/danishprakash/kizai/markdown"
	"github.com/gernest/front"
)

type Page struct {
	Files []string
	Dirs  []string
	Posts []Post
}

const (
	BASE_DIR  = "/home/danishprakash/code/kizai-site"
	DIR       = BASE_DIR + "/pages"
	BUILD_DIR = BASE_DIR + "/build"
	STATIC    = BASE_DIR + "/static/css"
	TEMPLATES = BASE_DIR + "/templates"
)

func chdir() {
	_ = os.Chdir(DIR)
	// currDir, err := os.Getwd()
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	// fmt.Println(currDir)
}

type Post struct {
	Title       string
	Slug        string
	Date        string
	Frontmatter map[string]interface{}
	Body        []byte
}

// handle posts/
func (p *Page) processDirs(dirCh chan<- bool) {
	for _, dir := range p.Dirs {
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

			// fmt.Println(dstDir, file.Name())

			// parse frontmatter
			m := front.NewMatter()
			m.Handle("---", front.YAMLHandler)
			file, err := os.Open(filepath.Join(srcDir, file.Name()))
			fm, md, err := m.Parse(file)
			if err != nil {
				panic("failed to parse file")
			}

			// fmt.Println("fm: ", fm)
			// fmt.Println("body: ", md)

			// absolute filepath for current file
			// md, err := ioutil.ReadFile(filepath.Join(srcDir, file.Name()))
			// if err != nil {
			// 	panic("failed to read file")
			// }

			// TODO: wrap this method to also
			// include template execution
			html := markdown.MDToHTML([]byte(md))

			// https://danishpraka.sh/posts/slug/
			slug := strings.TrimSuffix(filepath.Base(file.Name()), filepath.Ext(file.Name()))
			// fmt.Println("slug= ", slug)
			os.MkdirAll(filepath.Join(dstDir, slug), 0755)
			htmlFile := filepath.Join(dstDir, slug, "index.html")

			// fmt.Println("slug: ", html)
			f, err := os.Create(htmlFile)
			if err != nil {
				fmt.Println("failed to create file", err)
			}

			_, err = f.Write(html)
			if err != nil {
				fmt.Println("err writing to file", err)
				panic("failed to write file")
			}

			// TODO: sort posts by date
			// Parse frontmatter from the post (title, date)
			// fmt.Println(file.Name(), fm["title"])
			var title string
			if fm["title"] != nil {
				title = fmt.Sprintf("%v", fm["title"])
			}
			p.Posts = append(p.Posts, Post{
				Slug:        slug,
				Title:       title,
				Frontmatter: fm,
				Body:        html,
			})
		}
	}
	dirCh <- true
	fmt.Println("***DONE***")
}

func (p *Page) processFiles(dirCh <-chan bool) error {
	for _, file := range p.Files {
		if filepath.Ext(file) != ".md" || strings.Contains(filepath.Base(file), "readme") {
			continue
		}
		fmt.Println("file: ", file)

		var htmlFile string
		// pages/index.md => build/index.html (root)
		if file == "index.md" {
			htmlFile = filepath.Join(BUILD_DIR, "index.html")
		} else {
			// pages/about.md => build/about/index.html
			htmlDir := filepath.Join(BUILD_DIR, strings.TrimSuffix(file, ".md"))
			os.Mkdir(htmlDir, 0755)
			htmlFile = filepath.Join(htmlDir, "index.html")
		}

		// parse frontmatter and body from md file
		m := front.NewMatter()
		m.Handle("---", front.YAMLHandler)
		fl, err := os.Open(file)
		fm, md, err := m.Parse(fl)
		if err != nil {
			fmt.Println("err: ", file, err)
			panic("failed to parse file")
		}

		f, err := os.Create(htmlFile)
		if err != nil {
			fmt.Println("failed to create file", err)
		}

		htmlBody := markdown.MDToHTML([]byte(md))
		markdown.RenderHTML(htmlBody, fm, f)
	}
	return nil
}

func (p *Page) process() error {
	chdir()
	files, err := os.ReadDir(".")
	if err != nil {
		return err
	}

	for _, f := range files {
		fmt.Printf("process: %s\n", f.Name())
		if f.IsDir() {
			p.Dirs = append(p.Dirs, f.Name())
		} else {
			p.Files = append(p.Files, f.Name())
		}
	}

	dirCh := make(chan bool, 1)

	p.processDirs(dirCh)
	p.processFiles(dirCh)

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

func main() {
	setup()
	page := &Page{}
	if err := page.process(); err != nil {
		panic(err)
	}

	buildStaticDir := filepath.Join(BUILD_DIR, "static")
	os.Mkdir(buildStaticDir, 0755)
	if err := CopyDir(STATIC, buildStaticDir); err != nil {
		fmt.Printf("error copying static directory: %+v", err)
		os.Exit(1)
	}
}
