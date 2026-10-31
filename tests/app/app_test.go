package app_test

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/app"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/config"
)

// кладёт файл конфига во временную папку теста и возвращает путь к нему
func writeConfig(t *testing.T, name, raw string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

// конфиг базы со значениями по умолчанию, для тестов, где база не важна
func emptyDatabaseConfig(t *testing.T) string {
	t.Helper()

	return writeConfig(t, "database.yaml", "")
}

func emptyRedisConfig(t *testing.T) string {
	t.Helper()

	return writeConfig(t, "redis.yaml", "")
}

func TestRun_InvalidConfig(t *testing.T) {
	path := writeConfig(t, "config.yaml", "server:\n  port: 0\n")

	if err := app.Run(path, emptyDatabaseConfig(t), emptyRedisConfig(t)); err == nil {
		t.Error("expected error for invalid config")
	}
}

// без адреса базы сервер не стартует, и ошибка называет нужную переменную
func TestRun_MissingDatabaseURL(t *testing.T) {
	t.Setenv(config.EnvDatabaseURL, "")
	path := writeConfig(t, "config.yaml", "server:\n  host: 127.0.0.1\n  port: 1\n")

	err := app.Run(path, emptyDatabaseConfig(t), emptyRedisConfig(t))
	if err == nil {
		t.Fatal("expected error without database url")
	}
	if !strings.Contains(err.Error(), config.EnvDatabaseURL) {
		t.Errorf("error %q does not mention %s", err, config.EnvDatabaseURL)
	}
}

func TestRun_MissingRedisURL(t *testing.T) {
	t.Setenv(config.EnvDatabaseURL, "postgres://user@127.0.0.1:18081/kinopoisk?sslmode=disable")
	t.Setenv(config.EnvRedisURL, "")
	path := writeConfig(t, "config.yaml", "server:\n  host: 127.0.0.1\n  port: 18082\n")

	err := app.Run(path, emptyDatabaseConfig(t), emptyRedisConfig(t))
	if err == nil {
		t.Fatal("expected error without redis url")
	}
	if !strings.Contains(err.Error(), config.EnvRedisURL) {
		t.Errorf("error %q does not mention %s", err, config.EnvRedisURL)
	}
}

func TestRun_MissingConfig(t *testing.T) {
	if err := app.Run(filepath.Join(t.TempDir(), "missing.yaml"), emptyDatabaseConfig(t), emptyRedisConfig(t)); err == nil {
		t.Error("expected error for missing config")
	}
}

func TestRun_MissingDatabaseConfig(t *testing.T) {
	path := writeConfig(t, "config.yaml", "server:\n  host: 127.0.0.1\n  port: 1\n")

	if err := app.Run(path, filepath.Join(t.TempDir(), "missing.yaml"), emptyRedisConfig(t)); err == nil {
		t.Error("expected error for missing database config")
	}
}

func TestRun_MissingRedisConfig(t *testing.T) {
	path := writeConfig(t, "config.yaml", "server:\n  host: 127.0.0.1\n  port: 18081\n")

	if err := app.Run(path, emptyDatabaseConfig(t), filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Error("expected error for missing redis config")
	}
}

func closedAddr(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}
	return addr
}

func TestRun_DatabaseUnavailable(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	closedAddr := listener.Addr().String()
	listener.Close()

	t.Setenv(config.EnvDatabaseURL, "postgres://user@"+closedAddr+"/kinopoisk?sslmode=disable")
	path := writeConfig(t, "config.yaml", "server:\n  host: 127.0.0.1\n  port: 1\n")
	databasePath := writeConfig(t, "database.yaml", "connect_timeout: 2s\n")
	redisPath := writeConfig(t, "redis.yaml", "connect_timeout: 5s\n")

	done := make(chan error, 1)
	go func() { done <- app.Run(path, databasePath, redisPath) }()

	select {
	case err := <-done:
		if err == nil {
			t.Error("expected error when the database is unavailable")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not give up on an unavailable database")
	}
}

func TestRun_RedisUnavailable(t *testing.T) {
	t.Setenv(config.EnvDatabaseURL, "postgres://user@"+closedAddr(t)+"/kinopoisk?sslmode=disable")
	t.Setenv(config.EnvRedisURL, "redis://"+closedAddr(t))
	path := writeConfig(t, "config.yaml", "server:\n  host: 127.0.0.1\n  port: 18081\n")
	databasePath := writeConfig(t, "database.yaml", "connect_timeout: 2s\n")
	redisPath := writeConfig(t, "redis.yaml", "connect_timeout: 2s\n")

	done := make(chan error, 1)
	go func() { done <- app.Run(path, databasePath, redisPath) }()

	select {
	case err := <-done:
		if err == nil {
			t.Error("expected error when redis is unavailable")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not give up on an unavailable redis")
	}
}
