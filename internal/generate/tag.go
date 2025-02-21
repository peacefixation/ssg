package generate

import (
	"html/template"
	"path/filepath"

	"github.com/peacefixation/static-site-generator/internal/model"
	"github.com/peacefixation/static-site-generator/internal/tmpl"
	"github.com/peacefixation/static-site-generator/internal/util"
)

const (
	tagPageTemplate = "tag.html"
)

type tagData struct {
	Header template.HTML
	Tag    string
	Posts  []model.Post
}

func (g Generator) GenerateTagPages(posts []model.Post) error {
	// create ouput directory
	tagDir := filepath.Join(g.OutputDir, "tags")
	if err := util.CreateDir(g.DirCreator, tagDir); err != nil {
		return ErrGenerateFile{tagDir, err}
	}

	tagMap := groupPostsByTag(posts)

	// generate a page for each tag
	for tag, tagPosts := range tagMap {
		if err := g.generateTagPage(tagDir, tag, tagPosts); err != nil {
			return err
		}
	}

	return nil
}

func groupPostsByTag(posts []model.Post) map[string][]model.Post {
	tagMap := make(map[string][]model.Post)
	for _, post := range posts {
		for _, tag := range post.Tags {
			tagMap[tag] = append(tagMap[tag], post)
		}
	}

	return tagMap
}

func (g Generator) generateTagPage(tagDir, tag string, tagPosts []model.Post) error {
	path := filepath.Join(tagDir, tag+".html")
	out, err := g.FileCreator.Create(path)
	if err != nil {
		return ErrGenerateFile{path, err}
	}
	defer out.Close()

	err = tmpl.Process(filepath.Join(g.TemplateDir, tagPageTemplate), out, tagData{
		Header: g.HeaderFragment,
		Tag:    tag,
		Posts:  tagPosts,
	})
	if err != nil {
		return ErrGenerateFile{path, err}
	}

	return nil
}
