package markdown

import (
	"fmt"
	"os"

	"github.com/adrg/frontmatter"
	cnst "github.com/danishprakash/kizai/constants"
	"github.com/danishprakash/kizai/template"
	"github.com/sirupsen/logrus"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
)

type Page struct {
	FM     map[string]interface{}
	Body   string
	MDBody string
}

// func (p *Page) ParseFrontmatter(file string) error {
// 	// parse frontmatter and body from md file
// 	m := front.NewMatter()
// 	m.Handle("---", front.YAMLHandler)
// 	fl, err := os.Open(file)
// 	fm, md, err := m.Parse(fl)
// 	if err != nil {
// 		// logrus.Errorf("err: %+v", err)
// 		return err
// 	}

// 	p.FM = fm
// 	p.MDBody = md

// 	return nil
// }

func (p *Page) ParseFrontmatter(file string) error {
	fl, err := os.Open(file)
	if err != nil {
		panic(err)
	}

	md, err := frontmatter.Parse(fl, &p.FM)
	if err != nil {
		return err
	}
	p.MDBody = string(md)
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

func MarkdownToHTML(mdBody string) []byte {
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)
	return markdown.ToHTML([]byte(mdBody), nil, renderer)
}

// renders final HTML via templates
func (p *Page) RenderHTML(htmlFile string, data interface{}) error {
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
		fmt.Println("failed to create file", err)
	}

	// set post.html as default template
	if p.FM["layout"] == "" {
		p.FM["layout"] = "post"
	}

	tmplName := fmt.Sprintf("%s.html", p.FM["layout"])
	if err := t.ExecuteTemplate(f, tmplName, data); err != nil {
		fmt.Println("failed to execute template ", err)
		return err
	}

	return nil
}
