package tmpl

import "fmt"

type ErrParseTemplate struct{ error }

func (e ErrParseTemplate) Error() string {
	return fmt.Sprintf("failed to parse template. %v", e.error)
}

func (e ErrParseTemplate) Unwrap() error { return e.error }

type ErrExecuteTemplate struct{ error }

func (e ErrExecuteTemplate) Error() string {
	return fmt.Sprintf("failed to execute template. %v", e.error)
}

func (e ErrExecuteTemplate) Unwrap() error { return e.error }
