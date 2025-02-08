package file_test

import (
	"bytes"
	"io"
	"testing"
)

type MockFileCreator struct {
	Files map[string]*bytes.Buffer
}

func (w *MockFileCreator) Create(name string) (io.WriteCloser, error) {
	buf := &bytes.Buffer{}
	w.Files[name] = buf
	return nopCloser{buf}, nil
}

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error { return nil }

func TestCreateFile(t *testing.T) {
	creator := &MockFileCreator{
		Files: make(map[string]*bytes.Buffer),
	}

	filename := "test.txt"
	contentStr := "Hello, world!"

	// Create a file.
	writeCloser, err := creator.Create(filename)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}
	defer writeCloser.Close()

	// Check that the file was created.
	content, ok := creator.Files[filename]
	if !ok {
		t.Fatalf("file %q was not created", filename)
	}

	// Write content to the file.
	_, err = writeCloser.Write([]byte(contentStr))
	if err != nil {
		t.Fatalf("Write() failed: %v", err)
	}

	// Check that the file has the correct content.
	if content.String() != contentStr {
		t.Fatalf("got %q, want %q", content.String(), contentStr)
	}
}

type MockFileWriter struct {
	Files map[string][]byte
}

func (w *MockFileWriter) WriteFile(filename string, data []byte, perm int) error {
	w.Files[filename] = data
	return nil
}

func TestWriteFile(t *testing.T) {
	writer := &MockFileWriter{
		Files: make(map[string][]byte),
	}

	filename := "test.txt"
	content := []byte("Hello, world!")

	err := writer.WriteFile(filename, content, 0644)
	if err != nil {
		t.Fatalf("WriteFile() failed: %v", err)
	}

	got, ok := writer.Files[filename]
	if !ok {
		t.Fatalf("file %q was not created", filename)
	}

	if !bytes.Equal(got, content) {
		t.Fatalf("got %q, want %q", got, content)
	}
}
