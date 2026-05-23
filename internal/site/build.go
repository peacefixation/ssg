package site

import (
	"fmt"
	"html/template"
	"log"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/peacefixation/ssg/internal/config"
	"github.com/peacefixation/ssg/internal/datasource"
	"github.com/peacefixation/ssg/internal/enricher"
	"github.com/peacefixation/ssg/internal/renderer"
	"github.com/peacefixation/ssg/internal/theme"
	"gopkg.in/yaml.v3"
)

var contentExts = map[string]bool{
	".md": true, ".markdown": true, ".json": true, ".yaml": true, ".yml": true,
}

// listMeta is the in-memory representation of a list.yaml file.
// It carries browser extension metadata (title, fields) plus optional build
// overrides (template, cardTemplate, sortBy, sortOrder, limit).
type listMeta struct {
	Title        string `yaml:"title"`
	Template     string `yaml:"template"`
	CardTemplate string `yaml:"cardTemplate"`
	SortBy       string `yaml:"sortBy"`
	SortOrder    string `yaml:"sortOrder"`
	Limit        int    `yaml:"limit"`
}

// Build runs the full build pipeline for cfg, writing pages to cfg.OutputDir.
// It returns the total number of pages written across all items.
func Build(cfg *config.SiteConfig, registry *datasource.Registry, clean bool) (int, error) {
	if clean {
		if err := os.RemoveAll(cfg.OutputDir); err != nil {
			return 0, fmt.Errorf("cleaning output dir: %w", err)
		}
	}

	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return 0, fmt.Errorf("creating output dir: %w", err)
	}

	themeData, themeTemplateDir, err := loadTheme(cfg)
	if err != nil {
		return 0, err
	}
	if cfg.Theme != "" {
		if err := theme.CopyAssets(filepath.Join(cfg.ThemesDir, cfg.Theme), cfg.OutputDir); err != nil {
			return 0, fmt.Errorf("copying theme assets: %w", err)
		}
	}

	r, err := renderer.New(cfg.TemplateDir, themeTemplateDir)
	if err != nil {
		return 0, fmt.Errorf("initializing renderer: %w", err)
	}

	rootItems, err := scanDir(cfg.ContentDir, "", cfg, listMeta{})
	if err != nil {
		return 0, err
	}

	if cfg.Tags.Enabled {
		tagMap := collectTags(rootItems, registry, nil)
		tagsItem := buildTagsTree(tagMap, cfg, registry)
		rootItems = append(rootItems, tagsItem)
	}

	// Pre-fetch nav data for all root items so every page can render global nav.
	// The home page (index.html) is excluded — the site title serves as the home link.
	allNavItems := buildNavItems(rootItems, registry)
	rootNavItems := make([]map[string]any, 0, len(allNavItems))
	for _, item := range allNavItems {
		if item["outputPath"] != "index.html" {
			rootNavItems = append(rootNavItems, item)
		}
	}

	var siteMap []config.SiteMapNode
	if cfg.SiteMap {
		siteMap = buildSiteMap(rootItems, registry, cfg.ItemsDir)
	}

	var ogEnricher *enricher.OGEnricher
	if cfg.OGCacheFile != "" {
		referer := cfg.CanonicalURL
		if referer == "" {
			referer = cfg.BaseURL
		}
		ogEnricher = enricher.New(cfg.OGCacheFile, referer)
		if err := ogEnricher.LoadCache(); err != nil {
			log.Printf("warning: loading OG cache: %v", err)
		}
		defer func() {
			if err := ogEnricher.SaveCache(); err != nil {
				log.Printf("warning: saving OG cache: %v", err)
			}
		}()
	}

	count := 0
	for _, itemCfg := range rootItems {
		n, err := buildItem(cfg, itemCfg, registry, r, rootNavItems, themeData, []map[string]any{}, siteMap, ogEnricher)
		if err != nil {
			return count, fmt.Errorf("building item %q: %w", itemCfg.Name, err)
		}
		count += n
	}

	return count, nil
}

// loadTheme reads the theme config (if a theme is set) and returns the theme
// data to inject into templates and the path to the theme's template partials.
func loadTheme(cfg *config.SiteConfig) (theme.Data, string, error) {
	if cfg.Theme == "" {
		return theme.Data{}, "", nil
	}
	themeDir := filepath.Join(cfg.ThemesDir, cfg.Theme)
	themeCfg, err := theme.Load(themeDir)
	if err != nil {
		return theme.Data{}, "", fmt.Errorf("loading theme %q: %w", cfg.Theme, err)
	}
	return theme.BuildData(themeCfg), theme.TemplateDir(themeDir), nil
}

