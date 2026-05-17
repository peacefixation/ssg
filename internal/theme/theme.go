package theme

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is the in-memory representation of a theme.yaml file.
type Config struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	CSS         []string `yaml:"css"`
	JS          []string `yaml:"js"`
	CDNCSS      []string `yaml:"cdnCSS"`
	CDNJS       []string `yaml:"cdnJS"`
}

// Data is injected into every template under the key "Theme".
type Data struct {
	Name   string
	CSS    []string // root-relative output paths, e.g. ["/theme/style.css"]
	JS     []string // root-relative output paths, e.g. ["/theme/main.js"]
	CDNCSS []string // external CDN stylesheet URLs
	CDNJS  []string // external CDN script URLs
}

// Load reads theme.yaml from themeDir and returns the parsed Config.
func Load(themeDir string) (*Config, error) {
	path := filepath.Join(themeDir, "theme.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading theme config %q: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parsing theme config %q: %w", path, err)
	}
	return &cfg, nil
}

// TemplateDir returns the path to the theme's template partials directory.
func TemplateDir(themeDir string) string {
	return filepath.Join(themeDir, "templates")
}

// BuildData constructs the Data value injected into every template, mapping
// asset filenames to their root-relative output paths under /theme/.
func BuildData(cfg *Config) Data {
	css := make([]string, len(cfg.CSS))
	for i, f := range cfg.CSS {
		css[i] = "/theme/" + f
	}
	js := make([]string, len(cfg.JS))
	for i, f := range cfg.JS {
		js[i] = "/theme/" + f
	}
	return Data{Name: cfg.Name, CSS: css, JS: js, CDNCSS: cfg.CDNCSS, CDNJS: cfg.CDNJS}
}

// CopyAssets copies all theme files (excluding the templates/ subdirectory)
// from themeDir into outputDir/theme/, preserving directory structure.
func CopyAssets(themeDir, outputDir string) error {
	destRoot := filepath.Join(outputDir, "theme")
	skipTemplates := filepath.Join(themeDir, "templates")

	return filepath.WalkDir(themeDir, func(src string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Skip the templates/ subdirectory — it is for the renderer, not output.
		if d.IsDir() && src == skipTemplates {
			return filepath.SkipDir
		}
		// Skip theme.yaml itself.
		if !d.IsDir() && filepath.Base(src) == "theme.yaml" {
			return nil
		}

		rel, err := filepath.Rel(themeDir, src)
		if err != nil {
			return err
		}
		dest := filepath.Join(destRoot, rel)

		if d.IsDir() {
			return os.MkdirAll(dest, 0755)
		}
		return copyFile(src, dest)
	})
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
