package command

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	cnst "github.com/danishprakash/kizai/constants"
	"github.com/danishprakash/kizai/markdown"
	md "github.com/danishprakash/kizai/markdown"
	"github.com/danishprakash/kizai/utils"
	"gopkg.in/yaml.v2"

	"github.com/sirupsen/logrus"
)

type Blog struct {
	Files []string
	Dirs  []string
}

var (
	meta md.Meta
	allPosts []*md.Post
)

func initMeta() {
	data, err := os.ReadFile("meta.yml")
	if err != nil {
		log.Fatalf("Error reading file: %v", err)
	}

	err = yaml.Unmarshal(data, &meta)
	if err != nil {
		log.Fatalf("Error parsing YAML: %v", err)
	}
}

// processHome sets up the homepage enlisting all the posts
func processHome(dir string, posts []*md.Post) {
	// sort posts
	sort.Slice(posts, func(i, j int) bool {
		return posts[j].Date.Before(posts[i].Date)
	})

	page := md.Page{
		Meta:  meta,
		Posts: posts,
	}

	var htmlFilepath, mdFilepath string
	if dir == "posts" {
		htmlFilepath = filepath.Join(cnst.BUILD_DIR, "index.html")
		mdFilepath = filepath.Join(cnst.DIR, "index.md")
	} else if dir == "feed" {
		htmlFilepath = filepath.Join(cnst.BUILD_DIR, "feed.xml")
		mdFilepath = filepath.Join(cnst.DIR, dir, "feed.md")
	}

	if err := page.ParseMarkdown(mdFilepath); err != nil {
		logrus.Errorf("processDir: failed for file %s: %+v", mdFilepath, err)
	}

	page.Body = md.MarkdownToHTML(page.Markdown)
	err := page.RenderHTML(htmlFilepath)
	if err != nil {
		logrus.Errorf("processDir: %+v", err)
	}
}

// processDirs processes the various directories in source dir
func (b *Blog) processDirs() {
	for _, dir := range b.Dirs {
		if dir == "feed" {
			continue
		}
		var posts []*markdown.Post
		srcDir := filepath.Join(cnst.DIR, dir)

		// iterate over all items in the directory
		// (i.e. /posts/* or /reading/*)
		files, _ := os.ReadDir(srcDir)
		for _, file := range files {
			post, _ := processPage(file.Name(), dir)
			if post == nil {
				continue
			}

			// if we're processing /posts directory store
			// the posts so that we can use this information
			// to set up homepage, no need to independently
			// iterate over this directory again for the
			// purposes of fetching post titles and dates
			if dir == "posts" {
				posts = append(posts, post)
			}
			
			// collect all posts for tag processing
			if post != nil {
				allPosts = append(allPosts, post)
			}
		}

		// Once we've iterated through all the posts
		// use this information (namely title and date)
		// to also render the home page because we prefer
		// to show all the posts on the homepage instead
		// of a separate /blog page.
		if dir == "posts" {
			processHome("posts", posts)

			// While we're at it and have all the
			// post-related info, generate RSS feed
			processHome("feed", posts)
		}
	}
}

func processPage(file, dir string) (*markdown.Post, error) {
	// copy non-source files over to
	// build/ such as favicon
	if filepath.Ext(file) != ".md" {
		src := filepath.Join(cnst.DIR, file)
		dst := filepath.Join(cnst.BUILD_DIR, file)
		if err := utils.CopyFile(src, dst); err != nil {
			return nil, err
		}
		return nil, nil
	}

	srcDir := filepath.Join(cnst.DIR, dir)
	dstDir := filepath.Join(cnst.BUILD_DIR, dir)

	var htmlFile, htmlDir, mdFilepath, slug string
	mdFilepath = filepath.Join(srcDir, file)
	if file == "index.md" {
		// parse index pages for
		// directories within pages:
		//     build/books/index.html
		//     build/posts/index.html
		htmlDir = filepath.Join(dstDir, dir)
		htmlFile = filepath.Join(dstDir, "index.html")
		mdFilepath = filepath.Join(srcDir, "index.md")
	} else {
		// /pages/about.md => build/about/index.html
		// /pages/posts/makefiles-for-go.md => build/posts/makefiles-for-go/index.html
		slug = strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		htmlDir = filepath.Join(dstDir, slug)
		htmlFile = filepath.Join(htmlDir, "index.html")
	}
	os.MkdirAll(htmlDir, 0755)

	page := md.Page{
		Meta: meta,
	}
	if err := page.ParseMarkdown(mdFilepath); err != nil {
		logrus.Errorf("processDir: failed for file %s: %+v", htmlFile, err)
	}

	page.Body = md.MarkdownToHTML(page.Markdown)
	var url string
	if dir == "posts" {
		url = fmt.Sprintf("/posts/%s", slug)
	} else {
		url = fmt.Sprintf("/%s/%s", dir, slug)
	}
	
	page.Post = &md.Post{
		Slug:        slug,
		Frontmatter: page.Frontmatter,
		URL:         url,
		Body:        utils.XMLReadyString(page.Body),
	}
	if page.Frontmatter["date"] != nil {
		page.Post.Date, _ = time.Parse("2006-01-02", page.Frontmatter["date"].(string))
	}
	if page.Frontmatter["title"] != nil {
		page.Post.Title = utils.XMLReadyString(fmt.Sprintf("%v", page.Frontmatter["title"]))
	}
	if page.Frontmatter["tags"] != nil {
		if tagsList, ok := page.Frontmatter["tags"].([]interface{}); ok {
			for _, tag := range tagsList {
				if tagStr, ok := tag.(string); ok {
					page.Post.Tags = append(page.Post.Tags, tagStr)
				}
			}
		}
	}
	if page.Frontmatter["images"] != nil {
		if imagesList, ok := page.Frontmatter["images"].([]interface{}); ok {
			for _, img := range imagesList {
				if imgStr, ok := img.(string); ok {
					page.Post.Images = append(page.Post.Images, imgStr)
				}
			}
		}
	}

	err := page.RenderHTML(htmlFile)
	if err != nil {
		logrus.Errorf("processDir: %+v", err)
	}

	return page.Post, nil
}

