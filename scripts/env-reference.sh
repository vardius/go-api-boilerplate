#!/usr/bin/env bash
# Rewrites everything after the "<!-- generated -->" line of the website's
# environment variables page from each service's `env` command, once per
# store build, so the page can't drift from the config structs.
set -euo pipefail

page=website/src/content/docs/reference/environment-variables.md
marker='<!-- generated -->'
grep -qxF -- "$marker" "$page" || { echo "$page has no '$marker' line" >&2; exit 1; }

table() { go run -tags "$2" "./cmd/$1" env; }

{
  sed "/^${marker}\$/q" "$page"
  for svc in auth user; do
    base=$(table "$svc" "")
    printf '\n## %s\n\n%s\n' "$svc" "$base"
    for store in mysql:MySQL mongodb:MongoDB; do
      tag=persistence_${store%%:*}
      # The rows the store build adds to the memory build's table.
      extra=$(grep -vxF -f <(printf '%s\n' "$base") <(table "$svc" "$tag") || true)
      printf '\n### %s with the %s store\n\nBuilt with `-tags %s`, it also reads:\n\n%s\n%s\n' \
        "$svc" "${store#*:}" "$tag" "$(printf '%s\n' "$base" | head -2)" "$extra"
    done
  done
} > "$page.tmp"
mv "$page.tmp" "$page"
