package generate

import (
	"bytes"
	"html/template"
	"path/filepath"

	"github.com/peacefixation/static-site-generator/internal/tmpl"
)

const (
	headerFragmentTemplate = "header.html"
)

type HeaderFragmentData struct {
	Title string
}

func (g *Generator) GenerateHeaderFragment(title string) (template.HTML, error) {
	data := HeaderFragmentData{
		Title: title,
	}

	var buf bytes.Buffer

	err := tmpl.Process(filepath.Join(g.TemplateDir, headerFragmentTemplate), &buf, data)
	if err != nil {
		return "", ErrGenerateFragment{headerFragmentTemplate, err}
	}

	return template.HTML(buf.String()), nil
}
