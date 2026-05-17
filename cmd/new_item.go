package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/peacefixation/ssg/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	newItemList string
	newItemType string
)

var newItemCmd = &cobra.Command{
	Use:   "item --list <list> --type <type> [key=value ...]",
	Short: "Add a new item to a list",
	Long: `Add a new item to a list.

Fields are supplied as key=value arguments after the flags. Required fields
are defined by the item type. Missing required fields produce an error.

Example:
  ssg new item --list music --type youtube url=https://youtu.be/xyz title="My Song"`,
	RunE: runNewItem,
}

func init() {
	newItemCmd.Flags().StringVar(&newItemList, "list", "", "list to add the item to (required)")
	newItemCmd.Flags().StringVar(&newItemType, "type", "", "item type (required)")
	_ = newItemCmd.MarkFlagRequired("list")
	_ = newItemCmd.MarkFlagRequired("type")
	newCmd.AddCommand(newItemCmd)
}

func runNewItem(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	return createItem(cfg, newItemList, newItemType, parseFields(args))
}


func interactiveNewItem() error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	lists, err := scanLists(cfg.ContentDir)
	if err != nil {
		return err
	}
	if len(lists) == 0 {
		return fmt.Errorf("no lists found in %s", cfg.ContentDir)
	}

	listIdx, err := promptSelect("Select a list:", lists)
	if err != nil {
		return err
	}
	listName := lists[listIdx]

	listMeta := readListDirMeta(filepath.Join(cfg.ContentDir, listName, "list.yaml"))

	types, err := loadItemTypes(cfg.ItemsDir, listMeta.types)
	if err != nil {
		return fmt.Errorf("loading item types: %w", err)
	}
	if len(types) == 0 {
		return fmt.Errorf("no item types found")
	}

	typeNames := make([]string, len(types))
	for i, t := range types {
		typeNames[i] = t.typeName
	}
	typeIdx, err := promptSelect("Select a type:", typeNames)
	if err != nil {
		return err
	}
	it := types[typeIdx]

	data := make(map[string]string)
	for _, field := range it.Fields {
		label := field.Name
		if field.Required {
			label += " (required)"
		}
		var val string
		for {
			val, err = promptInput(label, "")
			if err != nil {
				return err
			}
			if !field.Required || strings.TrimSpace(val) != "" {
				break
			}
			fmt.Printf("%s is required\n", field.Name)
		}
		if val != "" {
			data[field.Name] = val
		}
	}

	return createItem(cfg, listName, it.typeName, data)
}

func createItem(cfg *config.SiteConfig, listName, typeName string, data map[string]string) error {
	listMeta, err := validateList(cfg.ContentDir, listName)
	if err != nil {
		return err
	}
	if err := validateType(listMeta, typeName, listName); err != nil {
		return err
	}
	it, err := resolveItemType(cfg.ItemsDir, typeName)
	if err != nil {
		return err
	}
	if err := checkRequiredFields(it, data); err != nil {
		return err
	}

	dir := filepath.Join(cfg.ContentDir, listName)
	if it.Format == "markdown" {
		return writeMarkdownItem(dir, typeName, data)
	}
	return writeYAMLItem(dir, typeName, data)
}

func writeYAMLItem(dir, typeName string, data map[string]string) error {
	item := make(map[string]any, len(data)+1)
	item["type"] = typeName
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

func scanLists(contentDir string) ([]string, error) {
	entries, err := os.ReadDir(contentDir)
	if err != nil {
		return nil, fmt.Errorf("reading content dir: %w", err)
	}
	var lists []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(contentDir, entry.Name(), "list.yaml")); err == nil {
			lists = append(lists, entry.Name())
		}
	}
	return lists, nil
}
