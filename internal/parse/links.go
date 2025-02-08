package parse

import (
	"gopkg.in/yaml.v2"
)

type Link struct {
	Name     string   `yaml:"name"`
	URL      string   `yaml:"url"`
	Category string   `yaml:"category"`
	Tags     []string `yaml:"tags"`
}

type LinksData struct {
	Links []Link `yaml:"links"`
}

func ParseLinks(content []byte) (LinksData, error) {
	var linkData LinksData
	err := yaml.Unmarshal(content, &linkData)
	return linkData, err
}
