package model

import "html/template"

type Post struct {
	Title         string
	Date          string
	FormattedDate string
	Description   string
	URL           template.URL
	Tags          []string
	Header        template.HTML
	Content       template.HTML
	ListItem      template.HTML
}
