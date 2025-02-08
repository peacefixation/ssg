#!/usr/bin/env bash

function usage() {
    echo "Usage: $0 \"title\" \"date\" \"tags\" \"description\" \"draft (true/false)\""
    exit 1
}

# check for 5 parameters
if [ "$#" -ne 5 ]; then
  usage
fi

title="$1"
date="$2"
tags="$3"
description="$4"
if [ "$5" == "y" ]; then
  draft="true"
else
  draft="false"
fi

cat <<EOF
---
title: $title
date: $date
tags: [$tags]
description: $description
draft: $draft
---
EOF
