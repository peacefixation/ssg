package generate

import (
	"html/template"

	"github.com/peacefixation/static-site-generator/internal/file"
)

type Generator struct {
	DirCreator     file.DirCreator
	FileReader     file.FileReader
	FileCreator    file.FileCreator
	ContentDir     string
	TemplateDir    string
	StaticDir      string
	OutputDir      string
	HeaderFragment template.HTML
	Title          string
}

func NewGenerator(contentDir, templateDir, staticDir, outputDir string, headerFragment template.HTML, title string, dirCreator file.DirCreator, fileReader file.FileReader, fileCreator file.FileCreator) *Generator {
	return &Generator{
		DirCreator:     dirCreator,
		FileReader:     fileReader,
		FileCreator:    fileCreator,
		ContentDir:     contentDir,
		TemplateDir:    templateDir,
		StaticDir:      staticDir,
		OutputDir:      outputDir,
		HeaderFragment: headerFragment,
		Title:          title,
	}
}
