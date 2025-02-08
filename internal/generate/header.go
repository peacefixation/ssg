package generate

import (
	"bytes"
	"html/template"

	"github.com/peacefixation/static-site-generator/internal/tmpl"
)

type HeaderFragmentData struct {
	Title string
}

func GenerateHeaderFragment(data HeaderFragmentData) (template.HTML, error) {
	var buf bytes.Buffer

	err := tmpl.Process("header.html", &buf, data)
	if err != nil {
		return "", err
	}

	return template.HTML(buf.String()), nil
}
