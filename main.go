package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
)

type Page struct {
	Files []string
	Dirs  []string
	Posts []Post
}

const DIR = "/home/danish/work/interviewstreet/programming/mine/site/content"
const BUILD_DIR = "/home/danish/work/interviewstreet/programming/mine/site/build"
const STATIC = "/home/danish/work/interviewstreet/programming/mine/site/assets/css"
const TEMPLATES = "/home/danish/work/interviewstreet/programming/mine/site/templates"

func chdir() {
	_ = os.Chdir(DIR)
	currDir, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(currDir)
}

type Post struct {
	Title string
	Date  string
}

// handle posts/
func (p *Page) processDirs(dirCh chan<- bool) {
	for _, dir := range p.Dirs {
		fmt.Println("dir: ", dir)
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

			fmt.Println(dstDir, file.Name())

			// absolute filepath for current file
			md, err := ioutil.ReadFile(filepath.Join(srcDir, file.Name()))
			if err != nil {
				panic("failed to read file")
			}

			// TODO: wrap this method to also
			// include template execution
			html := mdToHTML(md)

			// https://danishpraka.sh/posts/slug/
			slug := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
			os.Mkdir(filepath.Join(dstDir, slug), 0755)
			htmlFile := filepath.Join(dstDir, slug, "index.html")

			fmt.Println("slug: ", htmlFile)
			f, err := os.Create(htmlFile)
			if err != nil {
				fmt.Println("failed to create file", err)
			}

			_, err = f.Write(html)
			if err != nil {
				panic("failed to write file")
			}

			// Parse frontmatter from the post (title, date)
			p.Posts = append(p.Posts, Post{
				Title: slug,
			})
		}
	}
	dirCh <- true
}

func mdToHTML(md []byte) []byte {
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)
	return markdown.ToHTML(md, nil, renderer)
}

func (p *Page) processFiles(dirCh <-chan bool) error {
	for _, file := range p.Files {
		if filepath.Ext(file) != ".md" {
			continue
		}

		var htmlFile string
		if file == "_index.md" {
			htmlFile = filepath.Join(BUILD_DIR, "index.html")
		} else {
			htmlDir := filepath.Join(BUILD_DIR, strings.TrimSuffix(file, ".md"))
			os.Mkdir(htmlDir, 0755)
			htmlFile = filepath.Join(htmlDir, "index.html")
		}

		md, err := ioutil.ReadFile(file)
		if err != nil {
			panic("failed to read file")
		}

		// fmt.Printf("html for file: %s\n%s\n======\n", file, string(html))
		f, err := os.Create(htmlFile)
		if err != nil {
			fmt.Println("failed to create file", err)
		}

		html := mdToHTML(md)
		if file != "_index.md" {
			fmt.Println("skipping file: ", file)
			_, err = f.Write(html)
			if err != nil {
				panic("failed to write file")
			}
			continue
		}

		// render posts

		// wait for posts to be rendered
		select {
		case <-dirCh:
			break
		}

		// Load all templates
		// Execute with frontmatter and body(list of posts)
		fmt.Println("html: ", string(html))

		fmt.Println(f.Name())
		// execute template
		templPath := filepath.Join(TEMPLATES, "index.html")

		var b []byte
		if b, err = os.ReadFile(templPath); err != nil {
			return err
		}

		tmpl := template.Must(template.New("").Parse(string(b)))
		if err = tmpl.Execute(f, string(html)); err != nil {
			fmt.Println("failed to execute template ", err)
			return err
		}

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
		if f.IsDir() {
			p.Dirs = append(p.Dirs, f.Name())
		} else {
			p.Files = append(p.Files, f.Name())
		}
	}

	dirCh := make(chan bool, 1)

	p.processDirs(dirCh)
	p.processFiles(dirCh)

	return nil
}

func setup() {
	os.Mkdir(BUILD_DIR, 0755)
}

func main() {
	setup()
	page := &Page{}
	page.process()
	buildStaticDir := filepath.Join(BUILD_DIR, "static")
	os.Mkdir(buildStaticDir, 0755)
	if err := CopyDir(STATIC, buildStaticDir); err != nil {
		fmt.Printf("error copying static directory: %+v", err)
		os.Exit(1)
	}
}
