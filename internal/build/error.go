package build

import "fmt"

type ErrCopyStaticFile struct{ error }

func (e ErrCopyStaticFile) Error() string {
	return fmt.Sprintf("error copying file: %v", e.error)
}

func (e ErrCopyStaticFile) Unwrap() error { return e.error }
