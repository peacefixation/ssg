package build

import (
	"os"

	"github.com/peacefixation/static-site-generator/internal/generate"
	"github.com/peacefixation/static-site-generator/internal/parse"
	"github.com/peacefixation/static-site-generator/internal/util"
)

type Config struct {
	ContentDir     string
	TemplateDir    string
	OutputDir      string
	StaticDir      string
	SiteConfigPath string
	LinkConfigPath string
}

func BuildSite(config Config) error {
	err := util.CreateDir(config.OutputDir)
	if err != nil {
		return err
	}

	siteConfigContent, err := os.ReadFile(config.SiteConfigPath)
	if err != nil {
		return err
	}

	siteConfig, err := parse.ParseSiteConfig(siteConfigContent)
	if err != nil {
		return err
	}

	headerFragmentData := generate.HeaderFragmentData{
		Title: siteConfig.Title,
	}

	headerFragment, err := generate.GenerateHeaderFragment(headerFragmentData)
	if err != nil {
		return err
	}

	linkContent, err := os.ReadFile(config.LinkConfigPath)
	if err != nil {
		return err
	}

	linkData, err := parse.ParseLinks(linkContent)
	if err != nil {
		return err
	}

	generator := generate.NewGenerator(config.ContentDir, config.TemplateDir, config.OutputDir, headerFragment, siteConfig)

	err = generator.GenerateLinksPage(linkData.Links)
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

	err = generator.GenerateChromaCSS()
	if err != nil {
		return err
	}

	return nil
}
