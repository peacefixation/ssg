package generate

import (
	"bytes"
	"fmt"
	"path/filepath"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/styles"
)

const (
	chromaCSSFileName = "chroma.css"
	cssDir            = "css"
)

var (
	defaultSyntaxHighlightStyle = styles.Monokai
)

func (g Generator) GenerateChromaCSS(style string) error {
	syntaxHighlightStyle, ok := styles.Registry[style]
	if !ok {
		fmt.Printf("Invalid syntax highlight style: '%s', using default.\n", style)
		syntaxHighlightStyle = defaultSyntaxHighlightStyle
	}

	css, err := generateChromaCSS(syntaxHighlightStyle)
	if err != nil {
		return err
	}

	writer, err := g.FileCreator.Create(filepath.Join(g.OutputDir, cssDir, chromaCSSFileName))
	if err != nil {
		return err
	}
	defer writer.Close()

	_, err = writer.Write([]byte(css))
	if err != nil {
		return err
	}

	return nil
}

// GenerateChromaCSS generates the CSS for the Chroma syntax highlighting library
// used by the Goldmark markdown parser.
func generateChromaCSS(style *chroma.Style) (string, error) {
	var cssBuf bytes.Buffer
	formatter := html.New(html.WithClasses(true))
	err := formatter.WriteCSS(&cssBuf, style)
	if err != nil {
		return "", err
	}

	return cssBuf.String(), nil
}
