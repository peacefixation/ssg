package site

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
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
	Type         string `yaml:"type"` // "photos" triggers image-file scanning
	Template     string `yaml:"template"`
	CardTemplate string `yaml:"cardTemplate"`
	SortBy       string `yaml:"sortBy"`
	SortOrder    string `yaml:"sortOrder"`
	Limit        int    `yaml:"limit"`
}

// imageExts is the set of file extensions treated as images in a photos list.
var imageExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true,
	".gif": true, ".webp": true, ".avif": true,
}

// Builder holds the invariant context shared across the recursive build and is
// the receiver for buildItem, buildChildren, and renderChildCards.
type Builder struct {
	cfg          *config.SiteConfig
	registry     *datasource.Registry
	renderer     *renderer.Renderer
	rootNavItems []map[string]any
	themeData    theme.Data
	siteMap      []config.SiteMapNode
	ogEnricher   *enricher.OGEnricher
	ytEnricher   *enricher.YouTubeEnricher
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

	if cfg.StaticDir != "" {
		if err := copyStaticDir(cfg.StaticDir, cfg.OutputDir); err != nil {
			return 0, fmt.Errorf("copying static assets: %w", err)
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

	var ytEnricher *enricher.YouTubeEnricher
	if cfg.YouTubeCacheFile != "" && cfg.YouTubeAPIKey != "" {
		ytEnricher = enricher.NewYouTube(cfg.YouTubeCacheFile, cfg.YouTubeAPIKey)
		if err := ytEnricher.LoadCache(); err != nil {
			log.Printf("warning: loading YouTube cache: %v", err)
		}
		defer func() {
			if err := ytEnricher.SaveCache(); err != nil {
				log.Printf("warning: saving YouTube cache: %v", err)
			}
		}()
	}

	b := &Builder{
		cfg:          cfg,
		registry:     registry,
		renderer:     r,
		rootNavItems: rootNavItems,
		themeData:    themeData,
		siteMap:      siteMap,
		ogEnricher:   ogEnricher,
		ytEnricher:   ytEnricher,
	}

	count := 0
	for _, itemCfg := range rootItems {
		n, _, err := b.buildItem(itemCfg, []map[string]any{})
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

	if parent.Type == "photos" {
		// Photos directories are handled exclusively here. The regular file loop
		// is skipped so sidecar .yaml files are not registered as standalone items.
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if !imageExts[ext] {
				continue
			}
			stem := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
			baseData := map[string]any{
				"type":     "photo",
				"title":    stem,
				"filename": entry.Name(),
				"src":      "/" + outputPrefix + entry.Name(),
			}
			if sidecar := readSidecar(filepath.Join(dir, stem+".yaml")); sidecar != nil {
				maps.Copy(baseData, sidecar)
			}
			items = append(items, config.ItemConfig{
				Name:         stem,
				Template:     cfg.Defaults.Page.Template,
				CardTemplate: parent.CardTemplate,
				OutputPath:   outputPrefix + stem + "/index.html",
				DataSource: config.DataSourceConfig{
					Type: config.MapType,
					Data: baseData,
				},
			})
		}
	} else {
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
		Type:         meta.Type,
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
		ListType:     meta.Type,
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

// buildItem is the orchestrator for building a single page: prepare, enrich,
// build children, snapshot card data, inject template vars, write output.
// Returns the total page count (this item plus all descendants) and the
// pre-injection data snapshot used for card rendering by the parent.
func (b *Builder) buildItem(itemCfg config.ItemConfig, ancestors []map[string]any) (int, map[string]any, error) {
	item, err := b.prepareItem(itemCfg)
	if err != nil {
		return 0, nil, err
	}

	b.enrich(item)

	if isDraft(item.Data) && !b.cfg.Drafts {
		return 0, nil, nil
	}

	if tmpl, ok := item.Data["template"].(string); ok && tmpl != "" {
		item.Config.Template = tmpl
	}

	extra, err := b.buildSubLists(item, itemCfg)
	if err != nil {
		return 0, nil, err
	}
	itemCfg.Children = append(itemCfg.Children, extra...)

	if itemCfg.ListType == "photos" {
		srcDir := filepath.Dir(itemCfg.DataSource.Path)
		destDir := filepath.Join(b.cfg.OutputDir, filepath.Dir(itemCfg.OutputPath))
		if err := copyImages(srcDir, destDir); err != nil {
			return 0, nil, fmt.Errorf("copying photos for %q: %w", itemCfg.Name, err)
		}
	}

	title, _ := item.Data["title"].(string)
	childAncestors := make([]map[string]any, len(ancestors)+1)
	copy(childAncestors, ancestors)
	childAncestors[len(ancestors)] = map[string]any{"title": title, "outputPath": item.OutputPath}

	fragments, childCount, err := b.buildChildren(itemCfg, childAncestors)
	if err != nil {
		return 0, nil, err
	}

	cardData := make(map[string]any, len(item.Data))
	maps.Copy(cardData, item.Data)

	b.injectTemplateVars(item, ancestors, fragments, title)

	if err := writeItem(b.cfg.OutputDir, item, b.renderer); err != nil {
		return 0, nil, err
	}

	return childCount + 1, cardData, nil
}

// prepareItem creates the item from its datasource, applies item-type defaults,
// and sets a default icon if the item doesn't supply one.
func (b *Builder) prepareItem(itemCfg config.ItemConfig) (*Item, error) {
	ds, err := getDS(itemCfg, b.registry)
	if err != nil {
		return nil, fmt.Errorf("creating datasource: %w", err)
	}

	item, err := NewItem(itemCfg, ds)
	if err != nil {
		return nil, err
	}

	if typeName, ok := item.Data["type"].(string); ok && typeName != "" {
		if defaults := loadItemTypeDefaults(b.cfg.ItemsDir, typeName); defaults != nil {
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

	return item, nil
}

// enrich runs OG or YouTube enrichment on item if the relevant enricher is
// configured. Errors are logged as warnings; an unconfigured enricher is a no-op
// that leaves the "enrich" key intact.
func (b *Builder) enrich(item *Item) {
	enrichType, _ := item.Data["enrich"].(string)
	switch enrichType {
	case "opengraph":
		if b.ogEnricher == nil {
			return
		}
		if url, _ := item.Data["url"].(string); url != "" {
			force := b.cfg.RefreshOG || forceRefreshItem(item.Data)
			if ogData, err := b.ogEnricher.Enrich(url, force); err != nil {
				log.Printf("warning: OG enrichment failed for %s: %v", url, err)
			} else {
				maps.Copy(item.Data, ogData)
			}
		}
		delete(item.Data, "enrich")
	case "youtube-channel":
		if b.ytEnricher == nil {
			return
		}
		if channelID, _ := item.Data["channelId"].(string); channelID != "" {
			force := b.cfg.RefreshYouTube || forceRefreshYouTube(item.Data)
			if ytData, err := b.ytEnricher.Enrich(channelID, force); err != nil {
				log.Printf("warning: YouTube enrichment failed for %s: %v", channelID, err)
			} else {
				maps.Copy(item.Data, ytData)
			}
		}
		delete(item.Data, "enrich")
	}
}

// buildSubLists scans sibling directories named in the item's "lists" field
// and returns them as additional ItemConfigs to append to Children.
func (b *Builder) buildSubLists(item *Item, itemCfg config.ItemConfig) ([]config.ItemConfig, error) {
	rawLists, ok := item.Data["lists"].([]any)
	if !ok {
		return nil, nil
	}
	stem := stemOf(itemCfg.DataSource.Path)
	siblingDir := filepath.Join(filepath.Dir(itemCfg.DataSource.Path), stem)
	outputPrefix := strings.TrimSuffix(itemCfg.OutputPath, "index.html")
	var extra []config.ItemConfig
	for _, raw := range rawLists {
		name, _ := raw.(string)
		if name == "" {
			continue
		}
		sub, ok, err := scanDirItem(siblingDir, name, outputPrefix, b.cfg, listMeta{
			CardTemplate: itemCfg.CardTemplate,
			SortBy:       itemCfg.SortBy,
			SortOrder:    itemCfg.SortOrder,
			Limit:        itemCfg.Limit,
		})
		if err != nil {
			return nil, fmt.Errorf("scanning sub-list %q of %q: %w", name, itemCfg.Name, err)
		}
		if ok {
			extra = append(extra, sub)
		}
	}
	delete(item.Data, "lists")
	return extra, nil
}

// injectTemplateVars sets the standard page-level keys on item.Data.
// Must be called after the cardData snapshot is taken.
func (b *Builder) injectTemplateVars(item *Item, ancestors []map[string]any, fragments []template.HTML, title string) {
	staticJS := make([]string, len(b.cfg.StaticJS))
	for i, f := range b.cfg.StaticJS {
		staticJS[i] = "/static/" + f
	}
	item.Data["Site"] = b.cfg
	item.Data["OutputPath"] = item.OutputPath
	item.Data["RootItems"] = b.rootNavItems
	item.Data["List"] = fragments
	item.Data["Theme"] = b.themeData
	item.Data["StaticJS"] = staticJS
	breadcrumbLinks := ancestors
	if sp, ok := item.Data["sourcePath"].([]map[string]any); ok {
		breadcrumbLinks = sp
	}
	item.Data["BreadcrumbLinks"] = breadcrumbLinks
	item.Data["BreadcrumbCurrent"] = title
	item.Data["SiteMap"] = b.siteMap
	item.Data["PageTemplate"] = item.Config.Template
}

// childEntry pairs an ItemConfig with the fetched data needed for sorting and
// card rendering.
type childEntry struct {
	cfg  config.ItemConfig
	data map[string]any
}

// buildChildren builds each child's page recursively and renders card fragments
// for the parent's List. Enriched data returned by buildItem is reused directly
// for card rendering so each item is fetched and enriched only once.
func (b *Builder) buildChildren(itemCfg config.ItemConfig, ancestors []map[string]any) ([]template.HTML, int, error) {
	if len(itemCfg.Children) == 0 {
		return nil, 0, nil
	}

	// Build every child page and collect the pre-injection card data in one pass.
	entries := make([]childEntry, 0, len(itemCfg.Children))
	totalCount := 0
	for _, childCfg := range itemCfg.Children {
		n, cardData, err := b.buildItem(childCfg, ancestors)
		if err != nil {
			return nil, 0, fmt.Errorf("building child %q: %w", childCfg.Name, err)
		}
		totalCount += n
		if cardData == nil {
			continue // draft item
		}
		cardData["outputPath"] = childCfg.OutputPath
		cardData["count"] = len(childCfg.Children)
		entries = append(entries, childEntry{cfg: childCfg, data: cardData})
	}

	// Sort and apply limit.
	if itemCfg.SortBy != "" {
		sortChildEntries(entries, itemCfg.SortBy, itemCfg.SortOrder)
	}
	if itemCfg.Limit > 0 && len(entries) > itemCfg.Limit {
		entries = entries[:itemCfg.Limit]
	}

	// Inject nested list fragments before rendering cards.
	for i := range entries {
		if len(entries[i].cfg.Children) > 0 {
			grandFragments, err := b.renderChildCards(entries[i].cfg)
			if err != nil {
				return nil, 0, fmt.Errorf("rendering nested cards for %q: %w", entries[i].cfg.Name, err)
			}
			entries[i].data["List"] = grandFragments
		}
	}

	fragments, err := b.renderCards(itemCfg, entries)
	if err != nil {
		return nil, 0, err
	}

	return fragments, totalCount, nil
}

// renderChildCards fetches and renders card fragments for itemCfg's children
// without building their output pages. Used to populate List in card templates
// when a child is itself a list.
func (b *Builder) renderChildCards(itemCfg config.ItemConfig) ([]template.HTML, error) {
	entries := make([]childEntry, 0, len(itemCfg.Children))
	for _, childCfg := range itemCfg.Children {
		ds, err := getDS(childCfg, b.registry)
		if err != nil {
			return nil, fmt.Errorf("creating datasource for %q: %w", childCfg.Name, err)
		}
		data, err := ds.FetchOne()
		if err != nil {
			return nil, fmt.Errorf("fetching data for %q: %w", childCfg.Name, err)
		}
		if typeName, ok := data["type"].(string); ok && typeName != "" {
			if defaults := loadItemTypeDefaults(b.cfg.ItemsDir, typeName); defaults != nil {
				applyTypeDefaults(data, defaults)
			}
		}
		if isDraft(data) && !b.cfg.Drafts {
			continue
		}
		data["outputPath"] = childCfg.OutputPath
		entries = append(entries, childEntry{cfg: childCfg, data: data})
	}

	if itemCfg.SortBy != "" {
		sortChildEntries(entries, itemCfg.SortBy, itemCfg.SortOrder)
	}
	if itemCfg.Limit > 0 && len(entries) > itemCfg.Limit {
		entries = entries[:itemCfg.Limit]
	}

	return b.renderCards(itemCfg, entries)
}

// renderCards renders a card fragment for each entry using the parent list's
// card template (overridable per entry). Entries must already be sorted and limited.
func (b *Builder) renderCards(parent config.ItemConfig, entries []childEntry) ([]template.HTML, error) {
	fragments := make([]template.HTML, 0, len(entries))
	for _, e := range entries {
		cardTemplate := parent.CardTemplate
		if t, ok := e.data["cardTemplate"].(string); ok && t != "" {
			cardTemplate = t
		}
		fragment, err := b.renderer.RenderCard(cardTemplate, e.data)
		if err != nil {
			return nil, fmt.Errorf("rendering card for %q: %w", e.cfg.Name, err)
		}
		fragments = append(fragments, fragment)
	}
	return fragments, nil
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

// readSidecar reads a YAML sidecar file and returns its contents as a map.
// Returns nil if the file does not exist or cannot be parsed.
func readSidecar(path string) map[string]any {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var m map[string]any
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil
	}
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

// forceRefreshYouTube reports whether item data contains yt_refresh: true.
func forceRefreshYouTube(data map[string]any) bool {
	b, ok := data["yt_refresh"].(bool)
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

// copyStaticDir copies all files from src into outputDir/static/, preserving
// directory structure. Skips silently if src does not exist.
func copyStaticDir(src, outputDir string) error {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil
	}
	destRoot := filepath.Join(outputDir, "static")
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(destRoot, rel)
		if d.IsDir() {
			return os.MkdirAll(dest, 0755)
		}
		return copyFile(path, dest)
	})
}

// copyImages copies all image files from src into dest, preserving filenames.
// Skips silently if src does not exist.
func copyImages(src, dest string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if !imageExts[ext] {
			continue
		}
		if err := copyFile(filepath.Join(src, entry.Name()), filepath.Join(dest, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
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
