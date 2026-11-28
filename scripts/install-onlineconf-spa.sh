#!/usr/bin/env bash
set -euo pipefail

source_dir=${1:?Pass the administrative SPA dist directory}
public_root=${2:-/var/www/onlyseans-onlineconf}

if [[ ! -f "$source_dir/onlineconf/index.html" || ! -d "$source_dir/assets/onlineconf" ]]; then
    echo 'Expected onlineconf/index.html and assets/onlineconf in the SPA dist directory.' >&2
    exit 1
fi

mkdir -p "$public_root/onlineconf" "$public_root/assets/onlineconf"
cp -R "$source_dir/assets/onlineconf/." "$public_root/assets/onlineconf/"
staged_html=$(mktemp "$public_root/onlineconf/.index.XXXXXX")
trap 'rm -f "$staged_html"' EXIT
cp "$source_dir/onlineconf/index.html" "$staged_html"
chmod 644 "$staged_html"
mv "$staged_html" "$public_root/onlineconf/index.html"
printf 'Installed administrative SPA in %s\n' "$public_root"
