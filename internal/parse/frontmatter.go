package parse

import (
	"bytes"
	"errors"

	"gopkg.in/yaml.v2"
)

var errInvalidFrontmatter = errors.New("invalid frontmatter")

var delimiter = []byte("---")

type Frontmatter struct {
	Title       string   `yaml:"title"`
	Date        string   `yaml:"date"`
	Tags        []string `yaml:"tags"`
	Description string   `yaml:"description"`
}

func ParseFrontmatter(content []byte) (*Frontmatter, []byte, error) {
	parts := bytes.Split(content, delimiter)
	if len(parts) < 3 {
		return nil, content, errInvalidFrontmatter
	}

	var frontMatter Frontmatter
	err := yaml.Unmarshal(parts[1], &frontMatter)
	if err != nil {
		return nil, content, err
	}

	return &frontMatter, bytes.Join(parts[2:], delimiter), nil
}
