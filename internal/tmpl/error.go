package tmpl

import "fmt"

type ErrParseTemplate struct {
	Err error
}

func (e ErrParseTemplate) Error() string {
	return fmt.Sprintf("failed to parse template. %v", e.Err)
}

func (e ErrParseTemplate) Unwrap() error {
	return e.Err
}

type ErrExecuteTemplate struct {
	Err error
}

func (e ErrExecuteTemplate) Error() string {
	return fmt.Sprintf("failed to execute template. %v", e.Err)
}

func (e ErrExecuteTemplate) Unwrap() error {
	return e.Err
}
