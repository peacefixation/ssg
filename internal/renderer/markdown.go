package renderer

import (
	"bytes"
	"fmt"

	"github.com/russross/blackfriday/v2"
	"gopkg.in/yaml.v3"
)

const fmDelimiter = "---"

// ParseMarkdown splits src into YAML front matter and a Blackfriday-rendered HTML body.
// If src has no front matter delimiter, the entire content is treated as Markdown.
func ParseMarkdown(src []byte) (frontMatter map[string]any, bodyHTML string, err error) {
	frontMatter = make(map[string]any)

	content := bytes.TrimSpace(src)
	if !bytes.HasPrefix(content, []byte(fmDelimiter)) {
		bodyHTML = renderMarkdown(src)
		return frontMatter, bodyHTML, nil
	}

	// Strip the opening ---
	content = content[len(fmDelimiter):]

	// Find the closing --- (must be on its own line)
	end := bytes.Index(content, []byte("\n"+fmDelimiter))
	if end == -1 {
		return nil, "", fmt.Errorf("front matter: closing %q not found", fmDelimiter)
	}

	fmBytes := content[:end]
	body := content[end+1+len(fmDelimiter):]

	if err := yaml.Unmarshal(fmBytes, &frontMatter); err != nil {
		return nil, "", fmt.Errorf("parsing front matter: %w", err)
	}

	bodyHTML = renderMarkdown(bytes.TrimSpace(body))
	return frontMatter, bodyHTML, nil
}

func renderMarkdown(src []byte) string {
	return string(blackfriday.Run(src, blackfriday.WithExtensions(
		blackfriday.CommonExtensions|blackfriday.Footnotes,
	)))
}