func (p *Blog) processFiles() error {
	for _, file := range p.Files {
		processPage(file, "")
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
	if err := os.MkdirAll(buildStaticDir, 0755); err != nil {
		fmt.Printf("error creating %s directory: %+v", buildStaticDir, err)
		os.Exit(1)
	}
	if err := utils.CopyDir(cnst.STATIC, buildStaticDir); err != nil {
		fmt.Printf("error copying static directory: %+v", err)
		os.Exit(1)
	}
	files, err := os.ReadDir(cnst.DIR)
	if err != nil {
		return err
	}

	initMeta()
	
	// reset allPosts for each build
	allPosts = []*md.Post{}

	for _, f := range files {
		if f.IsDir() {
			b.Dirs = append(b.Dirs, f.Name())
		} else {
			b.Files = append(b.Files, f.Name())
		}
	}

	b.processFiles()
	b.processDirs()
	
	// process tags after all content is processed
	processTags()

	return nil
}

func clearIfDirExists(dir string) {
	err := os.RemoveAll(dir)
	if err != nil {
		panic(err)
	}
}

func processTags() {
	// collect unique tags and posts for each tag
	tagMap := make(map[string][]*md.Post)
	
	for _, post := range allPosts {
		for _, tag := range post.Tags {
			tagMap[tag] = append(tagMap[tag], post)
		}
	}
	
	// create /tags directory
	tagsDir := filepath.Join(cnst.BUILD_DIR, "tags")
	os.MkdirAll(tagsDir, 0755)
	
	// generate page for each tag
	for tag, posts := range tagMap {
		tagDir := filepath.Join(tagsDir, tag)
		os.MkdirAll(tagDir, 0755)
		
		// categorize posts by type
		var regularPosts, photos, books []*md.Post
		for _, post := range posts {
			if post.Frontmatter["layout"] == "photos" {
				photos = append(photos, post)
			} else if strings.HasPrefix(post.URL, "/reading/") {
				books = append(books, post)
			} else {
				regularPosts = append(regularPosts, post)
			}
		}
		
		// sort each category by date (newest first)
		sort.Slice(regularPosts, func(i, j int) bool {
			return regularPosts[j].Date.Before(regularPosts[i].Date)
		})
		sort.Slice(photos, func(i, j int) bool {
			return photos[j].Date.Before(photos[i].Date)
		})
		sort.Slice(books, func(i, j int) bool {
			return books[j].Date.Before(books[i].Date)
		})
		
		// generate markdown content for categorized posts
		var markdownContent strings.Builder
		markdownContent.WriteString(fmt.Sprintf("Found %d items tagged '%s'.\n\n", len(posts), tag))
		
		if len(regularPosts) > 0 {
			markdownContent.WriteString("## Posts\n\n")
			markdownContent.WriteString("| :--- |\n")
			for _, post := range regularPosts {
				markdownContent.WriteString(fmt.Sprintf("| [%s](%s) |\n", post.Title, post.URL))
			}
			markdownContent.WriteString("\n")
		}
		
		if len(photos) > 0 {
			markdownContent.WriteString("## Photos\n\n")
			markdownContent.WriteString("| :--- |\n")
			for _, post := range photos {
				markdownContent.WriteString(fmt.Sprintf("| [%s](%s) |\n", post.Title, post.URL))
			}
			markdownContent.WriteString("\n")
		}
		
		if len(books) > 0 {
			markdownContent.WriteString("## Books\n\n")
			markdownContent.WriteString("| :--- |\n")
			for _, post := range books {
				markdownContent.WriteString(fmt.Sprintf("| [%s](%s) |\n", post.Title, post.URL))
			}
		}
		
		page := md.Page{
			Meta:     meta,
			Markdown: markdownContent.String(),
			Frontmatter: map[string]interface{}{
				"layout": "tags",
				"title":  fmt.Sprintf("Posts tagged with '%s'", tag),
				"tag":    tag,
			},
		}
		
		page.Body = md.MarkdownToHTML(markdownContent.String())
		
		
		htmlFile := filepath.Join(tagDir, "index.html")
		err := page.RenderHTML(htmlFile)
		if err != nil {
			logrus.Errorf("processTags: %+v", err)
		}
	}
}

func setup() {
	clearIfDirExists(cnst.BUILD_DIR)
	os.Mkdir(cnst.BUILD_DIR, 0755)
}