// buildSiteMap recursively builds the full site map tree from scanned items,
// skipping the homepage. Titles are fetched from each item's datasource.
func buildSiteMap(items []config.ItemConfig, registry *datasource.Registry, itemsDir string) []config.SiteMapNode {
	nodes := make([]config.SiteMapNode, 0, len(items))
	for _, itemCfg := range items {
		if itemCfg.OutputPath == "index.html" {
			continue
		}
		if itemCfg.ExcludeFromSiteMap {
			continue
		}
		title := itemCfg.Name
		var externalURL, icon string
		if ds, err := registry.New(itemCfg.DataSource); err == nil {
			if data, err := ds.FetchOne(); err == nil {
				if typeName, ok := data["type"].(string); ok {
					if defaults := loadItemTypeDefaults(itemsDir, typeName); defaults != nil {
						applyTypeDefaults(data, defaults)
					}
				}
				if t, ok := data["title"].(string); ok && t != "" {
					title = t
				}
				if tmpl, ok := data["template"].(string); ok && tmpl == "sitemap.html" {
					continue
				}
				if u, ok := data["url"].(string); ok {
					externalURL = u
				}
				if ic, ok := data["icon"].(string); ok {
					icon = ic
				}
			}
		}
		if icon == "" {
			ext := strings.ToLower(filepath.Ext(itemCfg.DataSource.Path))
			if ext == ".md" || ext == ".markdown" {
				icon = "post"
			}
		}
		children := buildSiteMap(itemCfg.Children, registry, itemsDir)
		if icon == "" && len(children) > 0 {
			icon = "list"
		}
		nodes = append(nodes, config.SiteMapNode{
			Title:      title,
			OutputPath: itemCfg.OutputPath,
			URL:        externalURL,
			Icon:       icon,
			Children:   children,
		})
	}
	return nodes
}

// buildNavItems fetches the data for each item and returns lightweight nav
// records (title, outputPath, count) for injection into every page template.
func buildNavItems(items []config.ItemConfig, registry *datasource.Registry) []map[string]any {
	nav := make([]map[string]any, 0, len(items))
	for _, itemCfg := range items {
		ds, err := registry.New(itemCfg.DataSource)
		if err != nil {
			continue
		}
		data, err := ds.FetchOne()
		if err != nil {
			continue
		}
		data["outputPath"] = itemCfg.OutputPath
		data["name"] = itemCfg.Name
		data["count"] = len(itemCfg.Children)
		nav = append(nav, data)
	}
	return nav
}

// scanDir recursively walks dir and returns an ItemConfig for every discovered item:
//   - Files with a supported extension become page items.
//   - Subdirectories containing a list.yaml become directory items whose
//     Children are the result of recursively scanning that subdirectory.
func scanDir(dir, outputPrefix string, cfg *config.SiteConfig, parent listMeta) ([]config.ItemConfig, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading directory %q: %w", dir, err)
	}

	var items []config.ItemConfig
	for _, entry := range entries {
		if entry.IsDir() {
			item, ok, err := scanDirItem(dir, entry.Name(), outputPrefix, cfg, parent)
			if err != nil {
				return nil, err
			}
			if ok {
				items = append(items, item)
			}
		} else {
			item, ok := scanFileItem(dir, entry.Name(), outputPrefix, cfg)
			if ok {
				items = append(items, item)
			}
		}
	}
	return items, nil
}

// scanDirItem checks whether name is a directory item (contains list.yaml).
// Returns ok=false for directories without list.yaml (they are ignored).
func scanDirItem(parentDir, name, outputPrefix string, cfg *config.SiteConfig, parent listMeta) (config.ItemConfig, bool, error) {
	dir := filepath.Join(parentDir, name)
	listFile := filepath.Join(dir, "list.yaml")
	if _, err := os.Stat(listFile); err != nil {
		return config.ItemConfig{}, false, nil
	}

	meta := readListMeta(listFile)

	// Resolve each setting: list.yaml → parent list → site.yaml defaults.
	tmpl := first(meta.Template, parent.Template, cfg.Defaults.List.Template)
	cardTemplate := first(meta.CardTemplate, parent.CardTemplate, cfg.Defaults.List.CardTemplate)
	sortBy := first(meta.SortBy, parent.SortBy, cfg.Defaults.List.SortBy)
	sortOrder := first(meta.SortOrder, parent.SortOrder, cfg.Defaults.List.SortOrder)
	limit := meta.Limit
	if limit == 0 {
		limit = parent.Limit
	}
	if limit == 0 {
		limit = cfg.Defaults.List.Limit
	}

	resolved := listMeta{
		Template:     tmpl,
		CardTemplate: cardTemplate,
		SortBy:       sortBy,
		SortOrder:    sortOrder,
		Limit:        limit,
	}

	children, err := scanDir(dir, outputPrefix+name+"/", cfg, resolved)
	if err != nil {
		return config.ItemConfig{}, false, err
	}

	return config.ItemConfig{
		Name:         name,
		Template:     tmpl,
		CardTemplate: cardTemplate,
		OutputPath:   outputPrefix + name + "/index.html",
		DataSource:   config.DataSourceConfig{Type: config.FileType, Path: listFile},
		Children:     children,
		SortBy:       sortBy,
		SortOrder:    sortOrder,
		Limit:        limit,
	}, true, nil
}

