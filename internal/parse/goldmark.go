package parse

import (
	"bytes"
	"fmt"

	formatters_html "github.com/alecthomas/chroma/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

type ErrConvertMarkdownToHTML struct{ error }

func (e ErrConvertMarkdownToHTML) Error() string {
	return fmt.Sprintf("failed to convert markdown to HTML. %v", e.error)
}

func (e ErrConvertMarkdownToHTML) Unwrap() error { return e.error }

func ParseGoldmark(content []byte) (string, error) {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			highlighting.NewHighlighting(
				highlighting.WithFormatOptions(
					formatters_html.WithClasses(true),
				),
			),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
			html.WithUnsafe(),
		),
	)

	var htmlBuf bytes.Buffer
	if err := md.Convert(content, &htmlBuf); err != nil {
		return "", ErrConvertMarkdownToHTML{err}
	}

	return htmlBuf.String(), nil
}
