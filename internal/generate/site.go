package generate

import (
	"github.com/peacefixation/static-site-generator/internal/util"
)

func (g Generator) GenerateSite() error {
	err := util.CreateDir(g.DirCreator, g.OutputDir)
	if err != nil {
		return err
	}

	// other pages use the header so this must be generated first
	headerFragment, err := g.GenerateHeaderFragment(g.Title)
	if err != nil {
		return err
	}

	g.HeaderFragment = headerFragment

	// posts are the core of the site, they are listed on the index page and used to generate the tag pages
	posts, err := g.GeneratePosts()
	if err != nil {
		return err
	}

	err = g.GenerateIndex(posts)
	if err != nil {
		return err
	}

	err = g.GenerateTagPages(posts)
	if err != nil {
		return err
	}

	err = g.GenerateLinksPage(g.Links)
	if err != nil {
		return err
	}

	err = g.GenerateAboutPage()
	if err != nil {
		return err
	}

	err = copyStaticFiles(g.OutputDir, g.StaticDir, g.DirCreator, g.FileReader, g.FileCreator)
	if err != nil {
		return err
	}

	err = copyCSSFiles(g.OutputDir, g.StaticDir, g.DirCreator, g.FileReader, g.FileCreator)
	if err != nil {
		return err
	}

	// chroma CSS is used to style code blocks on the post pages
	err = g.GenerateChromaCSS(g.ChromaStyle)
	if err != nil {
		return err
	}

	return nil
}
