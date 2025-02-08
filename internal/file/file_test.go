package file_test

import (
	"bytes"
	"errors"
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

type MockFileReader struct {
	Files map[string][]byte
}

func (r *MockFileReader) ReadFile(name string) ([]byte, error) {
	content, ok := r.Files[name]
	if !ok {
		return nil, errors.New("file not found")
	}
	return content, nil
}

func TestReadFile(t *testing.T) {
	reader := &MockFileReader{
		Files: map[string][]byte{
			"test.txt": []byte("Hello, world!"),
		},
	}

	filename := "test.txt"
	content, err := reader.ReadFile(filename)
	if err != nil {
		t.Fatalf("ReadFile() failed: %v", err)
	}

	want := []byte("Hello, world!")

	if !bytes.Equal(content, want) {
		t.Fatalf("got %q, want %q", content, want)
	}
}

type MockDirCreator struct {
	Dirs map[string]bool
}

func (c *MockDirCreator) MkdirAll(path string, perm int) error {
	c.Dirs[path] = true
	return nil
}

func TestMkdirAll(t *testing.T) {
	creator := &MockDirCreator{
		Dirs: make(map[string]bool),
	}

	path := "test"
	err := creator.MkdirAll(path, 0755)
	if err != nil {
		t.Fatalf("MkdirAll() failed: %v", err)
	}

	if !creator.Dirs[path] {
		t.Fatalf("directory %q was not created", path)
	}
}
