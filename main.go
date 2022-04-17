package main

import (
	"fmt"
	"os"
)

type Page struct {
	Files []string
	Dirs  []string
}

const DIR = "/home/danish/work/interviewstreet/programming/mine/danishprakash.github.io"

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
		files, _ := os.ReadDir(dir)
		for _, file := range files {
			fmt.Println("dir: file: ", file)
		}
	}
	return nil
}

func (p *Page) processFiles() error {
	for _, file := range p.Files {
		fmt.Println("file: ", file)
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

func main() {
	page := &Page{}
	page.process()
}
