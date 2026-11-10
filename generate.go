//go:build generate

package main

//go:generate go mod download
//go:generate -command mockgen go run go.uber.org/mock/mockgen@v0.6.0
//go:generate mockgen -package=mocks -self_package=github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks -destination=tests/mocks/auth.go github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain PasswordHasher,PasswordVerifier,SessionStore,UserRepository
//go:generate mockgen -package=mocks -self_package=github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks -destination=tests/mocks/repositories.go github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain FilmRatingRepository,CollectionFilmRatingRepository,FilmRepository,CollectionRepository,PersonRepository,ReviewRepository,AvatarRepository,PosterRepository,LibraryRepository,PersonCatalogRepository
//go:generate mockgen -package=mocks -self_package=github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks -destination=tests/mocks/files.go github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/files AvatarStorage,AvatarWriter,AvatarURLStorage,PosterStorage,PosterFilmReader
//go:generate mockgen -package=mocks -self_package=github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks -destination=tests/mocks/profile.go github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/auth ProfileReader,ProfilePosterStorage
//go:generate mockgen -package=mocks -self_package=github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks -destination=tests/mocks/person_catalog.go github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/persons CatalogFilmRepository
//go:generate mockgen -package=mocks -self_package=github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks -destination=tests/mocks/poster_urls.go github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/films PosterURLStorage
//go:generate mockgen -package=mocks -self_package=github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks -destination=tests/mocks/router.go github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/router Authenticator,CatalogEditor
//go:generate mockgen -package=mocks -self_package=github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks -destination=tests/mocks/logger.go github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger Logger
//go:generate go run generate.go

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	paths, err := filepath.Glob("tests/mocks/*.go")
	if err != nil {
		fail(err)
	}
	for _, path := range paths {
		source, err := os.ReadFile(path)
		if err != nil {
			fail(err)
		}
		lines := strings.Split(string(source), "\n")
		result := make([]string, 0, len(lines))
		for _, line := range lines {
			if !strings.HasPrefix(strings.TrimSpace(line), "//") {
				result = append(result, line)
			}
		}
		formatted, err := format.Source([]byte(strings.Join(result, "\n")))
		if err != nil {
			fail(err)
		}
		if err := os.WriteFile(path, formatted, 0o644); err != nil {
			fail(err)
		}
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
