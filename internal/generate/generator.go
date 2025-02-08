package generate

import (
	"html/template"
)

type Generator struct {
	ContentDir     string
	TemplateDir    string
	OutputDir      string
	HeaderFragment template.HTML
	Title          string
}

func NewGenerator(contentDir, templateDir, outputDir string, headerFragment template.HTML, title string) Generator {
	return Generator{
		ContentDir:     contentDir,
		TemplateDir:    templateDir,
		OutputDir:      outputDir,
		HeaderFragment: headerFragment,
		Title:          title,
	}
}
