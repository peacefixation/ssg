package datasource

// DecoratedSource wraps an inner DataSource and merges extra fields into
// the data returned by FetchOne. Extra fields do not overwrite existing ones.
type DecoratedSource struct {
	inner  DataSource
	extras map[string]any
}

// NewDecoratedSource returns a DecoratedSource wrapping inner with the given extras.
func NewDecoratedSource(inner DataSource, extras map[string]any) *DecoratedSource {
	return &DecoratedSource{inner: inner, extras: extras}
}

func (d *DecoratedSource) FetchOne() (map[string]any, error) {
	data, err := d.inner.FetchOne()
	if err != nil {
		return nil, err
	}
	for k, v := range d.extras {
		if _, exists := data[k]; !exists {
			data[k] = v
		}
	}
	return data, nil
}

func (d *DecoratedSource) FetchMany() ([]map[string]any, error) {
	data, err := d.FetchOne()
	if err != nil {
		return nil, err
	}
	return []map[string]any{data}, nil
}
