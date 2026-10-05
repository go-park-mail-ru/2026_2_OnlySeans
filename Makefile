.PHONY: lint fmt test cover build run check test-api

# Команды для проверки качества кода
lint:
	@test -z "$$(gofmt -l .)" || (echo "Неотформатированные файлы:"; gofmt -l .; exit 1)
	go vet ./...

# Автоформатирование
fmt:
	gofmt -w .

# Запуск тестов (с проверкой гонок данных)
test:
	go test -race ./...

cover:
	bash scripts/coverage.sh

# Сборка приложения
build:
	go build -o server ./cmd/main.go

# Запуск сервера
run:
	go run ./cmd/main.go

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
