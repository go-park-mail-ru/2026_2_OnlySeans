package app_test

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/app"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/config"
)

const (
	validServerConfig = "server:\n  host: 127.0.0.1\n  port: 18080\n"

	// Адрес, к которому никто не подключится. Нужен тестам, которые
	// заканчиваются раньше, чем Run откроет базу.
	dummyDatabaseURL = "postgres://user@127.0.0.1:1/kinopoisk?sslmode=disable"

	// Тесты с настоящими Postgres и Redis пропускаются, если переменные не заданы.
	testDatabaseURLEnv = "TEST_DATABASE_URL"
	testRedisURLEnv    = "TEST_REDIS_URL"

	runTimeout = 10 * time.Second
)

// кладёт файл конфига во временную папку теста и возвращает путь к нему
func writeConfig(t *testing.T, name, raw string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(raw), 0o600))
	return path
}

// пути к четырём конфигам, которые принимает Run
type configPaths struct {
	server   string
	database string
	redis    string
	logger   string
}

// newConfigPaths создаёт рабочие конфиги. Пустой файл даёт значения по умолчанию.
func newConfigPaths(t *testing.T) configPaths {
	t.Helper()

	return configPaths{
		server:   writeConfig(t, "config.yaml", validServerConfig),
		database: writeConfig(t, "database.yaml", ""),
		redis:    writeConfig(t, "redis.yaml", ""),
		logger:   writeConfig(t, "logger.yaml", ""),
	}
}

func (p configPaths) run() error {
	return app.Run(p.server, p.database, p.redis, p.logger)
}

// runWithTimeout не даёт тесту зависнуть, если Run не возвращается
func runWithTimeout(t *testing.T, p configPaths) error {
	t.Helper()

	done := make(chan error, 1)
	go func() { done <- p.run() }()

	select {
	case err := <-done:
		return err
	case <-time.After(runTimeout):
		t.Fatal("Run did not return in time")
		return nil
	}
}

func missingFile(t *testing.T) string {
	t.Helper()

	return filepath.Join(t.TempDir(), "missing.yaml")
}

// адрес, на котором только что закрыли порт: подключиться к нему нельзя
func closedAddr(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	addr := listener.Addr().String()
	require.NoError(t, listener.Close())
	return addr
}

func requireEnv(t *testing.T, key string) string {
	t.Helper()

	value := os.Getenv(key)
	if value == "" {
		t.Skipf("%s is not set", key)
	}
	return value
}

func TestRun_InvalidConfig(t *testing.T) {
	p := newConfigPaths(t)
	p.server = writeConfig(t, "config.yaml", "server:\n  port: 0\n")

	err := p.run()

	require.ErrorContains(t, err, "load config")
	require.ErrorContains(t, err, "server.port")
}

// каждый отсутствующий файл конфига должен давать ошибку именно про него
func TestRun_MissingConfigFiles(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(p *configPaths, missing string)
		wantErr string
	}{
		{"нет основного конфига", func(p *configPaths, m string) { p.server = m }, "read config"},
		{"нет конфига базы", func(p *configPaths, m string) { p.database = m }, "read database config"},
		{"нет конфига redis", func(p *configPaths, m string) { p.redis = m }, "read redis config"},
		{"нет конфига логгера", func(p *configPaths, m string) { p.logger = m }, "read logger config"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// чтобы Run дошёл до чтения нужного файла, а не остановился на адресе базы
			t.Setenv(config.EnvDatabaseURL, dummyDatabaseURL)
			p := newConfigPaths(t)
			tt.mutate(&p, missingFile(t))

			require.ErrorContains(t, p.run(), tt.wantErr)
		})
	}
}

// без адреса базы сервер не стартует, и ошибка называет нужную переменную
func TestRun_MissingDatabaseURL(t *testing.T) {
	t.Setenv(config.EnvDatabaseURL, "")

	err := newConfigPaths(t).run()

	require.ErrorContains(t, err, config.EnvDatabaseURL)
}

func TestRun_MissingRedisURL(t *testing.T) {
	t.Setenv(config.EnvDatabaseURL, dummyDatabaseURL)
	t.Setenv(config.EnvRedisURL, "")

	err := newConfigPaths(t).run()

	require.ErrorContains(t, err, "load redis config")
	require.ErrorContains(t, err, "redis.url")
}

// неверные настройки логгера обнаруживаются при старте, до открытия базы
func TestRun_InvalidLoggerSettings(t *testing.T) {
	tests := []struct {
		name    string
		env     string
		value   string
		wantErr string
	}{
		{"неизвестный уровень", config.EnvLogLevel, "loud", "invalid log level"},
		{"неизвестный формат", config.EnvLogFormat, "xml", "invalid log format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(config.EnvDatabaseURL, dummyDatabaseURL)
			t.Setenv(tt.env, tt.value)

			err := newConfigPaths(t).run()

			require.ErrorContains(t, err, "create logger")
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestRun_DatabaseUnavailable(t *testing.T) {
	t.Setenv(config.EnvDatabaseURL, "postgres://user@"+closedAddr(t)+"/kinopoisk?sslmode=disable")
	p := newConfigPaths(t)
	p.database = writeConfig(t, "database.yaml", "connect_timeout: 2s\n")

	err := runWithTimeout(t, p)

	require.ErrorContains(t, err, "open database")
}

// Дальше тесты, которым нужна настоящая база: Run открывает Postgres раньше Redis.
// Запуск: TEST_DATABASE_URL=... TEST_REDIS_URL=... go test ./tests/app/ -v

func TestRun_RedisUnavailable(t *testing.T) {
	t.Setenv(config.EnvDatabaseURL, requireEnv(t, testDatabaseURLEnv))
	t.Setenv(config.EnvRedisURL, "redis://"+closedAddr(t))

	err := runWithTimeout(t, newConfigPaths(t))

	require.ErrorContains(t, err, "connect redis")
}

func TestRun_PortInUse(t *testing.T) {
	databaseURL := requireEnv(t, testDatabaseURLEnv)
	redisURL := requireEnv(t, testRedisURLEnv)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	t.Setenv(config.EnvDatabaseURL, databaseURL)
	t.Setenv(config.EnvRedisURL, redisURL)
	p := newConfigPaths(t)
	p.server = writeConfig(t, "config.yaml", fmt.Sprintf("server:\n  host: 127.0.0.1\n  port: %d\n", port))

	err = runWithTimeout(t, p)

	require.ErrorContains(t, err, "address already in use")
}
