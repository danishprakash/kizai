package markdown

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/danishprakash/kizai/template"
	"github.com/sirupsen/logrus"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
)

const (
	BASE_DIR  = "/home/danishprakash/code/kizai-site"
	TEMPLATES = BASE_DIR + "/templates"
)

func MDToHTML(md []byte) []byte {
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)
	return markdown.ToHTML(md, nil, renderer)
}

// renders final HTML via templates
func RenderHTML(htmlBody []byte, fm map[string]interface{}, f *os.File) error {
	templPath := filepath.Join(TEMPLATES, fmt.Sprintf("%s.html", fm["layout"]))
	fmt.Println(templPath)

	t, err := template.Load(TEMPLATES)
	if err != nil {
		logrus.Errorf("RenderHTML: %+v", err)
		return err
	}

	data := struct {
		FM   map[string]interface{}
		Body string
	}{fm, string(htmlBody)}
	tmplName := fmt.Sprintf("%s.html", fm["layout"])
	if err := t.ExecuteTemplate(f, tmplName, data); err != nil {
		fmt.Println("failed to execute template ", err)
		return err
	}

	return nil
}
