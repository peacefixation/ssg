package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/peacefixation/ssg/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	newItemList string
	newItemType string
)

var newItemCmd = &cobra.Command{
	Use:   "item [--list <list>] [--type <type>] [key=value ...]",
	Short: "Add a new item to a list",
	Long: `Add a new item to a list.

Fields are supplied as key=value arguments after the flags. Required fields
are defined by the item type. Missing required fields produce an error.

Omit --list to add the item directly to the root content directory.

Example:
  ssg new item --list music --type youtube url=https://youtu.be/xyz title="My Song"
  ssg new item --type page title="Home"`,
	RunE: runNewItem,
}

func init() {
	newItemCmd.Flags().StringVar(&newItemList, "list", "", "list to add the item to (defaults to root content directory)")
	newItemCmd.Flags().StringVar(&newItemType, "type", "", "item type")
	newCmd.AddCommand(newItemCmd)
}

func runNewItem(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	return createItem(cfg, newItemList, newItemType, parseFields(args))
}

func createItem(cfg *config.SiteConfig, listName, typeName string, data map[string]string) error {
	var dir string
	if listName == "" {
		dir = cfg.ContentDir
	} else {
		listMeta, err := validateList(cfg.ContentDir, listName)
		if err != nil {
			return err
		}
		if err := validateType(listMeta, typeName, listName); err != nil {
			return err
		}
		dir = filepath.Join(cfg.ContentDir, listName)
	}

	format := "yaml"
	if typeName != "" {
		it, err := resolveItemType(cfg.ItemsDir, typeName)
		if err != nil {
			return err
		}
		if err := checkRequiredFields(it, data); err != nil {
			return err
		}
		format = it.Format
	}

	if format == "markdown" {
		return writeMarkdownItem(dir, typeName, data)
	}
	return writeYAMLItem(dir, typeName, data)
}

func writeYAMLItem(dir, typeName string, data map[string]string) error {
	item := make(map[string]any, len(data)+1)
	if typeName != "" {
		item["type"] = typeName
	}
	for k, v := range data {
		item[k] = v
	}
	out, err := yaml.Marshal(item)
	if err != nil {
		return fmt.Errorf("marshalling item: %w", err)
	}
	destPath := filepath.Join(dir, generateFilename(data["title"], ".yaml"))
	if err := os.WriteFile(destPath, out, 0644); err != nil {
		return fmt.Errorf("writing item: %w", err)
	}
	fmt.Printf("Created %s\n", destPath)
	return nil
}

func writeMarkdownItem(dir, typeName string, data map[string]string) error {
	fm := make(map[string]any, len(data))
	for k, v := range data {
		fm[k] = v
	}
	fmBytes, err := yaml.Marshal(fm)
	if err != nil {
		return fmt.Errorf("marshalling frontmatter: %w", err)
	}
	content := "---\n" + string(fmBytes) + "---\n\n"
	destPath := filepath.Join(dir, generateFilename(data["title"], ".md"))
	if err := os.WriteFile(destPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing item: %w", err)
	}
	fmt.Printf("Created %s\n", destPath)
	return nil
}

var scanListExts = map[string]bool{
	".md": true, ".markdown": true, ".json": true, ".yaml": true, ".yml": true,
}
