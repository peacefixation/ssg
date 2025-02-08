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

type FileReader interface {
	ReadFile(name string) ([]byte, error)
}

// OSFileReader is an implementation of FileReader that reads from the file system.
type OSFileReader struct{}

// ReadFile reads the contents of a file from the file system.
func (r OSFileReader) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

type DirCreator interface {
	MkdirAll(path string, perm os.FileMode) error
}

// OSDirCreator is an implementation of DirCreator that creates directories on the file system.
type OSDirCreator struct{}

// MkdirAll creates a directory and any necessary parents on the file system.
func (c OSDirCreator) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}
