package generate

import (
	"html/template"
	"path/filepath"

	"github.com/peacefixation/static-site-generator/internal/model"
	"github.com/peacefixation/static-site-generator/internal/tmpl"
)

type LinksPageData struct {
	Header template.HTML
	Links  []model.Link
}

func (g Generator) GenerateLinksPage(links []model.Link) error {
	out, err := g.FileCreator.Create(filepath.Join(g.OutputDir, "links.html"))
	if err != nil {
		return ErrGenerateFile{"links.html", err}
	}
	defer out.Close()

	linksPageData := LinksPageData{
		Header: g.HeaderFragment,
		Links:  links,
	}

	err = tmpl.Process("links.html", out, linksPageData)
	if err != nil {
		return ErrGenerateFile{"links.html", err}
	}

	return nil
}
