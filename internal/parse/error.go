package parse

import "fmt"

type ErrParseFile struct {
	Filename string
	Err      error
}

func (e ErrParseFile) Error() string {
	return fmt.Sprintf("error parsing %s: %v", e.Filename, e.Err)
}

func (e ErrParseFile) Unwrap() error {
	return e.Err
}
