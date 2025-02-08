package parse

import (
	"bytes"
	"errors"
	"fmt"

	"gopkg.in/yaml.v2"
)

var errInvalidFrontmatter = errors.New("invalid frontmatter")

type ErrParseFrontMatter struct{ error }

func (e ErrParseFrontMatter) Error() string {
	return fmt.Sprintf("failed to parse frontmatter. %v", e.error)
}

func (e ErrParseFrontMatter) Unwrap() error { return e.error }

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
		return nil, content, ErrParseFrontMatter{err}
	}

	return &frontMatter, bytes.Join(parts[2:], delimiter), nil // TODO: is this Join necessary?
}
