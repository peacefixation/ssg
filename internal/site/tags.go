package site

import (
	"regexp"
	"sort"
	"strings"

	"github.com/peacefixation/ssg/internal/config"
	"github.com/peacefixation/ssg/internal/datasource"
)

// taggedItem pairs an ItemConfig with the ordered ancestor list chain leading to it
// (outermost first), captured at collection time.
type taggedItem struct {
	config    config.ItemConfig
	ancestors []map[string]any // [{title, outputPath}, ...]
}

// collectTags recursively walks items and returns a map from normalised tag name
// to the items carrying that tag. ancestors tracks the list chain at the current
// level of recursion and should be passed as nil at the top level.
func collectTags(items []config.ItemConfig, registry *datasource.Registry, ancestors []map[string]any) map[string][]taggedItem {
	result := make(map[string][]taggedItem)
	for _, item := range items {
		if len(item.Children) > 0 {
			// Directory item: fetch its title and recurse into children.
			title := item.Name
			if ds, err := registry.New(item.DataSource); err == nil {
				if data, err := ds.FetchOne(); err == nil {
					if t, ok := data["title"].(string); ok && t != "" {
						title = t
					}
				}
			}
			childAncestors := make([]map[string]any, len(ancestors)+1)
			copy(childAncestors, ancestors)
			childAncestors[len(ancestors)] = map[string]any{
				"title":      title,
				"outputPath": item.OutputPath,
			}
			for tag, tagged := range collectTags(item.Children, registry, childAncestors) {
				result[tag] = append(result[tag], tagged...)
			}
		} else {
			// Leaf item: read tags from its data.
			ds, err := registry.New(item.DataSource)
			if err != nil {
				continue
			}
			data, err := ds.FetchOne()
			if err != nil {
				continue
			}
			if isDraft(data) {
				continue
			}
			for _, tag := range extractTags(data) {
				result[tag] = append(result[tag], taggedItem{
					config:    item,
					ancestors: ancestors,
				})
			}
		}
	}
	return result
}

// extractTags reads the "tags" field from data and returns normalised tag strings.
// It handles both a comma-separated string and a YAML list of strings.
func extractTags(data map[string]any) []string {
	raw, ok := data["tags"]
	if !ok {
		return nil
	}
	var tags []string
	switch v := raw.(type) {
	case string:
		for _, t := range strings.Split(v, ",") {
			t = strings.TrimSpace(strings.ToLower(t))
			if t != "" {
				tags = append(tags, t)
			}
		}
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				s = strings.TrimSpace(strings.ToLower(s))
				if s != "" {
					tags = append(tags, s)
				}
			}
		}
	}
	return tags
}

// styleTemplates maps a style name to its default [template, cardTemplate] pair.
var styleTemplates = map[string][2]string{
	"list":    {"tags-list.html", "tag-list-card.html"},
	"cloud":   {"tags-cloud.html", "tag-cloud-card.html"},
	"heatmap": {"tags-heatmap.html", "tag-heatmap-card.html"},
}

// buildTagsTree constructs the virtual tags/ root ItemConfig from the collected
// tag map. The root lists all tags as children; each tag is itself a list whose
// children are the real items carrying that tag, decorated with sourcePath.
func buildTagsTree(tagMap map[string][]taggedItem, cfg *config.SiteConfig, registry *datasource.Registry) config.ItemConfig {
	// Resolve style-derived template defaults, then apply explicit overrides.
	style := cfg.Tags.Style
	var styleTemplate, styleCardTemplate string
	if d, ok := styleTemplates[style]; ok {
		styleTemplate, styleCardTemplate = d[0], d[1]
	}
	tagsTemplate := first(cfg.Tags.Template, styleTemplate, "tags.html")
	tagCardTemplate := first(cfg.Tags.CardTemplate, styleCardTemplate, "tag-card.html")
	tagTemplate := first(cfg.Tags.TagTemplate, "tag.html")
	itemCardTemplate := first(cfg.Tags.ItemCardTemplate, "tag-item-card.html")

	// Stable alphabetical order for tags.
	sortedTags := make([]string, 0, len(tagMap))
	for tag := range tagMap {
		sortedTags = append(sortedTags, tag)
	}
	sort.Strings(sortedTags)

	// Compute min/max counts for weight normalisation.
	minCount, maxCount := len(tagMap[sortedTags[0]]), len(tagMap[sortedTags[0]])
	for _, tag := range sortedTags {
		c := len(tagMap[tag])
		if c < minCount {
			minCount = c
		}
		if c > maxCount {
			maxCount = c
		}
	}

	tagChildren := make([]config.ItemConfig, 0, len(sortedTags))
	for _, tag := range sortedTags {
		items := tagMap[tag]
		slug := tagSlug(tag)

		weight := 1.0
		if maxCount > minCount {
			weight = float64(len(items)-minCount) / float64(maxCount-minCount)
		}

		children := make([]config.ItemConfig, 0, len(items))
		for _, ti := range items {
			inner, err := registry.New(ti.config.DataSource)
			if err != nil {
				continue
			}
			child := ti.config
			child.DataSourceOverride = datasource.NewDecoratedSource(inner, map[string]any{
				"sourcePath": ti.ancestors,
			})
			children = append(children, child)
		}

		tagChildren = append(tagChildren, config.ItemConfig{
			Name:         slug,
			Template:     tagTemplate,
			CardTemplate: itemCardTemplate,
			OutputPath:   "tags/" + slug + "/index.html",
			DataSource: config.DataSourceConfig{
				Type: config.MapType,
				Data: map[string]any{
					"title":  tag,
					"tag":    tag,
					"count":  len(items),
					"weight": weight,
				},
			},
			Children:  children,
			SortBy:    cfg.Defaults.List.SortBy,
			SortOrder: cfg.Defaults.List.SortOrder,
		})
	}

	return config.ItemConfig{
		Name:         "tags",
		Template:     tagsTemplate,
		CardTemplate: tagCardTemplate,
		OutputPath:   "tags/index.html",
		DataSource: config.DataSourceConfig{
			Type: config.MapType,
			Data: map[string]any{
				"title": "Tags",
				"style": first(style, "list"),
			},
		},
		Children:           tagChildren,
		SortBy:             "title",
		SortOrder:          "asc",
		ExcludeFromSiteMap: true,
	}
}

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9-]+`)

// tagSlug converts a tag name to a URL-safe slug.
func tagSlug(tag string) string {
	slug := strings.ToLower(tag)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = nonAlphaNum.ReplaceAllString(slug, "")
	return strings.Trim(slug, "-")
}
