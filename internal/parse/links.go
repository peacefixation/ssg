package parse

import (
	"fmt"

	"gopkg.in/yaml.v2"

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
