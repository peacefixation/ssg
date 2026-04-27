package site

import (
	"fmt"

	"github.com/peacefixation/ssg/internal/config"
	"github.com/peacefixation/ssg/internal/datasource"
)

// Item is a runtime instance of an ItemConfig with its data loaded.
type Item struct {
	Config     config.ItemConfig
	Data       map[string]any
	OutputPath string
}

// NewItem fetches data from ds and returns a populated Item ready for rendering.
func NewItem(cfg config.ItemConfig, ds datasource.DataSource) (*Item, error) {
	data, err := ds.FetchOne()
	if err != nil {
		return nil, fmt.Errorf("fetching data for item %q: %w", cfg.Name, err)
	}
	return &Item{
		Config:     cfg,
		Data:       data,
		OutputPath: cfg.OutputPath,
	}, nil
}
