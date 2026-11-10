//go:build generate

package main

//go:generate go mod download
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -package=mocks -self_package=github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks -destination=tests/mocks/auth.go github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain PasswordHasher,PasswordVerifier,SessionStore,UserRepository
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -package=mocks -self_package=github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks -destination=tests/mocks/repositories.go github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain FilmRatingRepository,CollectionFilmRatingRepository,FilmRepository,CollectionRepository,PersonRepository,ReviewRepository,AvatarRepository,PosterRepository,LibraryRepository,PersonCatalogRepository
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -package=mocks -self_package=github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks -destination=tests/mocks/files.go github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/files AvatarStorage,AvatarWriter,AvatarURLStorage,PosterStorage,PosterFilmReader
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -package=mocks -self_package=github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks -destination=tests/mocks/profile.go github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/auth ProfileReader,ProfilePosterStorage
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -package=mocks -self_package=github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks -destination=tests/mocks/person_catalog.go github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/persons CatalogFilmRepository
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -package=mocks -self_package=github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks -destination=tests/mocks/poster_urls.go github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/films PosterURLStorage
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -package=mocks -self_package=github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks -destination=tests/mocks/router.go github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/router Authenticator,CatalogEditor
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -package=mocks -self_package=github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks -destination=tests/mocks/logger.go github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger Logger
