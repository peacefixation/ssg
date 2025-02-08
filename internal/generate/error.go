package generate

import "fmt"

type ErrCreateFile struct {
	Err error
}

func (e ErrCreateFile) Error() string {
	return fmt.Sprintf("failed to create file. %v", e.Err)
}

func (e ErrCreateFile) Unwrap() error {
	return e.Err
}
