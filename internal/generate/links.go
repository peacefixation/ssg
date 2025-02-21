package generate

import (
	"bytes"
	"html/template"
	"path/filepath"

	"github.com/dyatlov/go-opengraph/opengraph"
	"github.com/peacefixation/static-site-generator/internal/model"
	"github.com/peacefixation/static-site-generator/internal/tmpl"
)

const (
	linksPageTemplate         = "links.html"
	linkItemTemplate          = "link-list-item.html"
	linkItemOpenGraphTemplate = "link-list-item-og.html"
	linksPageOutput           = "links.html"
)

type LinksPageData struct {
	Header template.HTML
	Links  []model.Link
	Title  string
}

func (g Generator) GenerateLinksPage(links []model.Link) error {
	var err error
	for i, link := range links {
		links[i].Fragment, err = g.processLinkFragment(link)
		if err != nil {
			return err
		}
	}

	out, err := g.FileCreator.Create(filepath.Join(g.OutputDir, linksPageOutput))
	if err != nil {
		return ErrGenerateFile{linksPageOutput, err}
	}
	defer out.Close()

	linksPageData := LinksPageData{
		Header: g.HeaderFragment,
		Links:  links,
		Title:  g.Title,
	}

	err = tmpl.Process(filepath.Join(g.TemplateDir, linksPageTemplate), out, linksPageData)
	if err != nil {
		return ErrGenerateFile{linksPageOutput, err}
	}

	return nil
}

func hasOpenGraphData(og *opengraph.OpenGraph) bool {
	return og != nil && og.Title != ""
}

func (g Generator) processLinkFragment(link model.Link) (template.HTML, error) {
	var buf bytes.Buffer
	templateName := linkItemTemplate

	if link.FetchOpenGraph && hasOpenGraphData(link.OpenGraph) {
		templateName = linkItemOpenGraphTemplate
	}

	err := tmpl.Process(filepath.Join(g.TemplateDir, templateName), &buf, link)
	if err != nil {
		return "", ErrGenerateFragment{templateName, err}
	}

	return template.HTML(buf.String()), nil
}
