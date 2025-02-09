package parse

import (
	"fmt"

	"gopkg.in/yaml.v2"

	"github.com/peacefixation/static-site-generator/internal/file"
	"github.com/peacefixation/static-site-generator/internal/model"
)

type ErrParseLinks struct{ error }

func (e ErrParseLinks) Error() string {
	return fmt.Sprintf("failed to parse links. %v", e.error)
}

func (e ErrParseLinks) Unwrap() error { return e.error }

type LinksData struct {
	Links []model.Link `yaml:"links"`
}

func ParseLinks(content []byte) (LinksData, error) {
	var linkData LinksData
	err := yaml.Unmarshal(content, &linkData)
	if err != nil {
		return linkData, ErrParseLinks{err}
	}
	return linkData, nil
}

type ErrWriteLinks struct {
	Path string
	error
}

func (e ErrWriteLinks) Error() string {
	return fmt.Sprintf("failed to write links to %q. %v", e.Path, e.error)
}

func (e ErrWriteLinks) Unwrap() error { return e.error }

func WriteLinks(linkData LinksData, fileCreator file.FileCreator, path string) error {
	content, err := yaml.Marshal(linkData)
	if err != nil {
		return ErrWriteLinks{path, err}
	}

	file, err := fileCreator.Create(path)
	if err != nil {
		return ErrWriteLinks{path, err}
	}
	defer file.Close()

	_, err = file.Write(content)
	if err != nil {
		return ErrWriteLinks{path, err}
	}

	return nil
}
