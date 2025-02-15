package generate

import (
	"html/template"

	"github.com/peacefixation/static-site-generator/internal/file"
	"github.com/peacefixation/static-site-generator/internal/model"
)

type Generator struct {
	DirCreator        file.DirCreator
	FileReader        file.FileReader
	FileCreator       file.FileCreator
	ContentDir        string
	TemplateDir       string
	StaticDir         string
	OutputDir         string
	HeaderFragment    template.HTML
	Title             string
	TitleFragmentPath string
	ChromaStyle       string
	Links             []model.Link
}

func NewGenerator(contentDir, templateDir, staticDir, outputDir string, title, titleFragmentPath string, dirCreator file.DirCreator, fileReader file.FileReader, fileCreator file.FileCreator, chromaStyle string, links []model.Link) *Generator {
	return &Generator{
		DirCreator:        dirCreator,
		FileReader:        fileReader,
		FileCreator:       fileCreator,
		ContentDir:        contentDir,
		TemplateDir:       templateDir,
		StaticDir:         staticDir,
		OutputDir:         outputDir,
		Title:             title,
		TitleFragmentPath: titleFragmentPath,
		ChromaStyle:       chromaStyle,
		Links:             links,
	}
}
