package build

import (
	"github.com/peacefixation/static-site-generator/internal/generate"
	"github.com/peacefixation/static-site-generator/internal/model"
	"github.com/peacefixation/static-site-generator/internal/util"
)

type Config struct {
	ContentDir     string
	TemplateDir    string
	OutputDir      string
	StaticDir      string
	SiteConfigPath string
	LinkConfigPath string
	Title          string
	ChromaStyle    string
	Links          []model.Link
}

func BuildSite(config Config) error {
	err := util.CreateDir(config.OutputDir)
	if err != nil {
		return err
	}

	headerFragmentData := generate.HeaderFragmentData{
		Title: config.Title,
	}

	headerFragment, err := generate.GenerateHeaderFragment(headerFragmentData)
	if err != nil {
		return err
	}

	generator := generate.NewGenerator(config.ContentDir, config.TemplateDir, config.OutputDir, headerFragment, config.Title)

	err = generator.GenerateLinksPage(config.Links)
	if err != nil {
		return err
	}

	posts, err := generator.GeneratePosts()
	if err != nil {
		return err
	}

	err = generator.GenerateIndex(posts)
	if err != nil {
		return err
	}

	err = generator.GenerateTagPages(posts)
	if err != nil {
		return err
	}

	err = copyStaticFiles(config.OutputDir, config.StaticDir)
	if err != nil {
		return err
	}

	err = generator.GenerateChromaCSS(config.ChromaStyle)
	if err != nil {
		return err
	}

	return nil
}
