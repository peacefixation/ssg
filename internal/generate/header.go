package generate

import (
	"bytes"
	"html/template"

	"github.com/peacefixation/static-site-generator/internal/tmpl"
)

type HeaderFragmentData struct {
	Title             string
	TitleFragmentPath string
	TitleFragment     template.HTML
}

const defaultTitleFragment = `<h1>{{.Title}}</h1>`

func (g *Generator) GenerateHeaderFragment() error {
	data := HeaderFragmentData{
		Title:             g.Title,
		TitleFragmentPath: g.TitleFragmentPath,
	}

	if data.TitleFragmentPath != "" {
		titleFragment, err := generateTitleFragment(data.TitleFragmentPath, data.Title)
		if err != nil {
			return err
		}
		data.TitleFragment = titleFragment
	} else {
		data.TitleFragment = defaultTitleFragment
	}

	var buf bytes.Buffer

	err := tmpl.Process("header.html", &buf, data)
	if err != nil {
		return ErrGenerateFragment{"header.html", err}
	}

	g.HeaderFragment = template.HTML(buf.String())

	return nil
}

func generateTitleFragment(path, title string) (template.HTML, error) {
	var buf bytes.Buffer

	err := tmpl.Process(path, &buf, struct{ Title string }{title})
	if err != nil {
		return "", ErrGenerateFragment{path, err}
	}

	return template.HTML(buf.String()), nil
}
