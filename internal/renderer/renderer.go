package renderer

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9-]+`)

// Renderer holds eagerly-parsed templates and renders items and lists.
type Renderer struct {
	tmpl *template.Template
}

// New parses all *.html templates found recursively under templateDir, then
// under themeTemplateDir (if non-empty). Theme partials are loaded second so
// they are available to site templates via {{template "head.html" .}} etc.
func New(templateDir, themeTemplateDir string) (*Renderer, error) {
	// tmpl is declared first so the render closure can reference it after parsing.
	var tmpl *template.Template

	funcs := template.FuncMap{
		// render executes a named template and returns safe HTML, enabling
		// dynamic dispatch (e.g. render "platform/youtube.html" .).
		"render": func(name string, data any) (template.HTML, error) {
			var buf strings.Builder
			if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
				return "", fmt.Errorf("rendering %q: %w", name, err)
			}
			return template.HTML(buf.String()), nil //nolint:gosec // rendered by our own templates
		},
		// youtubeID extracts the video ID from a YouTube watch URL.
		"youtubeID": func(rawURL string) string {
			u, err := url.Parse(rawURL)
			if err != nil {
				return ""
			}
			return u.Query().Get("v")
		},
		// tagSlug converts a tag name to the URL slug used by the tags pipeline.
		"tagSlug": func(tag string) string {
			slug := strings.ToLower(tag)
			slug = strings.ReplaceAll(slug, " ", "-")
			slug = nonAlphaNum.ReplaceAllString(slug, "")
			return strings.Trim(slug, "-")
		},
		// shortCount formats a numeric string with K/M suffix (e.g. "1234567" → "1.2M").
		"shortCount": func(s string) string {
			n, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return s
			}
			switch {
			case n >= 1_000_000:
				return strconv.FormatFloat(float64(n)/1_000_000, 'f', 1, 64) + "M"
			case n >= 1_000:
				return strconv.FormatFloat(float64(n)/1_000, 'f', 1, 64) + "K"
			default:
				return strconv.FormatInt(n, 10)
			}
		},
	}

	tmpl = template.New("").Funcs(funcs)

	for _, dir := range []string{templateDir, themeTemplateDir} {
		if dir == "" {
			continue
		}
		if err := parseDir(tmpl, dir); err != nil {
			return nil, err
		}
	}

	return &Renderer{tmpl: tmpl}, nil
}

func parseDir(tmpl *template.Template, dir string) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".html" {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading template %q: %w", path, err)
		}
		if _, err := tmpl.Parse(string(content)); err != nil {
			return fmt.Errorf("parsing template %q: %w", path, err)
		}
		return nil
	})
}

// Render executes the named template with data, writing output to w.
func (r *Renderer) Render(w io.Writer, templateName string, data map[string]any) error {
	if err := r.tmpl.ExecuteTemplate(w, templateName, data); err != nil {
		return fmt.Errorf("rendering template %q: %w", templateName, err)
	}
	return nil
}

// RenderFragment executes the named template with data and returns the output
// as a safe HTML fragment.
func (r *Renderer) RenderFragment(templateName string, data map[string]any) (template.HTML, error) {
	var buf strings.Builder
	if err := r.tmpl.ExecuteTemplate(&buf, templateName, data); err != nil {
		return "", fmt.Errorf("rendering template %q: %w", templateName, err)
	}
	return template.HTML(buf.String()), nil //nolint:gosec // rendered by our own templates
}
