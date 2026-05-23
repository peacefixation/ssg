package site_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/peacefixation/ssg/internal/config"
	"github.com/peacefixation/ssg/internal/datasource"
	"github.com/peacefixation/ssg/internal/site"
)

// buildTestSite runs Build with a minimal in-memory config rooted at dir.
func buildTestSite(t *testing.T, dir string) {
	t.Helper()
	cfg := &config.SiteConfig{
		Title:       "Test Site",
		ContentDir:  filepath.Join(dir, "content"),
		OutputDir:   filepath.Join(dir, "output"),
		TemplateDir: filepath.Join(dir, "templates"),
		ThemesDir:   filepath.Join(dir, "themes"),
		ItemsDir:    filepath.Join(dir, "items"),
		Defaults: config.Defaults{
			Page: config.PageDefaults{Template: "item.html"},
			List: config.ListDefaults{
				Template:     "list.html",
				CardTemplate: "card.html",
			},
		},
	}
	_, err := site.Build(cfg, datasource.DefaultRegistry(), false)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
}

// mustWriteFile creates parent directories and writes content to path.
func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// assertFile checks that path exists under the output directory.
func assertFile(t *testing.T, outputDir, rel string) {
	t.Helper()
	full := filepath.Join(outputDir, filepath.FromSlash(rel))
	if _, err := os.Stat(full); err != nil {
		t.Errorf("expected output file %s: %v", rel, err)
	}
}

func TestBuild_FileItemWithSubList(t *testing.T) {
	dir := t.TempDir()

	// Minimal templates — no head.html/foot.html required.
	mustWriteFile(t, filepath.Join(dir, "templates", "item.html"),
		`{{define "item.html"}}{{.title}}{{end}}`)
	mustWriteFile(t, filepath.Join(dir, "templates", "list.html"),
		`{{define "list.html"}}{{.title}}{{range .List}}{{.}}{{end}}{{end}}`)
	mustWriteFile(t, filepath.Join(dir, "templates", "card.html"),
		`{{define "card.html"}}{{.title}}{{end}}`)

	// File item declaring one sub-list.
	mustWriteFile(t, filepath.Join(dir, "content", "20260418T120000Z-lsg.yaml"),
		"title: LSG\nlists:\n  - live\n")

	// Sub-list directory with list.yaml and one content item.
	mustWriteFile(t, filepath.Join(dir, "content", "20260418T120000Z-lsg", "live", "list.yaml"),
		"title: Live Sets\n")
	mustWriteFile(t, filepath.Join(dir, "content", "20260418T120000Z-lsg", "live", "20260501T000000Z-wembley.yaml"),
		"title: Wembley\n")

	buildTestSite(t, dir)

	out := filepath.Join(dir, "output")
	assertFile(t, out, "20260418T120000Z-lsg/index.html")
	assertFile(t, out, "20260418T120000Z-lsg/live/index.html")
	assertFile(t, out, "20260418T120000Z-lsg/live/20260501T000000Z-wembley/index.html")
}

func TestBuild_FileItemWithSubList_DeepNesting(t *testing.T) {
	dir := t.TempDir()

	mustWriteFile(t, filepath.Join(dir, "templates", "item.html"),
		`{{define "item.html"}}{{.title}}{{end}}`)
	mustWriteFile(t, filepath.Join(dir, "templates", "list.html"),
		`{{define "list.html"}}{{.title}}{{range .List}}{{.}}{{end}}{{end}}`)
	mustWriteFile(t, filepath.Join(dir, "templates", "card.html"),
		`{{define "card.html"}}{{.title}}{{end}}`)

	// Top-level file item.
	mustWriteFile(t, filepath.Join(dir, "content", "20260418T120000Z-lsg.yaml"),
		"title: LSG\nlists:\n  - live\n")

	// Sub-list with a nested file item that also declares a sub-list.
	mustWriteFile(t, filepath.Join(dir, "content", "20260418T120000Z-lsg", "live", "list.yaml"),
		"title: Live Sets\n")
	mustWriteFile(t, filepath.Join(dir, "content", "20260418T120000Z-lsg", "live", "20260501T000000Z-wembley.yaml"),
		"title: Wembley\nlists:\n  - sets\n")

	// Sub-sub-list.
	mustWriteFile(t, filepath.Join(dir, "content", "20260418T120000Z-lsg", "live", "20260501T000000Z-wembley", "sets", "list.yaml"),
		"title: Sets\n")
	mustWriteFile(t, filepath.Join(dir, "content", "20260418T120000Z-lsg", "live", "20260501T000000Z-wembley", "sets", "20260501T200000Z-set1.yaml"),
		"title: Set 1\n")

	buildTestSite(t, dir)

	out := filepath.Join(dir, "output")
	assertFile(t, out, "20260418T120000Z-lsg/index.html")
	assertFile(t, out, "20260418T120000Z-lsg/live/index.html")
	assertFile(t, out, "20260418T120000Z-lsg/live/20260501T000000Z-wembley/index.html")
	assertFile(t, out, "20260418T120000Z-lsg/live/20260501T000000Z-wembley/sets/index.html")
	assertFile(t, out, "20260418T120000Z-lsg/live/20260501T000000Z-wembley/sets/20260501T200000Z-set1/index.html")
}

func TestBuild_FileItemWithNoSubListDir_Skipped(t *testing.T) {
	dir := t.TempDir()

	mustWriteFile(t, filepath.Join(dir, "templates", "item.html"),
		`{{define "item.html"}}{{.title}}{{end}}`)
	mustWriteFile(t, filepath.Join(dir, "templates", "list.html"),
		`{{define "list.html"}}{{.title}}{{end}}`)
	mustWriteFile(t, filepath.Join(dir, "templates", "card.html"),
		`{{define "card.html"}}{{.title}}{{end}}`)

	// File item declares "live" but the sibling directory does not exist.
	mustWriteFile(t, filepath.Join(dir, "content", "20260418T120000Z-lsg.yaml"),
		"title: LSG\nlists:\n  - live\n")

	// Build should succeed; the missing sub-list is silently skipped.
	buildTestSite(t, dir)

	assertFile(t, filepath.Join(dir, "output"), "20260418T120000Z-lsg/index.html")
}
