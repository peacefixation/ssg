package tmpl

import (
	"html/template"
	"io"
	"path/filepath"
)

const templateDir = "templates"

func Process(templateFile string, w io.Writer, data any) error {
	tmpl, err := template.ParseFiles(filepath.Join(templateDir, templateFile))
	if err != nil {
		return ErrParseTemplate{err}
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		return ErrExecuteTemplate{err}
	}

	return nil
}
