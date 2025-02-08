package build

import (
	"github.com/peacefixation/static-site-generator/internal/file"
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

func BuildSite(config Config, dirCreator file.DirCreator, fileReader file.FileReader, fileCreator file.FileCreator) error {
	err := util.CreateDir(dirCreator, config.OutputDir)
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

	generator := generate.NewGenerator(config.ContentDir, config.TemplateDir, config.StaticDir, config.OutputDir, headerFragment, config.Title, dirCreator, fileReader, fileCreator)

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

	err = copyStaticFiles(config.OutputDir, config.StaticDir, dirCreator, fileReader, fileCreator)
	if err != nil {
		return err
	}

	err = generator.GenerateChromaCSS(config.ChromaStyle)
	if err != nil {
		return err
	}

	return nil
}
