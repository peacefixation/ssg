package file

import (
	"io"
	"os"
)

type FileCreator interface {
	Create(name string) (io.WriteCloser, error)
}

// OSFileCreator is an implementation of FileCreator that writes to the file system.
type OSFileCreator struct{}

// Create creates a file on the file system.
func (w OSFileCreator) Create(name string) (io.WriteCloser, error) {
	return os.Create(name)
}

type FileWriter interface {
	WriteFile(filename string, data []byte, perm os.FileMode) error
}

// OSFileWriter is an implementation of FileWriter that writes to the file system.
type OSFileWriter struct{}

// WriteFile writes data to a file on the file system.
func (w OSFileWriter) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return os.WriteFile(filename, data, perm)
}
