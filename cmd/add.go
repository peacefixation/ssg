package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/peacefixation/ssg/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	addList string
	addType string
)

var addCmd = &cobra.Command{
	Use:   "add --list <list> --type <type> [key=value ...]",
	Short: "Add a new item to a list",
	Long: `Add a new item to a list.

Fields are supplied as key=value arguments after the flags. Required fields
are defined by the item type. Missing required fields produce an error.

Example:
  ssg add --list music --type youtube url=https://youtu.be/xyz title="My Song"`,
	RunE: runAdd,
}

func init() {
	addCmd.Flags().StringVar(&addList, "list", "", "list to add the item to (required)")
	addCmd.Flags().StringVar(&addType, "type", "", "item type (required)")
	_ = addCmd.MarkFlagRequired("list")
	_ = addCmd.MarkFlagRequired("type")
	rootCmd.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	listMeta, err := validateList(cfg.ContentDir, addList)
	if err != nil {
		return err
	}

	if err := validateType(listMeta, addType, addList); err != nil {
		return err
	}

	it, err := resolveItemType(cfg.ItemsDir, addType)
	if err != nil {
		return err
	}

	data := parseFields(args)

	if err := checkRequiredFields(it, data); err != nil {
		return err
	}

	item := make(map[string]any, len(data)+1)
	item["type"] = addType
	for k, v := range data {
		item[k] = v
	}

	destPath := filepath.Join(cfg.ContentDir, addList, generateFilename(data["title"]))
	if err := writeItemFile(destPath, item); err != nil {
		return err
	}

	fmt.Printf("Created %s\n", destPath)
	return nil
}

func validateList(contentDir, listName string) (listDirMeta, error) {
	listFile := filepath.Join(contentDir, listName, "list.yaml")
	if _, err := os.Stat(listFile); err != nil {
		return listDirMeta{}, fmt.Errorf("list %q not found (expected %s)", listName, listFile)
	}
	return readListDirMeta(listFile), nil
}

func validateType(meta listDirMeta, typeName, listName string) error {
	if len(meta.types) > 0 && !slices.Contains(meta.types, typeName) {
		return fmt.Errorf("type %q is not allowed in list %q (allowed: %s)",
			typeName, listName, strings.Join(meta.types, ", "))
	}
	return nil
}

func resolveItemType(itemsDir, typeName string) (itemType, error) {
	types, err := loadItemTypes(itemsDir, []string{typeName})
	if err != nil {
		return itemType{}, fmt.Errorf("loading item types: %w", err)
	}
	if len(types) == 0 {
		return itemType{}, fmt.Errorf("item type %q not found in %s", typeName, itemsDir)
	}
	return types[0], nil
}

func checkRequiredFields(it itemType, data map[string]string) error {
	var missing []string
	for _, field := range it.Fields {
		if field.Required {
			if v, ok := data[field.Name]; !ok || strings.TrimSpace(v) == "" {
				missing = append(missing, field.Name)
			}
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required fields: %s", strings.Join(missing, ", "))
	}
	return nil
}

func writeItemFile(destPath string, item map[string]any) error {
	out, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling item: %w", err)
	}
	if err := os.WriteFile(destPath, append(out, '\n'), 0644); err != nil {
		return fmt.Errorf("writing item: %w", err)
	}
	return nil
}

// parseFields converts ["key=value", ...] args into a map.
// Values containing "=" are split on the first "=" only.
func parseFields(args []string) map[string]string {
	m := make(map[string]string, len(args))
	for _, arg := range args {
		k, v, ok := strings.Cut(arg, "=")
		if ok && k != "" {
			m[k] = v
		}
	}
	return m
}

type listDirMeta struct {
	title string
	types []string
}

func readListDirMeta(path string) listDirMeta {
	data, err := os.ReadFile(path)
	if err != nil {
		return listDirMeta{}
	}
	var raw struct {
		Title string   `yaml:"title"`
		Types []string `yaml:"types"`
	}
	_ = yaml.Unmarshal(data, &raw)
	return listDirMeta{title: raw.Title, types: raw.Types}
}

// itemType is the parsed representation of an items/{type}.yaml file.
type itemType struct {
	typeName string
	Name     string      `yaml:"name"`
	Fields   []typeField `yaml:"fields"`
}

type typeField struct {
	Name     string `yaml:"name"`
	Required bool   `yaml:"required"`
}

// loadItemTypes reads all *.yaml files from itemsDir, filtered to allowedTypes if non-empty.
func loadItemTypes(itemsDir string, allowedTypes []string) ([]itemType, error) {
	entries, err := os.ReadDir(itemsDir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	allowed := make(map[string]bool, len(allowedTypes))
	for _, t := range allowedTypes {
		allowed[t] = true
	}

	var types []itemType
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		typeName := strings.TrimSuffix(entry.Name(), ".yaml")
		if len(allowed) > 0 && !allowed[typeName] {
			continue
		}
		data, err := os.ReadFile(filepath.Join(itemsDir, entry.Name()))
		if err != nil {
			return nil, err
		}
		var t itemType
		if err := yaml.Unmarshal(data, &t); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", entry.Name(), err)
		}
		t.typeName = typeName
		types = append(types, t)
	}
	return types, nil
}

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// generateFilename produces a timestamped slug filename, e.g. 20260425T120000Z-my-title.json.
func generateFilename(title string) string {
	ts := time.Now().UTC().Format("20060102T150405Z")
	slug := nonAlnum.ReplaceAllString(strings.ToLower(title), "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return ts + ".json"
	}
	return ts + "-" + slug + ".json"
}
