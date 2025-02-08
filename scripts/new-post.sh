#! /usr/bin/env bash

set -euo pipefail

scriptDir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
frontmatterScript="$scriptDir/new-frontmatter.sh"
contentDir="$scriptDir/../content/posts"

echo "Create a new post (press enter to use default values):"

# Title
defaultTitle="New Post"
read -p "Enter the title [$defaultTitle]: " title
title=${title:-$defaultTitle}

# Date
postDate=$(date -u --rfc-3339=seconds | sed 's/ /T/')

# Tags
defaultTags=""
read -p "Enter tags (comma separated) [$defaultTags]: " tags
tags=${tags:-$defaultTags}

# Description
defaultDescription=""
read -p "Enter description [$defaultDescription]: " description
description=${description:-$defaultDescription}

# Draft
defaultDraft="y"
read -p "Is this a draft? (y/n) [$defaultDraft]: " draft
draft=${draft:-$defaultDraft}

slug=$(echo -n "$title" | tr '[:upper:]' '[:lower:]' | tr '[:space:]' '-' | tr -cd '[:alnum:]-')

filename="${contentDir}/${postDate}_${slug}.md"
touch "$filename"

"$frontmatterScript" "$title" "$postDate" "$tags" "$description" "$draft" > "$filename"

echo "Created new post: $filename"
