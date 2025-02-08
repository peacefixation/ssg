package generate

import (
	"html/template"

	"github.com/peacefixation/static-site-generator/internal/file"
)

type Generator struct {
	FileCreator    file.FileCreator
	ContentDir     string
	TemplateDir    string
	OutputDir      string
	HeaderFragment template.HTML
	Title          string
}

func NewGenerator(contentDir, templateDir, outputDir string, headerFragment template.HTML, title string, fileCreator file.FileCreator) *Generator {
	return &Generator{
		FileCreator:    fileCreator,
		ContentDir:     contentDir,
		TemplateDir:    templateDir,
		OutputDir:      outputDir,
		HeaderFragment: headerFragment,
		Title:          title,
	}
}
