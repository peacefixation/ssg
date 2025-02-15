package build

import (
	"github.com/peacefixation/static-site-generator/internal/file"
	"github.com/peacefixation/static-site-generator/internal/generate"
	"github.com/peacefixation/static-site-generator/internal/model"
	"github.com/peacefixation/static-site-generator/internal/util"
)

type Config struct {
	ContentDir        string
	TemplateDir       string
	OutputDir         string
	StaticDir         string
	Title             string
	TitleFragmentPath string
	ChromaStyle       string
	Links             []model.Link
}

func BuildSite(config Config, dirCreator file.DirCreator, fileReader file.FileReader, fileCreator file.FileCreator) error {
	err := util.CreateDir(dirCreator, config.OutputDir)
	if err != nil {
		return err
	}

	generator := generate.NewGenerator(config.ContentDir, config.TemplateDir, config.StaticDir, config.OutputDir, config.Title, config.TitleFragmentPath, dirCreator, fileReader, fileCreator)

	// other pages use the header so this must be generated first
	err = generator.GenerateHeaderFragment()
	if err != nil {
		return err
	}

	// posts are the core of the site, they are listed on the index page and used to generate the tag pages
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

	err = generator.GenerateLinksPage(config.Links)
	if err != nil {
		return err
	}

	err = copyStaticFiles(config.OutputDir, config.StaticDir, dirCreator, fileReader, fileCreator)
	if err != nil {
		return err
	}

	// chroma CSS is used to style code blocks on the post pages
	err = generator.GenerateChromaCSS(config.ChromaStyle)
	if err != nil {
		return err
	}

	return nil
}
