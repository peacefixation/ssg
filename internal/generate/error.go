package generate

import "fmt"

type ErrGenerateFile struct {
	Path string
	error
}

func (e ErrGenerateFile) Error() string {
	return fmt.Sprintf("failed to generate file %q. %v", e.Path, e.error)
}

func (e ErrGenerateFile) Unwrap() error { return e.error }

type ErrGenerateFragment struct {
	Name string
	error
}

func (e ErrGenerateFragment) Error() string {
	return fmt.Sprintf("failed to generate fragment %q. %v", e.Name, e.error)
}

func (e ErrGenerateFragment) Unwrap() error { return e.error }
