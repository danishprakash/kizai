package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
)

type Page struct {
	Files []string
	Dirs  []string
}

const DIR = "/home/danish/work/interviewstreet/programming/mine/site/content"
const BUILD_DIR = "/home/danish/work/interviewstreet/programming/mine/site/build"

func chdir() {
	_ = os.Chdir(DIR)
	currDir, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(currDir)
}

func (p *Page) processDirs() error {
	for _, dir := range p.Dirs {
		fmt.Println("dir: ", dir)
		srcDir := filepath.Join(DIR, dir)
		dstDir := filepath.Join(BUILD_DIR, dir)
		os.Mkdir(dstDir, 0755)

		files, _ := os.ReadDir(dir)
		for _, file := range files {
			fn := file.Name()
			if filepath.Ext(fn) != ".md" {
				continue
			}

			fmt.Println(dstDir, file.Name())
			md, err := ioutil.ReadFile(filepath.Join(srcDir, file.Name()))
			if err != nil {
				panic("failed to read file")
			}
			html := mdToHTML(md)

			slug := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
			htmlFile := filepath.Join(dstDir, fmt.Sprintf("%s.html", slug))
			fmt.Println("slug: ", htmlFile)
			// os.Mkdir(filepath.Join(dstDir, slug), 0755)
			f, err := os.Create(htmlFile)
			if err != nil {
				fmt.Println("failed to create file", err)
			}

			_, err = f.Write(html)
			if err != nil {
				panic("failed to write file")
			}
		}
	}
	return nil
}

func mdToHTML(md []byte) []byte {
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)
	return markdown.ToHTML(md, nil, renderer)
}

func (p *Page) processFiles() error {
	for _, file := range p.Files {
		if filepath.Ext(file) != ".md" {
			continue
		}

		htmlDir := filepath.Join(BUILD_DIR, strings.TrimSuffix(file, ".md"))
		os.Mkdir(htmlDir, 0755)
		htmlFile := filepath.Join(htmlDir, "index.html")

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
		_, err = f.Write(html)
		if err != nil {
			panic("failed to write file")
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

	p.processDirs()
	p.processFiles()

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
}
