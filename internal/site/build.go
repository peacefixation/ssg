package site

import (
	"fmt"
	"html/template"
	"log"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/peacefixation/ssg/internal/config"
	"github.com/peacefixation/ssg/internal/datasource"
	"github.com/peacefixation/ssg/internal/enricher"
	"github.com/peacefixation/ssg/internal/renderer"
	"github.com/peacefixation/ssg/internal/theme"
)

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
	if err := setupOutput(cfg, clean); err != nil {
		return 0, err
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

	rootNavItems := buildRootNav(rootItems, registry)

	var siteMap []config.SiteMapNode
	if cfg.SiteMap {
		siteMap = buildSiteMap(rootItems, registry, cfg.ItemsDir)
	}

	ogEnricher, ytEnricher, cleanup := initEnrichers(cfg)
	defer cleanup()

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

// setupOutput cleans (if requested) and creates the output directory.
func setupOutput(cfg *config.SiteConfig, clean bool) error {
	if clean {
		if err := os.RemoveAll(cfg.OutputDir); err != nil {
			return fmt.Errorf("cleaning output dir: %w", err)
		}
	}
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}
	return nil
}

// buildRootNav fetches nav metadata for all root items and filters out the
// homepage — the site title serves as the home link in the global nav.
func buildRootNav(items []config.ItemConfig, registry *datasource.Registry) []map[string]any {
	all := buildNavItems(items, registry)
	nav := make([]map[string]any, 0, len(all))
	for _, item := range all {
		if item["outputPath"] != "index.html" {
			nav = append(nav, item)
		}
	}
	return nav
}

// initEnrichers creates and warms the OG and YouTube enrichers from cfg.
// The returned cleanup func saves both caches and should be deferred by the caller.
func initEnrichers(cfg *config.SiteConfig) (*enricher.OGEnricher, *enricher.YouTubeEnricher, func()) {
	var og *enricher.OGEnricher
	if cfg.OGCacheFile != "" {
		referer := cfg.CanonicalURL
		if referer == "" {
			referer = cfg.BaseURL
		}
		og = enricher.New(cfg.OGCacheFile, referer)
		if err := og.LoadCache(); err != nil {
			log.Printf("warning: loading OG cache: %v", err)
		}
	}

	var yt *enricher.YouTubeEnricher
	if cfg.YouTubeCacheFile != "" && cfg.YouTubeAPIKey != "" {
		yt = enricher.NewYouTube(cfg.YouTubeCacheFile, cfg.YouTubeAPIKey)
		if err := yt.LoadCache(); err != nil {
			log.Printf("warning: loading YouTube cache: %v", err)
		}
	}

	cleanup := func() {
		if og != nil {
			if err := og.SaveCache(); err != nil {
				log.Printf("warning: saving OG cache: %v", err)
			}
		}
		if yt != nil {
			if err := yt.SaveCache(); err != nil {
				log.Printf("warning: saving YouTube cache: %v", err)
			}
		}
	}

	return og, yt, cleanup
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
