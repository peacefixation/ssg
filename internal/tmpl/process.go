package tmpl

import (
	"fmt"
	"html/template"
	"io"
)

func Process(templatePath string, w io.Writer, data any) error {
	fmt.Println("Processing template:", templatePath)
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return ErrParseTemplate{err}
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		return ErrExecuteTemplate{err}
	}

	return nil
}
