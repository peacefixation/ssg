package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/peacefixation/ssg/internal/config"
	"gopkg.in/yaml.v3"
)

// --- appendListToFile ---

func TestAppendListYAML_NoExistingLists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "item.yaml")
	if err := os.WriteFile(path, []byte("title: LSG\ntype: soundcloud\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := appendListToFile(path, "live"); err != nil {
		t.Fatalf("appendListToFile: %v", err)
	}
	meta := readFileItemMeta(path)
	if len(meta.Lists) != 1 || meta.Lists[0] != "live" {
		t.Errorf("expected lists=[live], got %v", meta.Lists)
	}
}

func TestAppendListYAML_ExistingLists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "item.yaml")
	if err := os.WriteFile(path, []byte("title: LSG\nlists:\n  - live\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := appendListToFile(path, "2002"); err != nil {
		t.Fatalf("appendListToFile: %v", err)
	}
	meta := readFileItemMeta(path)
	if len(meta.Lists) != 2 || meta.Lists[0] != "live" || meta.Lists[1] != "2002" {
		t.Errorf("expected lists=[live 2002], got %v", meta.Lists)
	}
}

func TestAppendListJSON_NoExistingLists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "item.json")
	data, _ := json.Marshal(map[string]any{"title": "LSG", "type": "youtube"})
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	if err := appendListToFile(path, "live"); err != nil {
		t.Fatalf("appendListToFile: %v", err)
	}
	meta := readFileItemMeta(path)
	if len(meta.Lists) != 1 || meta.Lists[0] != "live" {
		t.Errorf("expected lists=[live], got %v", meta.Lists)
	}
}

func TestAppendListJSON_ExistingLists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "item.json")
	data, _ := json.Marshal(map[string]any{"title": "LSG", "lists": []string{"live"}})
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	if err := appendListToFile(path, "2002"); err != nil {
		t.Fatalf("appendListToFile: %v", err)
	}
	meta := readFileItemMeta(path)
	if len(meta.Lists) != 2 || meta.Lists[0] != "live" || meta.Lists[1] != "2002" {
		t.Errorf("expected lists=[live 2002], got %v", meta.Lists)
	}
}

func TestAppendListMarkdown_WithFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "item.md")
	if err := os.WriteFile(path, []byte("---\ntitle: LSG\n---\n\nBody text.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := appendListToFile(path, "live"); err != nil {
		t.Fatalf("appendListToFile: %v", err)
	}
	data, _ := os.ReadFile(path)
	fm := extractFrontmatter(data)
	if fm == nil {
		t.Fatal("expected frontmatter")
	}
	var m struct {
		Title string   `yaml:"title"`
		Lists []string `yaml:"lists"`
	}
	_ = yaml.Unmarshal(fm, &m)
	if len(m.Lists) != 1 || m.Lists[0] != "live" {
		t.Errorf("expected lists=[live], got %v", m.Lists)
	}
	if m.Title != "LSG" {
		t.Errorf("title not preserved: %q", m.Title)
	}
}

func TestAppendListMarkdown_NoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "item.md")
	if err := os.WriteFile(path, []byte("Just a body.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := appendListToFile(path, "live"); err != nil {
		t.Fatalf("appendListToFile: %v", err)
	}
	data, _ := os.ReadFile(path)
	fm := extractFrontmatter(data)
	if fm == nil {
		t.Fatal("expected frontmatter to be added")
	}
	var m struct {
		Lists []string `yaml:"lists"`
	}
	_ = yaml.Unmarshal(fm, &m)
	if len(m.Lists) != 1 || m.Lists[0] != "live" {
		t.Errorf("expected lists=[live], got %v", m.Lists)
	}
}

// --- createList with file item parent ---

func testSiteConfig(contentDir string) *config.SiteConfig {
	return &config.SiteConfig{ContentDir: contentDir}
}

func mustMkListDir(t *testing.T, dir, title string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	data, _ := yaml.Marshal(map[string]any{"title": title})
	if err := os.WriteFile(filepath.Join(dir, "list.yaml"), data, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestCreateList_FileItemParent_CreatesSubListAndUpdatesFile(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")
	mustMkListDir(t, contentDir, "Site")
	if err := os.WriteFile(filepath.Join(contentDir, "20260418-lsg.yaml"),
		[]byte("title: LSG\ntype: soundcloud\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := createList(testSiteConfig(contentDir), "20260418-lsg/live", newListConfig{Title: "Live Sets"}); err != nil {
		t.Fatalf("createList: %v", err)
	}

	if _, err := os.Stat(filepath.Join(contentDir, "20260418-lsg", "live", "list.yaml")); err != nil {
		t.Errorf("expected sub-list list.yaml: %v", err)
	}
	meta := readFileItemMeta(filepath.Join(contentDir, "20260418-lsg.yaml"))
	if len(meta.Lists) != 1 || meta.Lists[0] != "live" {
		t.Errorf("expected lists=[live], got %v", meta.Lists)
	}
}

func TestCreateList_FileItemParent_SiblingDirCreatedWhenMissing(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")
	mustMkListDir(t, contentDir, "Site")
	if err := os.WriteFile(filepath.Join(contentDir, "20260418-lsg.yaml"),
		[]byte("title: LSG\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := createList(testSiteConfig(contentDir), "20260418-lsg/live", newListConfig{Title: "Live"}); err != nil {
		t.Fatalf("createList: %v", err)
	}
	if _, err := os.Stat(filepath.Join(contentDir, "20260418-lsg")); err != nil {
		t.Errorf("expected sibling dir to be created: %v", err)
	}
}

func TestCreateList_FileItemParent_SecondSubList(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")
	mustMkListDir(t, contentDir, "Site")
	if err := os.WriteFile(filepath.Join(contentDir, "20260418-lsg.yaml"),
		[]byte("title: LSG\nlists:\n  - live\n"), 0644); err != nil {
		t.Fatal(err)
	}
	mustMkListDir(t, filepath.Join(contentDir, "20260418-lsg", "live"), "Live Sets")

	if err := createList(testSiteConfig(contentDir), "20260418-lsg/2002", newListConfig{Title: "2002"}); err != nil {
		t.Fatalf("createList: %v", err)
	}
	meta := readFileItemMeta(filepath.Join(contentDir, "20260418-lsg.yaml"))
	if len(meta.Lists) != 2 {
		t.Errorf("expected 2 lists, got %v", meta.Lists)
	}
}
