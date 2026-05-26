package site

import (
	"path/filepath"
	"strings"

	"github.com/peacefixation/ssg/internal/config"
	"github.com/peacefixation/ssg/internal/datasource"
)

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
