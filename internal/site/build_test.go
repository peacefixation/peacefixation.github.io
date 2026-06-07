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
		TypesDir:    filepath.Join(dir, "types"),
		Defaults: config.Defaults{
			Template:     "item.html",
			CardTemplate: "card.html",
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

func TestBuild_NodeYaml_BackwardsCompat(t *testing.T) {
	dir := t.TempDir()

	mustWriteFile(t, filepath.Join(dir, "templates", "item.html"),
		`{{define "item.html"}}{{.title}}{{end}}`)
	mustWriteFile(t, filepath.Join(dir, "templates", "card.html"),
		`{{define "card.html"}}{{.title}}{{end}}`)

	// A directory with the new node.yaml convention.
	mustWriteFile(t, filepath.Join(dir, "content", "music", "node.yaml"),
		"title: Music\n")
	mustWriteFile(t, filepath.Join(dir, "content", "music", "20260501T000000Z-track.yaml"),
		"title: Track\n")

	buildTestSite(t, dir)

	out := filepath.Join(dir, "output")
	assertFile(t, out, "music/index.html")
	assertFile(t, out, "music/20260501T000000Z-track/index.html")
}

func TestBuild_ListYaml_BackwardsCompat(t *testing.T) {
	dir := t.TempDir()

	mustWriteFile(t, filepath.Join(dir, "templates", "item.html"),
		`{{define "item.html"}}{{.title}}{{end}}`)
	mustWriteFile(t, filepath.Join(dir, "templates", "card.html"),
		`{{define "card.html"}}{{.title}}{{end}}`)

	mustWriteFile(t, filepath.Join(dir, "content", "music", "node.yaml"),
		"title: Music\n")
	mustWriteFile(t, filepath.Join(dir, "content", "music", "20260501T000000Z-track.yaml"),
		"title: Track\n")

	buildTestSite(t, dir)

	out := filepath.Join(dir, "output")
	assertFile(t, out, "music/index.html")
	assertFile(t, out, "music/20260501T000000Z-track/index.html")
}

func TestBuild_SidecarNodeYaml_GivesFileItemChildren(t *testing.T) {
	dir := t.TempDir()

	mustWriteFile(t, filepath.Join(dir, "templates", "item.html"),
		`{{define "item.html"}}{{.title}}{{range .Children}}{{.}}{{end}}{{end}}`)
	mustWriteFile(t, filepath.Join(dir, "templates", "card.html"),
		`{{define "card.html"}}{{.title}}{{end}}`)

	// File item with a sidecar node.yaml.
	mustWriteFile(t, filepath.Join(dir, "content", "20260418T120000Z-event.yaml"),
		"title: Event\n")
	mustWriteFile(t, filepath.Join(dir, "content", "20260418T120000Z-event.node.yaml"),
		"cardTemplate: card.html\nsortBy: date\n")

	// Child directory discovered from the sibling dir.
	mustWriteFile(t, filepath.Join(dir, "content", "20260418T120000Z-event", "photos", "node.yaml"),
		"title: Photos\n")
	mustWriteFile(t, filepath.Join(dir, "content", "20260418T120000Z-event", "photos", "20260501T000000Z-photo.yaml"),
		"title: Photo 1\n")

	buildTestSite(t, dir)

	out := filepath.Join(dir, "output")
	assertFile(t, out, "20260418T120000Z-event/index.html")
	assertFile(t, out, "20260418T120000Z-event/photos/index.html")
	assertFile(t, out, "20260418T120000Z-event/photos/20260501T000000Z-photo/index.html")
}
