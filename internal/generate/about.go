package generate

import (
	"html/template"
	"path/filepath"

	"github.com/peacefixation/static-site-generator/internal/tmpl"
)

type AboutPageData struct {
	Header template.HTML
	Title  string
}

func (g Generator) GenerateAboutPage() error {
	out, err := g.FileCreator.Create(filepath.Join(g.OutputDir, "about.html"))
	if err != nil {
		return ErrGenerateFile{"links.html", err}
	}
	defer out.Close()

	linksPageData := AboutPageData{
		Header: g.HeaderFragment,
		Title:  g.Title,
	}

	err = tmpl.Process("about.html", out, linksPageData)
	if err != nil {
		return ErrGenerateFile{"about.html", err}
	}

	return nil
}
