package generate

import (
	"bytes"
	"fmt"
	"os"
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

func (g Generator) GenerateChromaCSS() error {
	syntaxHighlightStyle, ok := styles.Registry[g.SiteConfig.SyntaxHighlightStyle]
	if !ok {
		fmt.Printf("Invalid syntax highlight style: '%s', using default.\n", g.SiteConfig.SyntaxHighlightStyle)
		syntaxHighlightStyle = defaultSyntaxHighlightStyle
	}

	css, err := generateChromaCSS(syntaxHighlightStyle)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(g.OutputDir, cssDir, chromaCSSFileName), []byte(css), 0644)
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
