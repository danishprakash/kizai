package markdown

import (
	"fmt"
	"os"

	"github.com/danishprakash/kizai/template"
	"github.com/gernest/front"
	"github.com/sirupsen/logrus"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
)

const (
	BASE_DIR  = "/home/danishprakash/code/kizai-site"
	TEMPLATES = BASE_DIR + "/templates"
)

type Page struct {
	FM     map[string]interface{}
	Body   string
	MDBody string
}

func (p *Page) ParseFrontmatter(file string) error {
	// parse frontmatter and body from md file
	m := front.NewMatter()
	m.Handle("---", front.YAMLHandler)
	fl, err := os.Open(file)
	fm, md, err := m.Parse(fl)
	if err != nil {
		// logrus.Errorf("err: %+v", err)
		return err
	}

	p.FM = fm
	p.MDBody = md

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

func MarkdownToHTML(mdFilepath string) []byte {
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)
	return markdown.ToHTML(Readfile(mdFilepath), nil, renderer)
}

// renders final HTML via templates
func (p *Page) RenderHTML(htmlBody []byte, htmlFile string) error {
	t, err := template.Load(TEMPLATES)
	if err != nil {
		logrus.Errorf("RenderHTML: %+v", err)
		return err
	}

	// created final html file:
	//    build/about/index.html
	//    build/post/slug/index.html
	f, err := os.Create(htmlFile)
	if err != nil {
		fmt.Println("failed to create file", err)
	}

	data := struct {
		FM   map[string]interface{}
		Body string
	}{p.FM, string(htmlBody)}
	tmplName := fmt.Sprintf("%s.html", p.FM["layout"])
	if err := t.ExecuteTemplate(f, tmplName, data); err != nil {
		fmt.Println("failed to execute template ", err)
		return err
	}

	return nil
}
