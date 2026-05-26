package site

import (
	"fmt"
	"html/template"
	"sort"
	"time"

	"github.com/peacefixation/ssg/internal/config"
)

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
