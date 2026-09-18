#!/usr/bin/env bash
# Inject YAML frontmatter into Terraform provider doc files that lack it.
# The upjet scraper expects page_title and description in frontmatter.
set -euo pipefail

docs_dir="${1:?Usage: $0 <docs-directory>}"

for f in "$docs_dir"/*.md; do
  [ -f "$f" ] || continue

  # Skip files that already have frontmatter
  if head -1 "$f" | grep -q '^---$'; then
    continue
  fi

  basename="$(basename "$f" .md)"
  resource="fusionauth_${basename}"

  # Extract the first H1 heading as the description
  description="$(grep -m1 '^# ' "$f" | sed 's/^# //')"
  [ -z "$description" ] && description="$resource"

  # Prepend frontmatter
  tmp="$(mktemp)"
  {
    echo '---'
    echo "page_title: \"${resource} Resource - fusionauth\""
    echo 'description: |-'
    echo "  ${description}"
    echo '---'
    echo ''
    cat "$f"
  } > "$tmp"
  mv "$tmp" "$f"
done
