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
	Title             string
	TitleFragmentPath string
	TitleFragment     template.HTML
}

const defaultTitleFragment = `<h1>{{.Title}}</h1>`

func (g *Generator) GenerateHeaderFragment(title string) (template.HTML, error) {
	data := HeaderFragmentData{
		Title:             title,
		TitleFragmentPath: g.TitleFragmentPath,
	}

	if data.TitleFragmentPath != "" {
		titleFragment, err := g.generateTitleFragment(data.TitleFragmentPath, data.Title)
		if err != nil {
			return "", err
		}
		data.TitleFragment = titleFragment
	} else {
		data.TitleFragment = defaultTitleFragment
	}

	var buf bytes.Buffer

	err := tmpl.Process(filepath.Join(g.TemplateDir, headerFragmentTemplate), &buf, data)
	if err != nil {
		return "", ErrGenerateFragment{headerFragmentTemplate, err}
	}

	return template.HTML(buf.String()), nil
}

func (g *Generator) generateTitleFragment(path, title string) (template.HTML, error) {
	var buf bytes.Buffer

	err := tmpl.Process(filepath.Join(g.TemplateDir, path), &buf, struct{ Title string }{title})
	if err != nil {
		return "", ErrGenerateFragment{path, err}
	}

	return template.HTML(buf.String()), nil
}
