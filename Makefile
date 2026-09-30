# Команды для проверки качества кода
lint:
	gofmt -l .
	go vet ./...

# Запуск тестов
test:
	go test ./...

# Сборка приложения
build:
	go build -o server ./cmd/main.go

# Запуск сервера
run:
	go run ./cmd/main.go

# Полная проверка (одной командой: make check)
check: lint test build

# Тесты через curl (регистрация и проверка сессии)
test-api:
	# Регистрация
	curl -i -c cookies.txt -X POST http://127.0.0.1:8080/api/register \
	-H "Content-Type: application/json" \
	-d '{"email":"anna@example.com","username":"anna","password":"Password1"}'
	
	# Проверка /me
	curl -i -b cookies.txt http://127.0.0.1:8080/api/me