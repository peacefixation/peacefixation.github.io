package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/peacefixation/ssg/internal/config"
	"gopkg.in/yaml.v3"
)

const (
	testItemTitle = "Test Item"
	testItemSlug  = "20260418-item"
	testListA     = "list-a"
	testListB     = "list-b"
)

// --- appendListToFile ---

func TestAppendListYAML_NoExistingLists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "item.yaml")
	if err := os.WriteFile(path, []byte("title: "+testItemTitle+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := appendListToFile(path, testListA); err != nil {
		t.Fatalf("appendListToFile: %v", err)
	}
	meta := readFileItemMeta(path)
	if len(meta.Lists) != 1 || meta.Lists[0] != testListA {
		t.Errorf("expected lists=[%s], got %v", testListA, meta.Lists)
	}
}

func TestAppendListYAML_ExistingLists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "item.yaml")
	if err := os.WriteFile(path, []byte("title: "+testItemTitle+"\nlists:\n  - "+testListA+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := appendListToFile(path, testListB); err != nil {
		t.Fatalf("appendListToFile: %v", err)
	}
	meta := readFileItemMeta(path)
	if len(meta.Lists) != 2 || meta.Lists[0] != testListA || meta.Lists[1] != testListB {
		t.Errorf("expected lists=[%s %s], got %v", testListA, testListB, meta.Lists)
	}
}

func TestAppendListJSON_NoExistingLists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "item.json")
	data, _ := json.Marshal(map[string]any{"title": testItemTitle})
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	if err := appendListToFile(path, testListA); err != nil {
		t.Fatalf("appendListToFile: %v", err)
	}
	meta := readFileItemMeta(path)
	if len(meta.Lists) != 1 || meta.Lists[0] != testListA {
		t.Errorf("expected lists=[%s], got %v", testListA, meta.Lists)
	}
}

func TestAppendListJSON_ExistingLists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "item.json")
	data, _ := json.Marshal(map[string]any{"title": testItemTitle, "lists": []string{testListA}})
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	if err := appendListToFile(path, testListB); err != nil {
		t.Fatalf("appendListToFile: %v", err)
	}
	meta := readFileItemMeta(path)
	if len(meta.Lists) != 2 || meta.Lists[0] != testListA || meta.Lists[1] != testListB {
		t.Errorf("expected lists=[%s %s], got %v", testListA, testListB, meta.Lists)
	}
}

