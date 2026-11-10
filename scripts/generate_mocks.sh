#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
version=$(go list -m -f '{{.Version}}' go.uber.org/mock)
module=$(go list -m -f '{{.Path}}')
mkdir -p tests/mocks
output=$(mktemp)
trap 'rm -f "$output"' EXIT

generate() {
    local filename=$1 package=$2 interfaces=$3
    go run "go.uber.org/mock/mockgen@$version" -package=mocks -self_package="$module/tests/mocks" -destination="$output" "$package" "$interfaces"
    sed '/^[[:space:]]*\/\//d' "$output" > "tests/mocks/$filename.go"
    gofmt -w "tests/mocks/$filename.go"
}

generate auth "$module/internal/domain" 'PasswordHasher,PasswordVerifier,SessionStore,UserRepository'
generate repositories "$module/internal/domain" 'FilmRatingRepository,CollectionFilmRatingRepository,FilmRepository,CollectionRepository,PersonRepository,ReviewRepository,AvatarRepository,PosterRepository,LibraryRepository,PersonCatalogRepository'
generate files "$module/internal/usecase/files" 'AvatarStorage,AvatarWriter,AvatarURLStorage,AccountAvatarRepository,PosterStorage,PosterFilmReader'
generate profile "$module/internal/usecase/auth" 'ProfileReader,ProfilePosterStorage'
generate person_catalog "$module/internal/usecase/persons" CatalogFilmRepository
generate poster_urls "$module/internal/services/films" PosterURLStorage
generate router "$module/internal/delivery/router" 'Authenticator,CatalogEditor'

generate logger "$module/pkg/logger" Logger
