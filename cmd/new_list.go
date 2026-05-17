package cmd

import (
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
	newListTitle        string
	newListTypes        string
	newListTemplate     string
	newListCardTemplate string
	newListSortBy       string
	newListSortOrder    string
	newListLimit        int
)

var newListCmd = &cobra.Command{
	Use:   "list <name> --title <title> [flags]",
	Short: "Create a new list",
	Long: `Create a new list directory with a list.yaml.

Example:
  ssg new list music --title "Music" --types soundcloud,youtube
  ssg new list music/live --title "Live Sets" --sort-by date`,
	Args: cobra.ExactArgs(1),
	RunE: runNewList,
}

func init() {
	newListCmd.Flags().StringVar(&newListTitle, "title", "", "list title (required)")
	newListCmd.Flags().StringVar(&newListTypes, "types", "", "comma-separated item type allowlist")
	newListCmd.Flags().StringVar(&newListTemplate, "template", "", "override list page template")
	newListCmd.Flags().StringVar(&newListCardTemplate, "card-template", "", "override child card template")
	newListCmd.Flags().StringVar(&newListSortBy, "sort-by", "", "field to sort children by")
	newListCmd.Flags().StringVar(&newListSortOrder, "sort-order", "", "sort order: asc or desc")
	newListCmd.Flags().IntVar(&newListLimit, "limit", 0, "max children to render (0 = unlimited)")
	_ = newListCmd.MarkFlagRequired("title")
	newCmd.AddCommand(newListCmd)
}

func runNewList(cmd *cobra.Command, args []string) error {
	if newListSortOrder != "" && newListSortOrder != "asc" && newListSortOrder != "desc" {
		return fmt.Errorf("--sort-order must be asc or desc")
	}
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	lc := newListConfig{
		Title:        newListTitle,
		Template:     newListTemplate,
		CardTemplate: newListCardTemplate,
		SortBy:       newListSortBy,
		SortOrder:    newListSortOrder,
		Limit:        newListLimit,
	}
	if newListTypes != "" {
		for _, t := range strings.Split(newListTypes, ",") {
			if s := strings.TrimSpace(t); s != "" {
				lc.Types = append(lc.Types, s)
			}
		}
	}
	return createList(cfg, args[0], lc)
}

func interactiveNewList() error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	var name string
	for {
		name, err = promptInput("List path (relative to content dir)", "")
		if err != nil {
			return err
		}
		if strings.TrimSpace(name) != "" {
			break
		}
		fmt.Println("Path is required")
	}

	var title string
	for {
		title, err = promptInput("Title", "")
		if err != nil {
			return err
		}
		if strings.TrimSpace(title) != "" {
			break
		}
		fmt.Println("Title is required")
	}

	typesRaw, err := promptInput("Types (comma-separated, blank to skip)", "")
	if err != nil {
		return err
	}
	lc := newListConfig{Title: title}
	if typesRaw != "" {
		for _, t := range strings.Split(typesRaw, ",") {
			if s := strings.TrimSpace(t); s != "" {
				lc.Types = append(lc.Types, s)
			}
		}
	}

	return createList(cfg, name, lc)
}

func createList(cfg *config.SiteConfig, name string, lc newListConfig) error {
	destDir := filepath.Join(cfg.ContentDir, name)

	if _, err := os.Stat(destDir); err == nil {
		return fmt.Errorf("directory %s already exists", destDir)
	}

	parentDir := filepath.Dir(destDir)
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		return fmt.Errorf("parent directory %s does not exist", parentDir)
	}

	if err := os.Mkdir(destDir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	out, err := yaml.Marshal(lc)
	if err != nil {
		return fmt.Errorf("marshalling list config: %w", err)
	}

	listFile := filepath.Join(destDir, "list.yaml")
	if err := os.WriteFile(listFile, out, 0644); err != nil {
		return fmt.Errorf("writing list.yaml: %w", err)
	}

	fmt.Printf("Created %s\n", listFile)
	return nil
}

type newListConfig struct {
	Title        string   `yaml:"title"`
	Types        []string `yaml:"types,omitempty"`
	Template     string   `yaml:"template,omitempty"`
	CardTemplate string   `yaml:"cardTemplate,omitempty"`
	SortBy       string   `yaml:"sortBy,omitempty"`
	SortOrder    string   `yaml:"sortOrder,omitempty"`
	Limit        int      `yaml:"limit,omitempty"`
}

// --- shared helpers (moved from add.go) ---

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

type itemType struct {
	typeName string
	Name     string      `yaml:"name"`
	Format   string      `yaml:"format"` // "markdown" or "" (default: yaml)
	Fields   []typeField `yaml:"fields"`
}

type typeField struct {
	Name     string `yaml:"name"`
	Required bool   `yaml:"required"`
}

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

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func generateFilename(title, ext string) string {
	ts := time.Now().UTC().Format("20060102T150405Z")
	slug := nonAlnum.ReplaceAllString(strings.ToLower(title), "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return ts + ext
	}
	return ts + "-" + slug + ext
}
