package parse

import (
	"gopkg.in/yaml.v2"

	"github.com/peacefixation/static-site-generator/internal/model"
)

type LinksData struct {
	Links []model.Link `yaml:"links"`
}

func ParseLinks(content []byte) (LinksData, error) {
	var linkData LinksData
	err := yaml.Unmarshal(content, &linkData)
	return linkData, err
}
