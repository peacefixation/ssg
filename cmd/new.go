package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new <name>",
	Short: "Scaffold a new site skeleton in the current directory",
	Args:  cobra.ExactArgs(1),
	RunE:  runNew,
}

func runNew(cmd *cobra.Command, args []string) error {
	name := args[0]

	dirs := []string{"templates", "content", "public"}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}

	siteYAML := fmt.Sprintf(`title: %s
baseURL: http://localhost:8080
outputDir: public
templateDir: templates
contentDir: content
`, name)
	if err := writeFile("site.yaml", siteYAML); err != nil {
		return err
	}

	indexTemplate := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>{{ .title }} — {{ .Site.Title }}</title>
</head>
<body>
  <h1>{{ .title }}</h1>
  {{ .body }}
</body>
</html>
`
	if err := writeFile("templates/index.html", indexTemplate); err != nil {
		return err
	}

	indexContent := `---
title: Welcome
---
Hello, world! This is your new SSG site.
`
	if err := writeFile("content/index.md", indexContent); err != nil {
		return err
	}

	fmt.Printf("Created new site: %s\n", name)
	fmt.Println("  ssg build")
	return nil
}

func writeFile(path, content string) error {
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
