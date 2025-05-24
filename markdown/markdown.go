package markdown

import (
	"fmt"
	"os"
	"time"

	"github.com/adrg/frontmatter"
	cnst "github.com/danishprakash/kizai/constants"
	"github.com/danishprakash/kizai/template"
	"github.com/sirupsen/logrus"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
)

type Meta struct {
	Name  string `yaml:"name"`
	URL   string `yaml:"url"`
	Title string `yaml:"title"`
}

// Page stores an information for a page
// It could be /about, or /pages/* or even /
type Page struct {
	Frontmatter map[string]interface{}
	Meta
	Markdown string
	Body     string
	Post     *Post
	Posts    []*Post
}

type Post struct {
	Title       string
	Slug        string
	Date        time.Time
	Frontmatter map[string]interface{}
	Body        string
	URL         string
	Images      []string
}

// ParseMarkdown parse a markdown file into frontmatter and body
func (p *Page) ParseMarkdown(file string) error {
	fl, err := os.Open(file)
	if err != nil {
		panic(err)
	}

	md, err := frontmatter.Parse(fl, &p.Frontmatter)
	if err != nil {
		return err
	}

	p.Markdown = string(md)
	return nil
}

func Readfile(filepath string) []byte {
	d, err := os.ReadFile(filepath)
	if err != nil {
		logrus.Errorf("failed to read index file: %+v", err)
		return nil
	}

	return d
}

func MarkdownToHTML(md string) string {
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)
	return string(markdown.ToHTML([]byte(md), nil, renderer))
}

// renders final HTML via templates
func (p *Page) RenderHTML(htmlFile string) error {
	t, err := template.Load(cnst.TEMPLATES)
	if err != nil {
		logrus.Errorf("RenderHTML: %+v", err)
		return err
	}

	// created final html file:
	//    build/about/index.html
	//    build/post/slug/index.html
	f, err := os.Create(htmlFile)
	if err != nil {
		return fmt.Errorf("failed to create file", err)
	}

	// set post.html as default template
	if p.Frontmatter["layout"] == "" {
		p.Frontmatter["layout"] = "post"
	}

	tmplName := fmt.Sprintf("%s.html", p.Frontmatter["layout"])
	if err := t.ExecuteTemplate(f, tmplName, p); err != nil {
		logrus.Errorf("failed to execute template %w", err)
		return err
	}

	return nil
}