// scanFileItem checks whether name is a supported content file.
// Returns ok=false for files with unsupported extensions and for list.yaml.
func scanFileItem(parentDir, name, outputPrefix string, cfg *config.SiteConfig) (config.ItemConfig, bool) {
	if name == "list.yaml" {
		return config.ItemConfig{}, false
	}
	ext := strings.ToLower(filepath.Ext(name))
	if !contentExts[ext] {
		return config.ItemConfig{}, false
	}

	stem := strings.TrimSuffix(name, filepath.Ext(name))
	outputPath := outputPrefix + stem + "/index.html"
	if outputPrefix == "" && stem == "index" {
		outputPath = "index.html"
	}

	return config.ItemConfig{
		Name:         stem,
		Template:     cfg.Defaults.Page.Template,
		CardTemplate: cfg.Defaults.List.CardTemplate,
		OutputPath:   outputPath,
		DataSource:   config.DataSourceConfig{Type: config.FileType, Path: filepath.Join(parentDir, name)},
	}, true
}

// buildItem fetches this item's data, recursively builds all child pages,
// assembles child card fragments into List, injects standard template keys,
// and writes the output page. Returns the total number of pages written
// (this item plus all descendants).
func buildItem(
	cfg *config.SiteConfig,
	itemCfg config.ItemConfig,
	registry *datasource.Registry,
	r *renderer.Renderer,
	rootNavItems []map[string]any,
	themeData theme.Data,
	ancestors []map[string]any,
	siteMap []config.SiteMapNode,
	ogEnricher *enricher.OGEnricher,
) (int, error) {
	ds, err := getDS(itemCfg, registry)
	if err != nil {
		return 0, fmt.Errorf("creating datasource: %w", err)
	}

	item, err := NewItem(itemCfg, ds)
	if err != nil {
		return 0, err
	}

	if typeName, ok := item.Data["type"].(string); ok && typeName != "" {
		if defaults := loadItemTypeDefaults(cfg.ItemsDir, typeName); defaults != nil {
			applyTypeDefaults(item.Data, defaults)
		}
	}

	if _, ok := item.Data["icon"]; !ok {
		if strings.HasSuffix(itemCfg.DataSource.Path, "list.yaml") {
			item.Data["icon"] = "list"
		} else {
			ext := strings.ToLower(filepath.Ext(itemCfg.DataSource.Path))
			if ext == ".md" || ext == ".markdown" {
				item.Data["icon"] = "post"
			}
		}
	}

	if enrichType, _ := item.Data["enrich"].(string); enrichType == "opengraph" && ogEnricher != nil {
		if url, _ := item.Data["url"].(string); url != "" {
			force := cfg.RefreshOG || forceRefreshItem(item.Data)
			ogData, err := ogEnricher.Enrich(url, force)
			if err != nil {
				log.Printf("warning: OG enrichment failed for %s: %v", url, err)
			} else {
				maps.Copy(item.Data, ogData)
			}
		}
		delete(item.Data, "enrich")
	}

	if isDraft(item.Data) && !cfg.Drafts {
		return 0, nil
	}

	// Allow frontmatter / list.yaml to override the page template.
	if tmpl, ok := item.Data["template"].(string); ok && tmpl != "" {
		item.Config.Template = tmpl
	}

	// If the item declares sub-lists, scan each sibling directory and append the
	// resulting ItemConfigs to Children before buildChildren is called.
	if rawLists, ok := item.Data["lists"].([]any); ok {
		stem := stemOf(itemCfg.DataSource.Path)
		siblingDir := filepath.Join(filepath.Dir(itemCfg.DataSource.Path), stem)
		outputPrefix := strings.TrimSuffix(itemCfg.OutputPath, "index.html")
		for _, raw := range rawLists {
			name, _ := raw.(string)
			if name == "" {
				continue
			}
			sub, ok, err := scanDirItem(siblingDir, name, outputPrefix, cfg, listMeta{
					CardTemplate: itemCfg.CardTemplate,
					SortBy:       itemCfg.SortBy,
					SortOrder:    itemCfg.SortOrder,
					Limit:        itemCfg.Limit,
				})
			if err != nil {
				return 0, fmt.Errorf("scanning sub-list %q of %q: %w", name, itemCfg.Name, err)
			}
			if ok {
				itemCfg.Children = append(itemCfg.Children, sub)
			}
		}
		delete(item.Data, "lists")
	}

	// Build the ancestors slice for children: ancestors + this item.
	title, _ := item.Data["title"].(string)
	childAncestors := make([]map[string]any, len(ancestors)+1)
	copy(childAncestors, ancestors)
	childAncestors[len(ancestors)] = map[string]any{"title": title, "outputPath": item.OutputPath}

	// Recursively build child pages and collect their card fragments.
	fragments, childCount, err := buildChildren(cfg, itemCfg, registry, r, rootNavItems, themeData, childAncestors, siteMap, ogEnricher)
	if err != nil {
		return 0, err
	}

	item.Data["Site"] = cfg
	item.Data["OutputPath"] = item.OutputPath
	item.Data["RootItems"] = rootNavItems
	item.Data["List"] = fragments
	item.Data["Theme"] = themeData
	item.Data["BreadcrumbLinks"] = ancestors
	item.Data["BreadcrumbCurrent"] = title
	item.Data["SiteMap"] = siteMap
	item.Data["PageTemplate"] = item.Config.Template

	if err := writeItem(cfg.OutputDir, item, r); err != nil {
		return 0, err
	}

	return childCount + 1, nil
}

