package generate

import (
	"html/template"
	"path/filepath"

	"github.com/peacefixation/static-site-generator/internal/tmpl"
	"github.com/peacefixation/static-site-generator/internal/util"
)

type tagData struct {
	Header template.HTML
	Tag    string
	Posts  []Post
}

func (g Generator) GenerateTagPages(posts []Post) error {
	// create ouput directory
	tagDir := filepath.Join(g.OutputDir, "tags")
	if err := util.CreateDir(tagDir); err != nil {
		return err
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

func groupPostsByTag(posts []Post) map[string][]Post {
	tagMap := make(map[string][]Post)
	for _, post := range posts {
		for _, tag := range post.Tags {
			tagMap[tag] = append(tagMap[tag], post)
		}
	}

	return tagMap
}

func (g Generator) generateTagPage(tagDir, tag string, tagPosts []Post) error {
	out, err := g.FileCreator.Create(filepath.Join(tagDir, tag+".html"))
	if err != nil {
		return ErrCreateFile{Err: err}
	}
	defer out.Close()

	err = tmpl.Process("tag.html", out, tagData{
		Header: g.HeaderFragment,
		Tag:    tag,
		Posts:  tagPosts,
	})

	return err
}
