package generate

import (
	"html/template"
	"os"
	"path/filepath"

	"github.com/peacefixation/static-site-generator/internal/model"
	"github.com/peacefixation/static-site-generator/internal/tmpl"
)

type LinksPageData struct {
	Header template.HTML
	Links  []model.Link
}

func (g Generator) GenerateLinksPage(links []model.Link) error {
	out, err := os.Create(filepath.Join(g.OutputDir, "links.html"))
	if err != nil {
		return ErrCreateFile{Err: err}
	}
	defer out.Close()

	linksPageData := LinksPageData{
		Header: g.HeaderFragment,
		Links:  links,
	}

	return tmpl.Process("links.html", out, linksPageData)
}
