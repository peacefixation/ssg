package generate_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/peacefixation/static-site-generator/internal/generate"
)

type BufferWriteCloser struct {
	bytes.Buffer
}

func (b *BufferWriteCloser) Close() error {
	return nil
}

func (b *BufferWriteCloser) String() string {
	return b.Buffer.String()
}

type MockFileCreator struct {
	WriteCloser io.WriteCloser
}

func (m *MockFileCreator) Create(name string) (io.WriteCloser, error) {
	m.WriteCloser = &BufferWriteCloser{}
	return m.WriteCloser, nil
}

func TestGenerateAboutPage(t *testing.T) {
	fileCreator := &MockFileCreator{}

	g := generate.NewGenerator(
		"",
		"../../templates",
		"",
		"",
		"Peace Fixation",
		"title-svg.html",
		nil,
		nil,
		fileCreator,
		"",
		nil,
	)

	err := g.GenerateAboutPage()
	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}

	if fileCreator.WriteCloser == nil {
		t.Errorf("Expected a file to be created, but got nil")
	}

	if !strings.HasPrefix(fileCreator.WriteCloser.(*BufferWriteCloser).String(), "<!DOCTYPE html>") {
		t.Errorf("Expected file to contain HTML content")
	}
}
