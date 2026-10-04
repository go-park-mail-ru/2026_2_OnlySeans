#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

out_dir=coverage
profile="$out_dir/coverage.out"
report="$out_dir/coverage.html"

mkdir -p "$out_dir"

go test ./tests/... -coverpkg=./internal/...,./router/... -coverprofile="$profile"

go tool cover -html="$profile" -o "$report"

total=$(go tool cover -func="$profile" | awk '/^total:/ { sub("%", "", $NF); print $NF }')

echo
echo "Общее покрытие: ${total}%"
echo "HTML-отчёт: $report"

if [[ -n "${COVERAGE_MIN:-}" ]]; then
	if awk -v total="$total" -v min="$COVERAGE_MIN" 'BEGIN { exit !(total < min) }'; then
		echo "Покрытие ниже порога ${COVERAGE_MIN}%" >&2
		exit 1
	fi
fi