// childEntry pairs an ItemConfig with the fetched data needed for sorting and
// card rendering.
type childEntry struct {
	cfg  config.ItemConfig
	data map[string]any
}

// buildChildren builds each child's page recursively, then fetches child data,
// sorts, applies limit, and renders card fragments for the parent's List.
func buildChildren(
	cfg *config.SiteConfig,
	itemCfg config.ItemConfig,
	registry *datasource.Registry,
	r *renderer.Renderer,
	rootNavItems []map[string]any,
	themeData theme.Data,
	ancestors []map[string]any,
	siteMap []config.SiteMapNode,
	ogEnricher *enricher.OGEnricher,
) ([]template.HTML, int, error) {
	if len(itemCfg.Children) == 0 {
		return nil, 0, nil
	}

	// Build every child page first (regardless of limit, so all pages exist).
	totalCount := 0
	for _, childCfg := range itemCfg.Children {
		n, err := buildItem(cfg, childCfg, registry, r, rootNavItems, themeData, ancestors, siteMap, ogEnricher)
		if err != nil {
			return nil, 0, fmt.Errorf("building child %q: %w", childCfg.Name, err)
		}
		totalCount += n
	}

	// Fetch child data for sorting and card rendering.
	entries := make([]childEntry, 0, len(itemCfg.Children))
	for _, childCfg := range itemCfg.Children {
		ds, err := getDS(childCfg, registry)
		if err != nil {
			return nil, 0, fmt.Errorf("creating datasource for child %q: %w", childCfg.Name, err)
		}
		data, err := ds.FetchOne()
		if err != nil {
			return nil, 0, fmt.Errorf("fetching data for child %q: %w", childCfg.Name, err)
		}
		if typeName, ok := data["type"].(string); ok && typeName != "" {
			if defaults := loadItemTypeDefaults(cfg.ItemsDir, typeName); defaults != nil {
				applyTypeDefaults(data, defaults)
			}
		}
		if _, ok := data["icon"]; !ok {
			if strings.HasSuffix(childCfg.DataSource.Path, "list.yaml") {
				data["icon"] = "list"
			} else {
				ext := strings.ToLower(filepath.Ext(childCfg.DataSource.Path))
				if ext == ".md" || ext == ".markdown" {
					data["icon"] = "post"
				}
			}
		}
		if enrichType, _ := data["enrich"].(string); enrichType == "opengraph" && ogEnricher != nil {
			if url, _ := data["url"].(string); url != "" {
				ogData, err := ogEnricher.Enrich(url, cfg.RefreshOG || forceRefreshItem(data))
				if err != nil {
					log.Printf("warning: OG enrichment failed for %s: %v", url, err)
				} else {
					maps.Copy(data, ogData)
				}
			}
			delete(data, "enrich")
		}
		if isDraft(data) && !cfg.Drafts {
			continue
		}
		data["outputPath"] = childCfg.OutputPath
		data["count"] = len(childCfg.Children)
		entries = append(entries, childEntry{cfg: childCfg, data: data})
	}

	// Sort and apply limit.
	if itemCfg.SortBy != "" {
		sortChildEntries(entries, itemCfg.SortBy, itemCfg.SortOrder)
	}
	if itemCfg.Limit > 0 && len(entries) > itemCfg.Limit {
		entries = entries[:itemCfg.Limit]
	}

	// Render a card fragment for each displayed child.
	fragments := make([]template.HTML, 0, len(entries))
	for _, e := range entries {
		// Parent list's cardTemplate is the default; child data can override.
		cardTemplate := itemCfg.CardTemplate
		if t, ok := e.data["cardTemplate"].(string); ok && t != "" {
			cardTemplate = t
		}
		fragment, err := r.RenderCard(cardTemplate, e.data)
		if err != nil {
			return nil, 0, fmt.Errorf("rendering card for %q: %w", e.cfg.Name, err)
		}
		fragments = append(fragments, fragment)
	}

	return fragments, totalCount, nil
}

