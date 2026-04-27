# ssg

A convention-based static site generator written in Go. Content lives in files; the directory tree is the site structure.

## Concepts

**Items** are content files (`.md`, `.json`, `.yaml`) in the content directory. Each item has a data source, a template, and an output path.

**Lists** are directories containing a `list.yaml` file. A list renders its children as cards, sorted and paginated according to its config. Lists can be nested.

## Getting started

Initialise a new site in the current directory:

```bash
ssg new mysite
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
theme: default

defaults:
  page:
    template: page.html
  list:
    template: list.html
    cardTemplate: item.html
    sortBy: date
    sortOrder: desc
    limit: 0        # 0 = unlimited
```

## Adding a list

Create a directory under `content/` with a `list.yaml`:

```yaml
# content/music/list.yaml
title: Music
types:
  - youtube
  - soundcloud
```

The `types` field restricts which item types can be added to this list, and corresponds to definition files in the `items/` directory.

## Adding items

Use `ssg add` to add a new item to a list:

```bash
ssg add --list music --type youtube url=https://youtu.be/xyz title="Banco de Gaia"
```

Required fields are defined by the item type. Missing required fields produce an error. Items are written as JSON files named `{timestamp}-{slug}.json`.

## Item types

Item type definitions live in `items/`:

```yaml
# items/youtube.yaml
name: YouTube Video
defaults:
  platform: youtube
fields:
  - name: url
    required: true
  - name: title
    required: true
```

At build time, `defaults` are merged into item data (item fields take precedence).

## CLI reference

| Command | Description |
|---|---|
| `ssg new <name>` | Scaffold a new site in the current directory |
| `ssg build [--clean]` | Build the site to `outputDir` |
| `ssg serve [--watch]` | Serve locally; `--watch` hot-reloads on changes |
| `ssg add --list <list> --type <type> [key=value ...]` | Add a new item to a list |
