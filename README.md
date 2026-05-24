# ssg

A convention-based static site generator written in Go. Content lives in files; the directory tree is the site structure.

## Concepts

**Items** are content files (`.md`, `.yaml`) in the content directory. Each item has a data source, a template, and an output path. Items can be anything: a home page, a blog post, an embedded video, a link.

**Lists** are directories containing a `list.yaml` file. A list renders its children as cards, sorted and paginated according to its config. Lists can be nested.

## Getting started

Initialise a new site skeleton in the current directory:

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
itemsDir: items
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

## Creating a list

```bash
ssg new list music --title "Music" --types youtube,soundcloud
```

Or interactively:

```bash
ssg new
```

This creates `content/music/list.yaml`:

```yaml
title: Music
types:
  - youtube
  - soundcloud
```

The `types` field restricts which item types can be added to the list. Leave it out to allow all types. Lists can be nested using path arguments: `ssg new list music/live --title "Live Sets"`.

## Adding items

```bash
ssg new item --list music --type youtube url=https://youtu.be/xyz title="Banco de Gaia"
```

Or run `ssg new` with no arguments for an interactive prompt.

Fields are supplied as `key=value` arguments. Required fields are defined by the item type — missing required fields produce an error.

Items are written as timestamped files named `{timestamp}-{slug}.{ext}`, e.g. `20260418T120000Z-banco-de-gaia.yaml`. The format depends on the item type:

- Most types → `.yaml` file
- Types with `format: markdown` → `.md` file with YAML frontmatter and an empty body for prose content

## Item types

Item type definitions live in `items/`:

```yaml
# items/youtube.yaml
name: YouTube Video
defaults:
  embed: youtube
fields:
  - name: url
    required: true
  - name: title
    required: true
```

At build time, `defaults` are merged into item data (item fields take precedence).

To make an item type produce a markdown file with YAML frontmatter, add `format: markdown`:

```yaml
# items/post.yaml
name: Post
format: markdown
fields:
  - name: title
    required: true
  - name: tags
    required: true
```

This produces a `.md` file with a YAML frontmatter block and an empty body for the post content.

## YouTube channels

The `youtube-channel` item type fetches channel metadata and the latest video from the YouTube Data API v3 and renders them as a rich page and card.

### Prerequisites

Create a Google Cloud project, enable the YouTube Data API v3, and generate an API key. Set it in your environment before running a build:

```bash
export YOUTUBE_DATA_API_KEY=your_api_key_here
```

Add the variable to a `.env` file to make it persistent:

```
YOUTUBE_DATA_API_KEY=your_api_key_here
```

Then source it before building: `export $(cat .env | xargs)`.

### Create a channels list

```bash
ssg new list channels --title "Channels" --types youtube-channel
```

### Add a channel

```bash
ssg new item --list channels --type youtube-channel channelId=UCxxxxxxxxxxxxxxxxxxxxxx title="Channel Name"
```

Or write the YAML directly:

```yaml
# content/channels/20260524T120000Z-my-channel.yaml
type: youtube-channel
channelId: UCxxxxxxxxxxxxxxxxxxxxxx
title: My Channel   # fallback title if the API call fails
```

At build time the following fields are fetched and injected into the item's template data:

| Field | Description |
|---|---|
| `yt_channel_title` | Channel display name |
| `yt_description` | Channel about text |
| `yt_thumbnail` | Channel thumbnail URL |
| `yt_subscriber_count` | Subscriber count (string) |
| `yt_latest_video_id` | Latest video ID |
| `yt_latest_video_title` | Latest video title |

### Caching

Fetched data is cached in `youtube-cache.json` (configurable via `youtubeCacheFile` in `site.yaml`). Subsequent builds use the cache and make no API calls. Commit this file to avoid hitting API quota on every build.

To re-fetch all channels:

```bash
ssg build --refresh-yt
```

To re-fetch a single channel, add `yt_refresh: true` to its content file and remove it after the next build.

## Custom templates

Any item can override its template by setting a `template` field in its frontmatter or YAML data:

```yaml
# content/blog/20260517T053555Z-my-post.md
---
title: My Post
tags: Go
template: blog-post.html
---

Post body here.
```

The named template must exist in `templateDir`.

## CLI reference

| Command | Description |
|---|---|
| `ssg init <name>` | Scaffold a new site skeleton |
| `ssg build [--clean]` | Build the site to `outputDir` |
| `ssg serve [--watch]` | Serve locally; `--watch` hot-reloads on changes |
| `ssg new` | Interactively create a list or item |
| `ssg new list <name> --title <title> [flags]` | Create a new list |
| `ssg new item --list <list> --type <type> [key=value ...]` | Add a new item to a list |
