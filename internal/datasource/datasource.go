package datasource

import (
	"fmt"

	"github.com/peacefixation/ssg/internal/config"
)

// DataSource loads data for items and lists.
type DataSource interface {
	// FetchOne returns a single data record, used to populate an item.
	FetchOne() (map[string]any, error)
	// FetchMany returns multiple records, used to populate a list.
	FetchMany() ([]map[string]any, error)
}

// FactoryFunc constructs a DataSource from a config.
type FactoryFunc func(cfg config.DataSourceConfig) (DataSource, error)

// Registry maps datasource type names to their factory functions.
type Registry struct {
	factories map[config.DataSourceType]FactoryFunc
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{factories: make(map[config.DataSourceType]FactoryFunc)}
}

// Register adds a factory for the given type.
func (r *Registry) Register(typeName config.DataSourceType, factory FactoryFunc) {
	r.factories[typeName] = factory
}

// New constructs a DataSource for the given config.
func (r *Registry) New(cfg config.DataSourceConfig) (DataSource, error) {
	factory, ok := r.factories[cfg.Type]
	if !ok {
		return nil, fmt.Errorf("unknown datasource type: %q", cfg.Type)
	}
	return factory(cfg)
}

// DefaultRegistry returns a Registry pre-populated with the built-in drivers.
func DefaultRegistry() *Registry {
	r := NewRegistry()
	r.Register(config.FileType, func(cfg config.DataSourceConfig) (DataSource, error) {
		return NewFileSource(cfg)
	})
	r.Register(config.APIType, func(cfg config.DataSourceConfig) (DataSource, error) {
		return NewAPISource(cfg)
	})
	return r
}