func TestAppendListMarkdown_WithFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "item.md")
	if err := os.WriteFile(path, []byte("---\ntitle: "+testItemTitle+"\n---\n\nBody text.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := appendListToFile(path, testListA); err != nil {
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
	if len(m.Lists) != 1 || m.Lists[0] != testListA {
		t.Errorf("expected lists=[%s], got %v", testListA, m.Lists)
	}
	if m.Title != testItemTitle {
		t.Errorf("title not preserved: %q", m.Title)
	}
}

func TestAppendListMarkdown_NoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "item.md")
	if err := os.WriteFile(path, []byte("Just a body.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := appendListToFile(path, testListA); err != nil {
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
	if len(m.Lists) != 1 || m.Lists[0] != testListA {
		t.Errorf("expected lists=[%s], got %v", testListA, m.Lists)
	}
}

// --- parent config inheritance ---

func TestCreateList_InheritsParentCardTemplate(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")
	mustMkListDir(t, contentDir, "Root")
	parentDir := filepath.Join(contentDir, "music")
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		t.Fatal(err)
	}
	parentYAML := "title: Music\ncardTemplate: embed-card.html\ntypes:\n  - soundcloud\n  - youtube\n"
	if err := os.WriteFile(filepath.Join(parentDir, "node.yaml"), []byte(parentYAML), 0644); err != nil {
		t.Fatal(err)
	}

	if err := createBranchNode(testSiteConfig(contentDir), "music/artist", branchNodeConfig{Title: "Artist"}); err != nil {
		t.Fatalf("createBranchNode: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(contentDir, "music", "artist", "node.yaml"))
	if err != nil {
		t.Fatalf("reading node.yaml: %v", err)
	}
	var got branchNodeConfig
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshalling: %v", err)
	}
	if got.CardTemplate != "embed-card.html" {
		t.Errorf("cardTemplate: want embed-card.html, got %q", got.CardTemplate)
	}
	if len(got.Types) != 2 || got.Types[0] != "soundcloud" || got.Types[1] != "youtube" {
		t.Errorf("types: want [soundcloud youtube], got %v", got.Types)
	}
}

func TestCreateList_ExplicitFlagOverridesParent(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")
	mustMkListDir(t, contentDir, "Root")
	parentDir := filepath.Join(contentDir, "music")
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		t.Fatal(err)
	}
	parentYAML := "title: Music\ncardTemplate: embed-card.html\n"
	if err := os.WriteFile(filepath.Join(parentDir, "node.yaml"), []byte(parentYAML), 0644); err != nil {
		t.Fatal(err)
	}

	lc := branchNodeConfig{Title: "Artist", CardTemplate: "custom-card.html"}
	if err := createBranchNode(testSiteConfig(contentDir), "music/artist", lc); err != nil {
		t.Fatalf("createBranchNode: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(contentDir, "music", "artist", "node.yaml"))
	if err != nil {
		t.Fatalf("reading node.yaml: %v", err)
	}
	var got branchNodeConfig
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshalling: %v", err)
	}
	if got.CardTemplate != "custom-card.html" {
		t.Errorf("cardTemplate: want custom-card.html, got %q", got.CardTemplate)
	}
}

// --- createBranchNode with file item parent ---

func testSiteConfig(contentDir string) *config.SiteConfig {
	return &config.SiteConfig{ContentDir: contentDir}
}

func mustMkListDir(t *testing.T, dir, title string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	data, _ := yaml.Marshal(map[string]any{"title": title})
	if err := os.WriteFile(filepath.Join(dir, "node.yaml"), data, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestCreateList_FileItemParent_CreatesSubListAndUpdatesFile(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")
	mustMkListDir(t, contentDir, "Site")
	if err := os.WriteFile(filepath.Join(contentDir, testItemSlug+".yaml"),
		[]byte("title: "+testItemTitle+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := createBranchNode(testSiteConfig(contentDir), testItemSlug+"/"+testListA, branchNodeConfig{Title: "List A"}); err != nil {
		t.Fatalf("createBranchNode: %v", err)
	}

	if _, err := os.Stat(filepath.Join(contentDir, testItemSlug, testListA, "node.yaml")); err != nil {
		t.Errorf("expected sub-node node.yaml: %v", err)
	}
	meta := readFileItemMeta(filepath.Join(contentDir, testItemSlug+".yaml"))
	if len(meta.Lists) != 1 || meta.Lists[0] != testListA {
		t.Errorf("expected lists=[%s], got %v", testListA, meta.Lists)
	}
}

func TestCreateList_FileItemParent_SiblingDirCreatedWhenMissing(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")
	mustMkListDir(t, contentDir, "Site")
	if err := os.WriteFile(filepath.Join(contentDir, testItemSlug+".yaml"),
		[]byte("title: "+testItemTitle+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := createBranchNode(testSiteConfig(contentDir), testItemSlug+"/"+testListA, branchNodeConfig{Title: "List A"}); err != nil {
		t.Fatalf("createBranchNode: %v", err)
	}
	if _, err := os.Stat(filepath.Join(contentDir, testItemSlug)); err != nil {
		t.Errorf("expected sibling dir to be created: %v", err)
	}
}

func TestCreateList_FileItemParent_SecondSubList(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")
	mustMkListDir(t, contentDir, "Site")
	if err := os.WriteFile(filepath.Join(contentDir, testItemSlug+".yaml"),
		[]byte("title: "+testItemTitle+"\nlists:\n  - "+testListA+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	mustMkListDir(t, filepath.Join(contentDir, testItemSlug, testListA), "List A")

	if err := createBranchNode(testSiteConfig(contentDir), testItemSlug+"/"+testListB, branchNodeConfig{Title: "List B"}); err != nil {
		t.Fatalf("createBranchNode: %v", err)
	}
	meta := readFileItemMeta(filepath.Join(contentDir, testItemSlug+".yaml"))
	if len(meta.Lists) != 2 {
		t.Errorf("expected 2 lists, got %v", meta.Lists)
	}
}
