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

func createList(cfg *config.SiteConfig, name string, lc newListConfig) error {
	destDir := filepath.Join(cfg.ContentDir, name)

	if _, err := os.Stat(destDir); err == nil {
		return fmt.Errorf("directory %s already exists", destDir)
	}

	parentDir := filepath.Dir(destDir)
	listName := filepath.Base(destDir)

	// If the parent directory does not contain a list.yaml it is not a standard
	// list directory. Check whether it is a file item's sibling container instead.
	// The root content directory has no list.yaml by convention and is always valid.
	parentIsRoot := filepath.Clean(parentDir) == filepath.Clean(cfg.ContentDir)
	if !parentIsRoot {
		if _, err := os.Stat(filepath.Join(parentDir, "list.yaml")); os.IsNotExist(err) {
			grandparentDir := filepath.Dir(parentDir)
			stem := filepath.Base(parentDir)
			itemPath, found := findFileItemByStem(grandparentDir, stem)
			if !found {
				if _, statErr := os.Stat(parentDir); os.IsNotExist(statErr) {
					return fmt.Errorf("parent directory %s does not exist", parentDir)
				}
				return fmt.Errorf("parent %q is not a list and no matching file item found", parentDir)
			}
			return createFileItemSubList(destDir, parentDir, listName, itemPath, lc)
		}
	}

	if parent, err := readListConfig(filepath.Join(parentDir, "list.yaml")); err == nil {
		lc = mergeParentConfig(lc, parent)
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

func createFileItemSubList(destDir, siblingDir, listName, itemPath string, lc newListConfig) error {
	// Create the sibling container directory if this is the first sub-list.
	if err := os.MkdirAll(siblingDir, 0755); err != nil {
		return fmt.Errorf("creating sibling directory: %w", err)
	}
	if err := os.Mkdir(destDir, 0755); err != nil {
		return fmt.Errorf("creating sub-list directory: %w", err)
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
	if err := appendListToFile(itemPath, listName); err != nil {
		return fmt.Errorf("updating %s: %w", itemPath, err)
	}
	fmt.Printf("Updated %s\n", itemPath)
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

func readListConfig(path string) (newListConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return newListConfig{}, err
	}
	var lc newListConfig
	if err := yaml.Unmarshal(data, &lc); err != nil {
		return newListConfig{}, err
	}
	return lc, nil
}

func mergeParentConfig(lc, parent newListConfig) newListConfig {
	if lc.CardTemplate == "" {
		lc.CardTemplate = parent.CardTemplate
	}
	if lc.Template == "" {
		lc.Template = parent.Template
	}
	if lc.SortBy == "" {
		lc.SortBy = parent.SortBy
	}
	if lc.SortOrder == "" {
		lc.SortOrder = parent.SortOrder
	}
	if lc.Limit == 0 {
		lc.Limit = parent.Limit
	}
	if lc.Types == nil {
		lc.Types = parent.Types
	}
	return lc
}

// --- list discovery ---

// fileItemMeta holds the fields from a content file that are relevant to list discovery.
type fileItemMeta struct {
	Title string   `yaml:"title" json:"title"`
	Lists []string `yaml:"lists" json:"lists"`
}

// readFileItemMeta reads title and lists from a content file (YAML, JSON, or Markdown).
func readFileItemMeta(path string) fileItemMeta {
	data, err := os.ReadFile(path)
	if err != nil {
		return fileItemMeta{}
	}
	var m fileItemMeta
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		_ = json.Unmarshal(data, &m)
	case ".md", ".markdown":
		if fm := extractFrontmatter(data); fm != nil {
			_ = yaml.Unmarshal(fm, &m)
		}
	default:
		_ = yaml.Unmarshal(data, &m)
	}
	return m
}

// extractFrontmatter returns the YAML block between leading --- delimiters, or nil.
func extractFrontmatter(data []byte) []byte {
	s := string(data)
	if !strings.HasPrefix(s, "---") {
		return nil
	}
	rest := s[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return nil
	}
	return []byte(rest[:idx])
}

// stemFromPath returns the filename stem of path (base name without extension).
func stemFromPath(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// findFileItemByStem looks in dir for a content file whose stem equals stem.
// Returns the full path and true if found.
func findFileItemByStem(dir, stem string) (string, bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", false
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "list.yaml" {
			continue
		}
		if stemFromPath(entry.Name()) == stem && scanListExts[strings.ToLower(filepath.Ext(entry.Name()))] {
			return filepath.Join(dir, entry.Name()), true
		}
	}
	return "", false
}

// appendListToFile appends listName to the "lists" field of a content file in-place.
func appendListToFile(path, listName string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return appendListJSON(path, data, listName)
	case ".md", ".markdown":
		return appendListMarkdown(path, data, listName)
	default:
		return appendListYAML(path, data, listName)
	}
}

func appendListYAML(path string, data []byte, listName string) error {
	var m map[string]any
	if err := yaml.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}
	if m == nil {
		m = make(map[string]any)
	}
	m["lists"] = append(toStringSlice(m["lists"]), listName)
	out, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshalling %s: %w", path, err)
	}
	return os.WriteFile(path, out, 0644)
}

func appendListJSON(path string, data []byte, listName string) error {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}
	m["lists"] = append(toStringSlice(m["lists"]), listName)
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling %s: %w", path, err)
	}
	return os.WriteFile(path, append(out, '\n'), 0644)
}

func appendListMarkdown(path string, data []byte, listName string) error {
	fm := extractFrontmatter(data)
	s := string(data)
	if fm == nil {
		// No frontmatter — prepend a new block.
		newFM, err := yaml.Marshal(map[string]any{"lists": []string{listName}})
		if err != nil {
			return err
		}
		return os.WriteFile(path, []byte("---\n"+string(newFM)+"---\n\n"+s), 0644)
	}
	var fmMap map[string]any
	if err := yaml.Unmarshal(fm, &fmMap); err != nil {
		return fmt.Errorf("parsing frontmatter in %s: %w", path, err)
	}
	if fmMap == nil {
		fmMap = make(map[string]any)
	}
	fmMap["lists"] = append(toStringSlice(fmMap["lists"]), listName)
	newFM, err := yaml.Marshal(fmMap)
	if err != nil {
		return fmt.Errorf("marshalling frontmatter: %w", err)
	}
	// Reconstruct: "---\n{fm}---{rest-after-closing-delimiter}"
	rest := s[3:] // skip opening ---
	idx := strings.Index(rest, "\n---")
	afterFM := rest[idx+4:] // everything after the closing ---
	return os.WriteFile(path, []byte("---\n"+string(newFM)+"---"+afterFM), 0644)
}

// toStringSlice coerces the value stored under "lists" in a parsed map to []string.
func toStringSlice(v any) []string {
	switch val := v.(type) {
	case []string:
		return val
	case []any:
		out := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

// --- shared helpers ---

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
