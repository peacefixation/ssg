package generate

import (
	"bytes"
	"html/template"
	"path/filepath"

	"github.com/dyatlov/go-opengraph/opengraph"
	"github.com/peacefixation/static-site-generator/internal/model"
	"github.com/peacefixation/static-site-generator/internal/tmpl"
)

type LinksPageData struct {
	Header template.HTML
	Links  []model.Link
}

func (g Generator) GenerateLinksPage(links []model.Link) error {
	var err error
	for i, link := range links {
		links[i].Fragment, err = processLinkFragment(link)
		if err != nil {
			return err
		}
	}

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

func hasOpenGraphData(og *opengraph.OpenGraph) bool {
	return og.Title != ""
}

func processLinkFragment(link model.Link) (template.HTML, error) {
	var buf bytes.Buffer
	templateName := "link-list-item.html"

	if hasOpenGraphData(link.OpenGraph) {
		templateName = "link-list-item-og.html"
	}

	err := tmpl.Process(templateName, &buf, link)
	if err != nil {
		return "", ErrGenerateFragment{templateName, err}
	}

	return template.HTML(buf.String()), nil
}
