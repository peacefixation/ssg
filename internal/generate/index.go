package generate

import (
	"html/template"
	"path/filepath"

	"github.com/peacefixation/static-site-generator/internal/model"
	"github.com/peacefixation/static-site-generator/internal/tmpl"
)

type indexData struct {
	Header template.HTML
	Title  string
	Posts  []model.Post
}

func (g Generator) GenerateIndex(posts []model.Post) error {
	out, err := g.FileCreator.Create(filepath.Join(g.OutputDir, "index.html"))
	if err != nil {
		return ErrGenerateFile{"index.html", err}
	}
	defer out.Close()

	indexData := indexData{
		Header: g.HeaderFragment,
		Title:  g.Title,
		Posts:  posts,
	}

	err = tmpl.Process("index.html", out, indexData)
	if err != nil {
		return ErrGenerateFile{"index.html", err}
	}

	return nil
}
