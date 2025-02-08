package generate

import (
	"html/template"

	"github.com/peacefixation/static-site-generator/internal/parse"
)

type Generator struct {
	ContentDir     string
	TemplateDir    string
	OutputDir      string
	HeaderFragment template.HTML
	SiteConfig     parse.SiteConfig
}

func NewGenerator(contentDir, templateDir, outputDir string, headerFragment template.HTML, siteConfig parse.SiteConfig) Generator {
	return Generator{
		ContentDir:     contentDir,
		TemplateDir:    templateDir,
		OutputDir:      outputDir,
		HeaderFragment: headerFragment,
		SiteConfig:     siteConfig,
	}
}
