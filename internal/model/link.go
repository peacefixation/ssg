package model

type Link struct {
	Name     string   `yaml:"name"`
	URL      string   `yaml:"url"`
	Category string   `yaml:"category"`
	Tags     []string `yaml:"tags"`
}
