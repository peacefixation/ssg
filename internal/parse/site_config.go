package parse

import "gopkg.in/yaml.v2"

type SiteConfig struct {
	Title                string `yaml:"title"`
	SyntaxHighlightStyle string `yaml:"syntax-highlight-style"`
}

func ParseSiteConfig(content []byte) (SiteConfig, error) {
	var config SiteConfig
	err := yaml.Unmarshal(content, &config)
	return config, err
}
