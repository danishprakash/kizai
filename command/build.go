package command

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	cnst "github.com/danishprakash/kizai/constants"
	md "github.com/danishprakash/kizai/markdown"
	"github.com/danishprakash/kizai/utils"

	"github.com/sirupsen/logrus"
)

func chdir() {
	_ = os.Chdir(cnst.DIR)
}

type Blog struct {
	Files []string
	Dirs  []string
}

type Post struct {
	Title       string
	Slug        string
	Date        time.Time
	Frontmatter map[string]interface{}
	Body        []byte
	URL         string
}

// handle posts/
func (b *Blog) processDirs() {
	for _, dir := range b.Dirs {
		srcDir := filepath.Join(cnst.DIR, dir)
		dstDir := filepath.Join(cnst.BUILD_DIR, dir)
		os.Mkdir(dstDir, 0755)

		var posts []Post

		// iterate over all the posts
		files, _ := os.ReadDir(srcDir)
		for _, file := range files {
			fn := file.Name()
			if filepath.Ext(fn) != ".md" {
				continue
			}

			mdFilepath := filepath.Join(srcDir, file.Name())

			// https://danishpraka.sh/posts/slug
			// https://danishpraka.sh/2019-12-7-using-makefiles-for-go
			// 		=> https://danishpraka.sh/using-makefiles-for-go
			//
			//^[0-9]*-[0-9]*-[0-9]*-(.*)
			slug := strings.TrimSuffix(filepath.Base(file.Name()), filepath.Ext(file.Name()))
			var re = regexp.MustCompile(`^[0-9]*-[0-9]*-[0-9]*-`)
			slug = re.ReplaceAllString(slug, "")
			os.MkdirAll(filepath.Join(dstDir, slug), 0755)
			htmlFile := filepath.Join(dstDir, slug, "index.html")

			page := md.Page{}
			if err := page.ParseFrontmatter(mdFilepath); err != nil {
				logrus.Errorf("processDir: %+v", err)
				continue
			}

			// parse date if present
			var date time.Time
			if page.FM["date"] != nil {
				date, _ = time.Parse("2006-01-02", page.FM["date"].(string))
			}

			htmlBody := md.MarkdownToHTML(page.MDBody)
			data := struct {
				FM   map[string]interface{}
				Body string
				Date time.Time
			}{page.FM, string(htmlBody), date}
			err := page.RenderHTML(htmlFile, data)
			if err != nil {
				logrus.Errorf("processDir: %+v", err)
			}

			// we only need to populate Posts if
			// we're dealing with posts, for all other
			// directories (books for now), we're done
			if filepath.Base(srcDir) != "posts" {
				continue
			}

			// TODO: sort posts by date
			// Parse frontmatter from the post (title, date)
			var title string
			if page.FM["title"] != nil {
				title = fmt.Sprintf("%v", page.FM["title"])
			}
			posts = append(posts, Post{
				Slug:        slug,
				Title:       title,
				Frontmatter: page.FM,
				Date:        date,
				Body:        md.MarkdownToHTML(page.MDBody),
				URL:         fmt.Sprintf("/posts/%s", slug),
			})
		}

		// parse index pages for
		// directories within pages:
		//     build/books/index.html
		//     build/posts/index.html
		indexHTML := filepath.Join(dstDir, "index.html")
		indexMDFilepath := filepath.Join(srcDir, "index.md")

		// I prefer showing posts on the homepage
		// so this won't work or there would be redundancy
		// to retain posts on home, have to handle it separately
		//    build/posts/index.md => build/index.md
		if filepath.Base(srcDir) == "posts" {
			indexHTML = filepath.Join(filepath.Clean(filepath.Join(dstDir, "..")), "index.html")
			indexMDFilepath = filepath.Join(filepath.Clean(filepath.Join(srcDir, "..")), "index.md")
		}

		page := md.Page{}
		if err := page.ParseFrontmatter(indexMDFilepath); err != nil {
			logrus.Errorf("processDir: failed for file %s: %+v", srcDir, err)
			continue
		}

		// sort posts
		sort.Slice(posts, func(i, j int) bool {
			return posts[j].Date.Before(posts[i].Date)
		})

		htmlBody := md.MarkdownToHTML(page.MDBody)
		data := struct {
			FM    map[string]interface{}
			Body  string
			Posts []Post
		}{page.FM, string(htmlBody), posts}
		err := page.RenderHTML(indexHTML, data)
		if err != nil {
			logrus.Errorf("processDir: %+v", err)
		}

	}
}

func (p *Blog) processFiles() error {
	for _, file := range p.Files {
		// TODO: rm this
		if filepath.Ext(file) != ".md" {
			continue
		}

		var htmlFile string
		if file == "index.md" {
			// this is handled in processDirs
			// reasoning given there as well
			continue
		} else {
			// pages/about.md => build/about/index.html
			htmlDir := filepath.Join(cnst.BUILD_DIR, strings.TrimSuffix(file, ".md"))
			os.Mkdir(htmlDir, 0755)
			htmlFile = filepath.Join(htmlDir, "index.html")
		}

		mdFilepath := filepath.Join(cnst.DIR, file)
		page := md.Page{}
		if err := page.ParseFrontmatter(mdFilepath); err != nil {
			logrus.Errorf("processDir: failed for file %s: %+v", htmlFile, err)
		}

		htmlBody := md.MarkdownToHTML(page.MDBody)

		data := struct {
			FM   map[string]interface{}
			Body string
		}{page.FM, string(htmlBody)}
		err := page.RenderHTML(htmlFile, data)
		if err != nil {
			logrus.Errorf("processDir: %+v", err)
		}

	}
	return nil
}

func Build() {
	logrus.SetReportCaller(true)
	setup()
	blog := &Blog{}
	if err := blog.process(); err != nil {
		panic(err)
	}
}

func (b *Blog) process() error {
	buildStaticDir := filepath.Join(cnst.BUILD_DIR, "static")
	os.Mkdir(buildStaticDir, 0755)
	if err := utils.CopyDir(cnst.STATIC, buildStaticDir); err != nil {
		fmt.Printf("error copying static directory: %+v", err)
		os.Exit(1)
	}
	files, err := os.ReadDir(cnst.DIR)
	if err != nil {
		return err
	}

	for _, f := range files {
		if f.IsDir() {
			b.Dirs = append(b.Dirs, f.Name())
		} else {
			b.Files = append(b.Files, f.Name())
		}
	}

	b.processDirs()
	b.processFiles()

	return nil
}

func clearIfDirExists(dir string) {
	err := os.RemoveAll(dir)
	if err != nil {
		panic(err)
	}
}

func setup() {
	clearIfDirExists(cnst.BUILD_DIR)
	os.Mkdir(cnst.BUILD_DIR, 0755)
}
