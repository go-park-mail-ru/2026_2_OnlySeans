//go:build generate

package main

//go:generate -command mockgen go run go.uber.org/mock/mockgen@v0.6.0
//go:generate mockgen -source=services/persons/internal/domain/person_repository.go -destination=tests/mocks/persons/repository.go -package=mocks
//go:generate mockgen -source=services/persons/internal/usecase/persons.go -destination=tests/mocks/persons/usecase.go -package=mocks
//go:generate mockgen -source=services/persons/internal/domain/person_catalog.go -destination=tests/mocks/persons/catalog_repository.go -package=mocks
//go:generate mockgen -source=services/persons/internal/usecase/catalog.go -destination=tests/mocks/persons/catalog_usecase.go -package=mocks
//go:generate mockgen -source=shared/pkg/logger/logger.go -destination=tests/mocks/shared/logger.go -package=mocks
//go:generate mockgen -source=services/user/internal/repository/profile.go -destination=tests/mocks/user/profile_reader.go -package=mocks
//go:generate mockgen -source=services/user/internal/usecase/profile.go -destination=tests/mocks/user/profile.go -package=mocks
//go:generate mockgen -source=services/user/internal/usecase/ports.go -destination=tests/mocks/user/ports.go -package=mocks
//go:generate mockgen -source=services/auth/internal/usecase/auth.go -destination=tests/mocks/auth/auth.go -package=mocks
//go:generate mockgen -source=services/auth/internal/usecase/session.go -destination=tests/mocks/auth/session.go -package=mocks
//go:generate mockgen -source=services/auth/internal/domain/session.go -destination=tests/mocks/auth/password.go -package=mocks -exclude_interfaces=PasswordHasher
