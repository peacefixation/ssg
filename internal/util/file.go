package util

import "os"

// CreateDir creates all directories in the path if they do not exist.
func CreateDir(path string) error {
	err := os.MkdirAll(path, 0755)
	if err != nil {
		return err
	}

	return nil
}
