package storage_test

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/config"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/storage"
)

// testDatabaseURLEnv задаёт адрес настоящей базы для проверки успешного подключения.
// Без неё тест подключения пропускается, остальные проверки работают без базы.
const testDatabaseURLEnv = "TEST_DATABASE_URL"

func databaseConfig(url string) config.Database {
	return config.Database{
		URL:             url,
		MaxOpenConns:    2,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Minute,
		ConnectTimeout:  2 * time.Second,
	}
}

// closedPort возвращает адрес, на котором точно никто не слушает.
func closedPort(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()
	require.NoError(t, listener.Close())
	return addr
}

func TestOpen_InvalidURL(t *testing.T) {
	db, err := storage.Open(context.Background(), databaseConfig("postgres://user@host:not-a-port/db"))
	require.Error(t, err)
	assert.Nil(t, db)
}

func TestOpen_DatabaseUnavailable(t *testing.T) {
	start := time.Now()
	db, err := storage.Open(context.Background(),
		databaseConfig("postgres://user:secret@"+closedPort(t)+"/kinopoisk?sslmode=disable"))

	require.Error(t, err)
	assert.Nil(t, db)
	assert.Contains(t, err.Error(), "ping database")
	assert.NotContains(t, err.Error(), "secret", "пароль из адреса не должен попадать в текст ошибки")
	assert.Less(t, time.Since(start), 10*time.Second, "подключение не должно висеть дольше таймаута")
}

func TestOpen_RespectsConnectTimeout(t *testing.T) {
	// Сервер принимает соединение, но молчит: без таймаута Open ждал бы вечно.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { listener.Close() })
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			t.Cleanup(func() { conn.Close() })
		}
	}()

	cfg := databaseConfig("postgres://user@" + listener.Addr().String() + "/kinopoisk?sslmode=disable")
	cfg.ConnectTimeout = 300 * time.Millisecond

	start := time.Now()
	_, err = storage.Open(context.Background(), cfg)
	require.Error(t, err)
	assert.Less(t, time.Since(start), 5*time.Second)
}

func TestOpen_RealDatabase(t *testing.T) {
	url := os.Getenv(testDatabaseURLEnv)
	if url == "" {
		t.Skipf("set %s to run against a real PostgreSQL", testDatabaseURLEnv)
	}

	db, err := storage.Open(context.Background(), databaseConfig(url))
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	assert.Equal(t, 2, db.Stats().MaxOpenConnections)
	assert.NoError(t, db.PingContext(context.Background()))
}
