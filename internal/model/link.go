package model

import (
	"html/template"

	"github.com/dyatlov/go-opengraph/opengraph"
)

type Link struct {
	Name           string               `yaml:"name"`
	URL            string               `yaml:"url"`
	Category       string               `yaml:"category"`
	Tags           []string             `yaml:"tags"`
	FetchOpenGraph bool                 `yaml:"fetch-opengraph"`
	OpenGraph      *opengraph.OpenGraph `yaml:"opengraph,omitempty"`
	Fragment       template.HTML        `yaml:"-"`
}
