package datasource

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/peacefixation/ssg/internal/config"
)

// APISource reads data from an HTTP API endpoint.
type APISource struct {
	cfg    config.DataSourceConfig
	client *http.Client
}

// NewAPISource returns an APISource. An optional "timeout" (seconds) may be
// set in cfg.Params; the default is 30 s.
func NewAPISource(cfg config.DataSourceConfig) (*APISource, error) {
	timeout := 30 * time.Second
	if t, ok := cfg.Params["timeout"]; ok {
		secs, err := strconv.Atoi(t)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout %q: %w", t, err)
		}
		timeout = time.Duration(secs) * time.Second
	}
	return &APISource{
		cfg:    cfg,
		client: &http.Client{Timeout: timeout},
	}, nil
}

// FetchOne GETs cfg.Path and unmarshals the JSON response as an object.
func (a *APISource) FetchOne() (map[string]any, error) {
	body, err := a.fetch()
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing API response as object: %w", err)
	}
	return result, nil
}

// FetchMany GETs cfg.Path and unmarshals the JSON response as an array.
func (a *APISource) FetchMany() ([]map[string]any, error) {
	body, err := a.fetch()
	if err != nil {
		return nil, err
	}
	var result []map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing API response as array: %w", err)
	}
	return result, nil
}

func (a *APISource) fetch() ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, a.cfg.Path, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	for k, v := range a.cfg.Headers {
		req.Header.Set(k, v)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", a.cfg.Path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API %s returned status %d", a.cfg.Path, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}
	return body, nil
}
