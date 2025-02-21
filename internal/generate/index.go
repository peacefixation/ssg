package generate

import (
	"html/template"
	"path/filepath"

	"github.com/peacefixation/static-site-generator/internal/model"
	"github.com/peacefixation/static-site-generator/internal/tmpl"
)

const (
	indexPageTemplate = "index.html"
	indexPageOutput   = "index.html"
)

type indexData struct {
	Header template.HTML
	Title  string
	Posts  []model.Post
}

func (g Generator) GenerateIndex(posts []model.Post) error {
	out, err := g.FileCreator.Create(filepath.Join(g.OutputDir, indexPageOutput))
	if err != nil {
		return ErrGenerateFile{indexPageOutput, err}
	}
	defer out.Close()

	indexData := indexData{
		Header: g.HeaderFragment,
		Title:  g.Title,
		Posts:  posts,
	}

	err = tmpl.Process(filepath.Join(g.TemplateDir, indexPageTemplate), out, indexData)
	if err != nil {
		return ErrGenerateFile{indexPageOutput, err}
	}

	return nil
}
