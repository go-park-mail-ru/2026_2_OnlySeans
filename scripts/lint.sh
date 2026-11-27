#!/bin/sh
set -eu

unformatted="$(gofmt -l .)"
if [ -n "$unformatted" ]; then
	printf '%s\n' "Unformatted files:" "$unformatted"
	exit 1
fi

cache_root="${TMPDIR:-${TEMP:-/tmp}}/onlyseans-go-build"
mkdir -p "$cache_root"

GOCACHE="$cache_root" GOWORK="$(pwd)/go.work" go vet \
	./api-gateway/... \
	./services/... \
	./shared/... \
	./tests/mocks/auth/... \
	./tests/mocks/user/... \
	./tests/mocks/persons/... \
	./tests/mocks/shared/... \
	./tests/api-gateway/... \
	./tests/auth/... \
	./tests/catalog/... \
	./tests/persons/... \
	./tests/user/... \
	./tests/shared/...
