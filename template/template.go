package template

import (
	"text/template"

	"github.com/sirupsen/logrus"
)

func Load(tmplDir string) (*template.Template, error) {
	t, err := template.ParseGlob(tmplDir + "/*")
	if err != nil {
		logrus.Errorf("Load: failed to parse glob pattern: %+v", err)
		return nil, err
	}
	return t, nil
}
