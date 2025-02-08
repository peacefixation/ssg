package generate

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/peacefixation/static-site-generator/internal/parse"
	"github.com/peacefixation/static-site-generator/internal/tmpl"
	"github.com/peacefixation/static-site-generator/internal/util"
)

const (
	contentPostsDir    = "posts"
	outputPostsDir     = "posts"
	contentPostFileExt = ".md"
	outputPostFileExt  = ".html"
)

type Post struct {
	Title         string
	Date          string
	FormattedDate string
	Description   string
	URL           template.URL
	Tags          []string
	Header        template.HTML
	Content       template.HTML
	ListItem      template.HTML
}

func (g Generator) GeneratePosts() ([]Post, error) {
	err := util.CreateDir(g.DirCreator, filepath.Join(g.OutputDir, outputPostsDir))
	if err != nil {
		return nil, err
	}

	var posts []Post

	err = filepath.Walk(filepath.Join(g.ContentDir, contentPostsDir), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Ext(path) != contentPostFileExt {
			return nil
		}

		post, err := g.GeneratePost(path)
		if err != nil {
			return err
		}

		posts = append(posts, post)

		return nil
	})
	if err != nil {
		return nil, err
	}

	// reverse the slice so we render posts in reverse chronological order
	util.ReverseSlice(posts)

	return posts, nil
}

func (g Generator) GeneratePost(path string) (Post, error) {
	content, err := g.FileReader.ReadFile(path)
	if err != nil {
		return Post{}, ErrGenerateFile{path, err}
	}

	frontMatter, content, err := parse.ParseFrontmatter(content)
	if err != nil {
		return Post{}, ErrGenerateFile{path, err}
	}

	htmlContent, err := parse.ParseGoldmark(content)
	if err != nil {
		return Post{}, ErrGenerateFile{path, err}
	}

	outputFilename, err := generateOutputFilename(path)
	if err != nil {
		return Post{}, ErrGenerateFile{path, err}
	}

	outputPath := filepath.Join(outputPostsDir, outputFilename)

	date, err := time.Parse(time.RFC3339, frontMatter.Date)
	if err != nil {
		return Post{}, ErrGenerateFile{path, err}
	}

	post := Post{
		Title:         frontMatter.Title,
		Date:          frontMatter.Date,
		FormattedDate: date.Format("January 2, 2006"),
		Tags:          frontMatter.Tags,
		URL:           template.URL(outputPath),
		Description:   frontMatter.Description,
		Header:        g.HeaderFragment,
		Content:       template.HTML(htmlContent),
	}

	post.ListItem, err = g.GeneratePostListItem(post)
	if err != nil {
		return post, ErrGenerateFile{path, err}
	}

	out, err := g.FileCreator.Create(filepath.Join(g.OutputDir, outputPath))
	if err != nil {
		return post, ErrGenerateFile{path, err}
	}
	defer out.Close()

	err = tmpl.Process("post.html", out, post)
	if err != nil {
		return post, ErrGenerateFile{path, err}
	}

	return post, nil
}

func (g Generator) GeneratePostListItem(post Post) (template.HTML, error) {
	var buf bytes.Buffer

	err := tmpl.Process("post-list-item.html", &buf, post)
	if err != nil {
		return "", err
	}

	return template.HTML(buf.String()), nil
}

func generateOutputFilename(path string) (string, error) {
	filenameBase := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	filenameParts := strings.Split(filenameBase, "_")
	if len(filenameParts) < 2 {
		return "", fmt.Errorf("invalid filename: %s", path)
	}
	filename := strings.Join(filenameParts[1:], "_")
	filename += outputPostFileExt
	return filename, nil
}
