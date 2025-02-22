package parse

import (
	"fmt"

	"gopkg.in/yaml.v2"
)

type ErrParseSiteConfig struct{ error }

func (e ErrParseSiteConfig) Error() string {
	return fmt.Sprintf("failed to parse site config. %v", e.error)
}

func (e ErrParseSiteConfig) Unwrap() error { return e.error }

type SiteConfig struct {
	Title                string `yaml:"title"`
	SyntaxHighlightStyle string `yaml:"syntax-highlight-style"`
	OpenGraphUserAgent   string `yaml:"open-graph-user-agent"`
}

func ParseSiteConfig(content []byte) (SiteConfig, error) {
	var config SiteConfig
	err := yaml.Unmarshal(content, &config)
	if err != nil {
		return config, ErrParseSiteConfig{err}
	}
	return config, nil
}
