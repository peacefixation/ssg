package generate

import (
	"html/template"
	"path/filepath"

	"github.com/peacefixation/static-site-generator/internal/tmpl"
)

const (
	aboutPageTemplate = "about.html"
	aboutPageOutput   = "about.html"
)

type AboutPageData struct {
	Header template.HTML
	Title  string
}

func (g Generator) GenerateAboutPage() error {
	out, err := g.FileCreator.Create(filepath.Join(g.OutputDir, aboutPageOutput))
	if err != nil {
		return ErrGenerateFile{aboutPageOutput, err}
	}
	defer out.Close()

	linksPageData := AboutPageData{
		Header: g.HeaderFragment,
		Title:  g.Title,
	}

	err = tmpl.Process(filepath.Join(g.TemplateDir, aboutPageTemplate), out, linksPageData)
	if err != nil {
		return ErrGenerateFile{aboutPageOutput, err}
	}

	return nil
}
