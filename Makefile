APP_PACKAGES := ./api-gateway/... ./services/... ./shared/...
MOCK_PACKAGES := ./tests/mocks/catalog/... ./tests/mocks/auth/... ./tests/mocks/user/... ./tests/mocks/persons/... ./tests/mocks/shared/...
TEST_PACKAGES := $(MOCK_PACKAGES) ./tests/api-gateway/... ./tests/auth/... ./tests/catalog/... ./tests/persons/... ./tests/user/... ./tests/shared/...
INTEGRATION_PACKAGES := ./tests/api-gateway/integration/... ./tests/catalog/integration/... ./tests/persons ./tests/migrations/...
TEST_WORKSPACE := $(CURDIR)/go.work

.PHONY: lint fmt test test-integration test-integration-compile cover build run check test-api mocks api-gen

# Команды для проверки качества кода
lint:
	@test -z "$$(gofmt -l .)" || (echo "Неотформатированные файлы:"; gofmt -l .; exit 1)
	GOWORK=$(TEST_WORKSPACE) go vet $(APP_PACKAGES) $(TEST_PACKAGES)

# Автоформатирование
fmt:
	gofmt -w .

# Запуск тестов (с проверкой гонок данных)
test: test-integration-compile
	GOWORK=$(TEST_WORKSPACE) go test -race $(APP_PACKAGES) $(TEST_PACKAGES)

# Компиляция интеграционных тестов обязательна и не требует запущенных зависимостей.
test-integration-compile:
	GOWORK=$(TEST_WORKSPACE) go test -tags=integration -run '^$$' $(INTEGRATION_PACKAGES)

# Реальный прогон не должен молча пропускаться из-за отсутствующей конфигурации.
test-integration:
	@for name in POSTERS_TEST_DATABASE_URL AVATAR_TEST_DATABASE_URL PERSONS_TEST_DATABASE_URL MIGRATIONS_TEST_DATABASE_URL INTEGRATION_S3_ENDPOINT INTEGRATION_S3_BUCKET RUSTFS_ACCESS_KEY RUSTFS_SECRET_KEY; do \
		test -n "$$(printenv $$name)" || { echo "Set $$name for integration tests"; exit 1; }; \
	done
	GOWORK=$(TEST_WORKSPACE) go test -race -count=1 -tags=integration $(INTEGRATION_PACKAGES)

cover:
	bash scripts/coverage.sh

mocks:
	GOWORK=$(TEST_WORKSPACE) go generate .

api-gen:
	GOWORK=$(TEST_WORKSPACE) go tool oapi-codegen -config oapi-codegen.yaml documentation.yaml

# Сборка приложения
build:
	mkdir -p bin
	go build -o bin/auth ./services/auth/cmd
	go build -o bin/user ./services/user/cmd
	go build -o bin/persons ./services/persons/cmd
	go build -o bin/catalog ./services/catalog/cmd
	go build -o bin/api-gateway ./api-gateway/cmd

# Запуск сервера
run:
	docker compose up --build

# Полная проверка (одной командой: make check)
check: lint test build

# Проверка API через curl (сервер должен быть запущен: make run)
test-api:
	@echo "== Регистрация =="
	curl -i -c cookies.txt -X POST http://localhost:8080/api/register \
		-H "Content-Type: application/json" \
		-d '{"email":"anna@example.com","username":"anna","password":"Password1"}'
	@echo "\n== Выход без сессии (ожидаем 401) =="
	curl -i -X POST http://localhost:8080/api/logout
	@echo "\n== Выход с сессией (ожидаем 204) =="
	curl -i -b cookies.txt -X POST http://localhost:8080/api/logout
	@echo "\n== Подборки (без входа, ожидаем 200) =="
	curl -i "http://localhost:8080/api/collections?limit=2"
	@rm -f cookies.txt

.PHONY: proto-gen
proto-gen:
	PATH="$(shell go env GOPATH)/bin:$$PATH" protoc -I . --go_out=paths=source_relative:. --go-grpc_out=paths=source_relative:. shared/proto/catalog/releases.proto shared/proto/auth/auth.proto
