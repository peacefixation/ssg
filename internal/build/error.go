package build

import "fmt"

type ErrCopyStaticFile struct {
	Err error
}

func (e ErrCopyStaticFile) Error() string {
	return fmt.Sprintf("error copying file: %v", e.Err)
}

func (e ErrCopyStaticFile) Unwrap() error {
	return e.Err
}
