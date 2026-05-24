# SSG — Static Site Generator

A convention-based static site generator written in Go, built around two primitives: **items** and **lists**.

## Primitives

### Item
An item is any content file (`.md`, `.json`, `.yaml`) in the content directory. It has a data source, a template, and an output path. Items can be anything: a home page, an embedded YouTube video, a blog post, a SoundCloud track.

### List
A list is a directory item — a directory containing a `list.yaml` file. Its children are all the items inside it (files and subdirectories with their own `list.yaml`). A list renders its children as card fragments, sorted and paginated according to its config. A list is itself an item and can be nested.

There is no separate configuration file that registers items. The content tree is scanned recursively at build time.

## Module

`github.com/peacefixation/ssg`

## Structure

```
cmd/             — CLI commands (Cobra)
content/         — Content files; directory structure mirrors output structure
internal/
  config/        — SiteConfig, ItemConfig, DataSourceConfig
  datasource/    — DataSource interface; file and API drivers
  enricher/      — OpenGraph metadata fetching and caching
  renderer/      — Go template renderer with custom functions
  site/          — Build pipeline: scan, build, render, write
  theme/         — Theme loading, asset copying, template partials
  server/        — Development HTTP file server
  watcher/       — File watcher for hot-reload
items/           — Item type definitions (e.g. youtube.yaml, soundcloud.yaml)
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
| `ssg new item` | Add a new item to a list |
| `ssg new list <name>` | Create a new list directory |

### `ssg build`

| Flag | Description |
|---|---|
| `-o, --output` | Output directory (overrides config) |
| `--clean` | Clean output directory before build |
| `--drafts` | Include draft items |
| `--refresh-og` | Bypass OpenGraph cache and re-fetch all items |

### `ssg serve`

| Flag | Description |
|---|---|
| `-p, --port` | Port to serve on (default: 8080) |
| `--watch` | Watch for changes and rebuild automatically |
| `--drafts` | Include draft items |

### `ssg new item`

```bash
ssg new item [--list <list>] [--type <type>] [key=value ...]
```

| Flag | Description |
|---|---|
| `--list` | List to add item to (defaults to root content directory) |
| `--type` | Item type (must match a file in `items/`) |

### `ssg new list`

```bash
ssg new list <name> --title <title> [flags]
```

| Flag | Description |
|---|---|
| `--title` | List title (required) |
| `--types` | Comma-separated allowlist of item types |
| `--template` | Override list page template |
| `--card-template` | Override child card template |
| `--sort-by` | Field to sort children by |
| `--sort-order` | Sort order: `asc` or `desc` |
| `--limit` | Maximum children to render (0 = unlimited) |

## Configuration (`site.yaml`)

```yaml
title: My Site
baseURL: http://localhost:8080
canonicalURL: https://example.com  # used for SEO; overrides baseURL if set
contentDir: content      # scanned recursively for items
outputDir: public        # HTML output
templateDir: templates   # site-specific templates
themesDir: themes
itemsDir: items          # item type definitions
theme: default
sitemap: true            # generate sitemap.xml
ogCacheFile: cache/opengraph.json      # OpenGraph metadata cache
youtubeCacheFile: cache/youtube-channel.json  # YouTube channel metadata cache

server:
  host: localhost
  port: 8080

defaults:
  page:
    template: page.html        # fallback for standalone file items
  list:
    template: list.html        # fallback for directory items
    cardTemplate: item.html    # fallback for child card fragments
    sortBy: date
    sortOrder: desc
    limit: 0                   # 0 = unlimited
```

`list.yaml` and item frontmatter can override `template`, `cardTemplate`, `sortBy`, `sortOrder`, and `limit` per-item.

## Build Pipeline

1. **Scan** — `scanDir` walks `contentDir` recursively. Directories with `list.yaml` become directory items (lists) with their contents as children. Files with supported extensions become leaf items.
2. **Theme** — Theme assets are copied to `output/theme/`; partial templates (`head.html`, `foot.html`) are loaded alongside site templates.
3. **Build** — `buildItem` is called recursively. For each item: fetch data → apply type defaults → build children → sort/limit children → render child cards → inject template vars → write output HTML.
4. **Nav** — Root items are pre-fetched and injected into every page as `RootItems` (filtered to exclude the current page).
5. **Enrich** — OpenGraph metadata is fetched for link items and cached in `ogCacheFile`. Use `--refresh-og` to bypass the cache.

## Template Data

Every template receives these variables:

| Variable | Description |
|---|---|
| `.Site` | Full `SiteConfig` (`.Site.Title`, `.Site.BaseURL`, etc.) |
| `.OutputPath` | Current item's output path, e.g. `music/index.html` |
| `.RootItems` | Slice of root-level nav items (`title`, `outputPath`, `count`) — self excluded |
| `.List` | Slice of `template.HTML` card fragments for child items |
| `.Theme` | `.Theme.CSS` and `.Theme.JS` — root-relative asset URLs |

Plus all fields from the content file itself (e.g. `.title`, `.body`, `.url`, `.embed`).

## Item Types (`items/`)

Item types standardise the fields and build-time defaults for a class of content. Each type is a YAML file in `itemsDir`:

```yaml
# items/youtube.yaml
name: YouTube Video
defaults:
  embed: youtube      # injected into item data if not already set
fields:
  - name: url
    required: true
  - name: title
    required: true
```

At build time, if an item's data contains a `type` key (e.g. `"type": "youtube"`), the corresponding type's `defaults` are merged in — existing item data fields take precedence.

Content files declare which types they accept in `list.yaml`:

```yaml
# content/music/list.yaml
title: Music
types:
  - youtube
  - soundcloud
```

The `ssg new item` command uses the `fields` list to validate required data; the `types` list filters which types are offered for that list.

### Item filename convention

Items are named `{timestamp}-{slug}.json`, e.g. `20260418T120000Z-banco-de-gaia-a-bee-song.json`. The file datasource extracts the timestamp and injects it as `date` if the content does not supply one.

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
