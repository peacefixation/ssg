package util

import (
	"os"

	"github.com/peacefixation/static-site-generator/internal/file"
)

// CreateDir creates all directories in the path if they do not exist.
func CreateDir(dirCreator file.DirCreator, path string) error {
	err := os.MkdirAll(path, 0755)
	if err != nil {
		return err
	}

	return nil
}
