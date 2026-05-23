package datasource

// MapSource implements DataSource by returning a fixed in-memory map.
// It is used for synthesized items that have no backing file on disk.
type MapSource struct {
	data map[string]any
}

// NewMapSource returns a MapSource that serves data on every FetchOne call.
func NewMapSource(data map[string]any) *MapSource {
	return &MapSource{data: data}
}

func (m *MapSource) FetchOne() (map[string]any, error) {
	out := make(map[string]any, len(m.data))
	for k, v := range m.data {
		out[k] = v
	}
	return out, nil
}

func (m *MapSource) FetchMany() ([]map[string]any, error) {
	d, err := m.FetchOne()
	if err != nil {
		return nil, err
	}
	return []map[string]any{d}, nil
}
