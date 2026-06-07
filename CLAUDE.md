# SSG — Static Site Generator

A convention-based static site generator written in Go, built around a single primitive: the **node**.

## Primitives

### Node

A node is any content file (`.md`, `.json`, `.yaml`) or directory in the content tree. It has a data source, a template, and an output path.

A **leaf node** is a content file. It has no children and can be anything: a home page, an embedded YouTube video, a blog post, a SoundCloud track.

A **branch node** is a directory containing a `node.yaml` file. Its children are all the nodes inside it (leaf nodes and nested branch nodes). A branch node renders its children as card fragments, sorted and paginated according to its config. Branch nodes can be nested.

There is no separate configuration file that registers nodes. The content tree is scanned recursively at build time.

## Module

`github.com/peacefixation/ssg`

## Structure

```
cmd/             — CLI commands (Cobra)
content/         — Content files; directory structure mirrors output structure
internal/
  config/        — SiteConfig, NodeConfig, DataSourceConfig
  datasource/    — DataSource interface; file and API drivers
  enricher/      — OpenGraph metadata fetching and caching
  renderer/      — Go template renderer with custom functions
  site/          — Build pipeline: scan, build, render, write
  theme/         — Theme loading, asset copying, template partials
  server/        — Development HTTP file server
  watcher/       — File watcher for hot-reload
types/           — Node type definitions (e.g. youtube.yaml, soundcloud.yaml)
templates/       — Site-specific HTML templates
themes/          — Theme directories (CSS, JS, partial templates)
site.yaml        — Site configuration
```

## Development

```bash
go run .
go build .
go test ./...
```

## CLI Commands

| Command | Description |
|---|---|
| `ssg build` | Build the site to `outputDir` |
| `ssg serve` | Build and serve locally |
| `ssg init <name>` | Scaffold a new site skeleton |
| `ssg new node` | Create a leaf node (content file) or branch node (directory) |

### `ssg build`

| Flag | Description |
|---|---|
| `-o, --output` | Output directory (overrides config) |
| `--clean` | Clean output directory before build |
| `--drafts` | Include draft nodes |
| `--refresh-og` | Bypass OpenGraph cache and re-fetch all nodes |

### `ssg serve`

| Flag | Description |
|---|---|
| `-p, --port` | Port to serve on (default: 8080) |
| `--watch` | Watch for changes and rebuild automatically |
| `--drafts` | Include draft nodes |

### `ssg new node`

Without a positional argument, creates a leaf node (content file):

```bash
ssg new node [--parent <node>] [--type <type>] [key=value ...]
```

With a positional argument `<name>` and `--title`, creates a branch node (directory + `node.yaml`):

```bash
ssg new node <name> --title <title> [flags]
```

| Flag | Description |
|---|---|
| `--parent` | Parent node to add the leaf node to (defaults to root content directory) |
| `--type` | Node type for leaf node (must match a file in `types/`) |
| `--title` | Branch node title (required when creating a branch) |
| `--types` | Comma-separated node type allowlist (branch nodes) |
| `--template` | Override page template |
| `--card-template` | Override child card template |
| `--sort-by` | Field to sort children by |
| `--sort-order` | Sort order: `asc` or `desc` |
| `--limit` | Maximum children to render (0 = unlimited) |

## Configuration (`site.yaml`)

```yaml
title: My Site
baseURL: http://localhost:8080
canonicalURL: https://example.com  # used for SEO; overrides baseURL if set
contentDir: content      # scanned recursively for nodes
outputDir: public        # HTML output
templateDir: templates   # site-specific templates
themesDir: themes
typesDir: types          # node type definitions
theme: default
sitemap: true            # generate sitemap.xml
ogCacheFile: cache/opengraph.json      # OpenGraph metadata cache
youtubeCacheFile: cache/youtube-channel.json  # YouTube channel metadata cache

server:
  host: localhost
  port: 8080

defaults:
  template: node.html        # fallback template for all nodes
  cardTemplate: card.html    # fallback for child card fragments
  sortBy: date
  sortOrder: desc
  limit: 0                   # 0 = unlimited
```

`node.yaml` and node frontmatter can override `template`, `cardTemplate`, `sortBy`, `sortOrder`, and `limit` per-node.

## Build Pipeline

1. **Scan** — `scanDir` walks `contentDir` recursively. Directories with `node.yaml` become branch nodes with their contents as children. Files with supported extensions become leaf nodes.
2. **Theme** — Theme assets are copied to `output/theme/`; partial templates (`head.html`, `foot.html`) are loaded alongside site templates.
3. **Build** — Each node is built recursively: fetch data → apply type defaults → build children → sort/limit children → render child cards → inject template vars → write output HTML.
4. **Nav** — Root nodes are pre-fetched and injected into every page as `RootItems` (filtered to exclude the current page).
5. **Enrich** — OpenGraph metadata is fetched for link nodes and cached in `ogCacheFile`. Use `--refresh-og` to bypass the cache.

## Template Data

Every template receives these variables:

| Variable | Description |
|---|---|
| `.Site` | Full `SiteConfig` (`.Site.Title`, `.Site.BaseURL`, etc.) |
| `.OutputPath` | Current node's output path, e.g. `music/index.html` |
| `.RootItems` | Slice of root-level nav nodes (`title`, `outputPath`, `count`) — self excluded |
| `.Children` | Slice of `template.HTML` card fragments for child nodes |
| `.Theme` | `.Theme.CSS` and `.Theme.JS` — root-relative asset URLs |

Plus all fields from the content file itself (e.g. `.title`, `.body`, `.url`, `.embed`).

## Node Types (`types/`)

Node types standardise the fields and build-time defaults for a class of content. Each type is a YAML file in `typesDir`:

```yaml
# types/youtube.yaml
name: YouTube Video
defaults:
  embed: youtube      # injected into node data if not already set
fields:
  - name: url
    required: true
  - name: title
    required: true
```

At build time, if a node's data contains a `type` key (e.g. `"type": "youtube"`), the corresponding type's `defaults` are merged in — existing node data fields take precedence.

Branch nodes declare which types they accept in `node.yaml`:

```yaml
# content/music/node.yaml
title: Music
types:
  - youtube
  - soundcloud
```

The `ssg new node` command uses the `fields` list to validate required data; the `types` list filters which types are offered for that branch node.

### Node filename convention

Leaf nodes are named `{timestamp}-{slug}.json`, e.g. `20260418T120000Z-banco-de-gaia-a-bee-song.json`. The file datasource extracts the timestamp and injects it as `date` if the content does not supply one.

## Themes (`themes/`)

```
themes/default/
  theme.yaml              # name, css: [...], js: [...]
  style.css               # copied to public/theme/style.css
  templates/
    head.html             # {{define "head.html"}} — HTML head, CSS injection
    foot.html             # {{define "foot.html"}} — closing tags, JS injection
```

Theme partials are loaded into the same template set as site templates. Every page template calls `{{template "head.html" .}}` and `{{template "foot.html" .}}`.

## Embed Templates

Embed templates live in `templates/embed/`. A card template dispatches to the correct one using the `render` custom function:

```html
{{render (printf "embed/%s.html" .embed) .}}
```

`render` calls `tmpl.ExecuteTemplate` at runtime, enabling dynamic dispatch without Go template's static `{{template}}` limitation.

Current embeds: `youtube`, `soundcloud`.

## Frameworks

- **Cobra** — CLI
- **Viper** — Configuration
- **Blackfriday** — Markdown parsing
