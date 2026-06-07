# ssg

A convention-based static site generator written in Go. Content lives in files; the directory tree is the site structure.

## Concepts

**Nodes** are the single primitive. A node has a data source, a template, and an output path.

**Leaf nodes** are content files (`.md`, `.yaml`, `.json`) in the content directory. They have no children and can be anything: a home page, a blog post, an embedded video, a link.

**Branch nodes** are directories containing a `node.yaml` file. A branch node renders its children as cards, sorted and paginated according to its config. Branch nodes can be nested.

## Getting started

Initialise a new site skeleton:

```bash
ssg init mysite
```

This scaffolds:

```
site.yaml            — site configuration
content/index.md     — home page content
templates/index.html — home page template
public/              — build output (empty)
```

Build and serve locally:

```bash
ssg serve --watch
```

## Configuration

`site.yaml` controls the top-level settings:

```yaml
title: My Site
baseURL: http://localhost:8080
contentDir: content
outputDir: public
templateDir: templates
themesDir: themes
typesDir: types
theme: default

defaults:
  template: node.html
  cardTemplate: card.html
  sortBy: date
  sortOrder: desc
  limit: 0        # 0 = unlimited
```

## Creating a branch node

```bash
ssg new node music --title "Music" --types youtube,soundcloud
```

This creates `content/music/node.yaml`:

```yaml
title: Music
types:
  - youtube
  - soundcloud
```

The `types` field restricts which node types can be added. Leave it out to allow all types. Branch nodes can be nested using path arguments: `ssg new node music/live --title "Live Sets"`.

## Adding a leaf node

```bash
ssg new node --parent music --type youtube url=https://youtu.be/xyz title="Banco de Gaia"
```

Fields are supplied as `key=value` arguments. Required fields are defined by the node type — missing required fields produce an error.

Leaf nodes are written as timestamped files named `{timestamp}-{slug}.{ext}`, e.g. `20260418T120000Z-banco-de-gaia.yaml`. The format depends on the node type:

- Most types → `.yaml` file
- Types with `format: markdown` → `.md` file with YAML frontmatter and an empty body for prose content

## Node types

Node type definitions live in `types/`:

```yaml
# types/youtube.yaml
name: YouTube Video
defaults:
  embed: youtube
fields:
  - name: url
    required: true
  - name: title
    required: true
```

At build time, `defaults` are merged into node data (node fields take precedence).

To make a node type produce a markdown file with YAML frontmatter, add `format: markdown`:

```yaml
# types/post.yaml
name: Post
format: markdown
fields:
  - name: title
    required: true
  - name: tags
    required: true
```

## YouTube channels

The `youtube-channel` node type fetches channel metadata and the latest video from the YouTube Data API v3.

### Prerequisites

Create a Google Cloud project, enable the YouTube Data API v3, and generate an API key:

```bash
export YOUTUBE_DATA_API_KEY=your_api_key_here
```

### Create a channels branch node

```bash
ssg new node channels --title "Channels" --types youtube-channel
```

### Add a channel

```bash
ssg new node --parent channels --type youtube-channel channelId=UCxxxxxxxxxxxxxxxxxxxxxx title="Channel Name"
```

Or write the YAML directly:

```yaml
# content/channels/20260524T120000Z-my-channel.yaml
type: youtube-channel
channelId: UCxxxxxxxxxxxxxxxxxxxxxx
title: My Channel
```

At build time the following fields are fetched and injected into the node's template data:

| Field | Description |
|---|---|
| `yt_channel_title` | Channel display name |
| `yt_description` | Channel about text |
| `yt_thumbnail` | Channel thumbnail URL |
| `yt_subscriber_count` | Subscriber count (string) |
| `yt_latest_video_id` | Latest video ID |
| `yt_latest_video_title` | Latest video title |

Fetched data is cached in `youtubeCacheFile` (configurable in `site.yaml`). Commit this file to avoid hitting API quota on every build.

## Custom templates

Any node can override its template by setting a `template` field in its frontmatter or YAML data:

```yaml
# content/blog/20260517T053555Z-my-post.md
---
title: My Post
tags: Go
template: blog-post.html
---

Post body here.
```

## CLI reference

| Command | Description |
|---|---|
| `ssg init <name>` | Scaffold a new site skeleton |
| `ssg build [--clean]` | Build the site to `outputDir` |
| `ssg serve [--watch]` | Serve locally; `--watch` hot-reloads on changes |
| `ssg new node <name> --title <title> [flags]` | Create a branch node |
| `ssg new node [--parent <node>] [--type <type>] [key=value ...]` | Create a leaf node |
