package generate

import (
	"html/template"
	"path/filepath"

	"github.com/peacefixation/static-site-generator/internal/tmpl"
)

type indexData struct {
	Header template.HTML
	Title  string
	Posts  []Post
}

func (g Generator) GenerateIndex(posts []Post) error {
	out, err := g.FileCreator.Create(filepath.Join(g.OutputDir, "index.html"))
	if err != nil {
		return ErrCreateFile{Err: err}
	}
	defer out.Close()

	indexData := indexData{
		Header: g.HeaderFragment,
		Title:  g.Title,
		Posts:  posts,
	}

	return tmpl.Process("index.html", out, indexData)
}