// sortChildEntries sorts entries in-place by the given field in the item data.
// time.Time values are compared chronologically; all other values as strings.
func sortChildEntries(entries []childEntry, field, order string) {
	sort.SliceStable(entries, func(i, j int) bool {
		ai := entries[i].data[field]
		bi := entries[j].data[field]

		at, aIsTime := ai.(time.Time)
		bt, bIsTime := bi.(time.Time)
		if aIsTime && bIsTime {
			if order == "desc" {
				return at.After(bt)
			}
			return at.Before(bt)
		}

		a, _ := ai.(string)
		b, _ := bi.(string)
		if order == "desc" {
			return a > b
		}
		return a < b
	})
}

// readListMeta reads build and browser-extension metadata from a list.yaml file.
func readListMeta(path string) listMeta {
	data, err := os.ReadFile(path)
	if err != nil {
		return listMeta{}
	}
	var m listMeta
	_ = yaml.Unmarshal(data, &m)
	return m
}

// itemTypeConfig is the in-memory representation of an items/{type}.yaml file.
type itemTypeConfig struct {
	Name     string         `yaml:"name"`
	Defaults map[string]any `yaml:"defaults"`
}

// loadItemTypeDefaults reads items/{typeName}.yaml and returns its defaults map.
// Returns nil if the file does not exist or cannot be parsed.
func loadItemTypeDefaults(itemsDir, typeName string) map[string]any {
	data, err := os.ReadFile(filepath.Join(itemsDir, typeName+".yaml"))
	if err != nil {
		return nil
	}
	var cfg itemTypeConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil
	}
	return cfg.Defaults
}

// applyTypeDefaults merges type defaults into data, skipping keys already set.
func applyTypeDefaults(data map[string]any, defaults map[string]any) {
	for k, v := range defaults {
		if _, exists := data[k]; !exists {
			data[k] = v
		}
	}
}

// excludeSelf returns rootNavItems without the entry whose outputPath matches currentPath.
// func excludeSelf(items []map[string]any, currentPath string) []map[string]any {
// 	result := make([]map[string]any, 0, len(items))
// 	for _, item := range items {
// 		if item["outputPath"] != currentPath {
// 			result = append(result, item)
// 		}
// 	}
// 	return result
// }

// stemOf returns the filename stem of path (base name without extension).
func stemOf(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// isDraft reports whether item data contains draft: true.
func isDraft(data map[string]any) bool {
	b, ok := data["draft"].(bool)
	return ok && b
}

// forceRefreshItem reports whether item data contains og_refresh: true.
func forceRefreshItem(data map[string]any) bool {
	b, ok := data["og_refresh"].(bool)
	return ok && b
}

// first returns the first non-empty string from the arguments.
func first(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// getDS returns the datasource for itemCfg. If DataSourceOverride is set it
// takes precedence over the registry, avoiding an import cycle by asserting
// the stored value to datasource.DataSource.
func getDS(itemCfg config.ItemConfig, registry *datasource.Registry) (datasource.DataSource, error) {
	if itemCfg.DataSourceOverride != nil {
		if ds, ok := itemCfg.DataSourceOverride.(datasource.DataSource); ok {
			return ds, nil
		}
	}
	return registry.New(itemCfg.DataSource)
}

func writeItem(outputDir string, item *Item, r *renderer.Renderer) error {
	outPath := filepath.Join(outputDir, filepath.FromSlash(item.OutputPath))

	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return fmt.Errorf("creating output directory for %s: %w", outPath, err)
	}

	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("creating output file %s: %w", outPath, err)
	}
	defer f.Close()

	return r.RenderItem(f, item.Config.Template, item.Data)
}
