package app_test

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
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

	dummyDatabaseURL = "postgres://user@127.0.0.1:1/kinopoisk?sslmode=disable"

	testDatabaseURLEnv = "TEST_DATABASE_URL"
	testRedisURLEnv    = "TEST_REDIS_URL"

	runTimeout = 10 * time.Second
)

func writeConfig(t *testing.T, name, raw string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(raw), 0o600))
	return path
}

type configPaths struct {
	server   string
	database string
	redis    string
	s3       string
	logger   string
}

func newConfigPaths(t *testing.T) configPaths {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	t.Setenv(config.EnvS3Endpoint, server.URL)
	t.Setenv(config.EnvS3Region, "us-east-1")
	t.Setenv(config.EnvS3Bucket, "avatars")
	t.Setenv(config.EnvS3AccessKey, "test-access-key")
	t.Setenv(config.EnvS3SecretKey, "test-secret-key")

	return configPaths{
		server:   writeConfig(t, "config.yaml", validServerConfig),
		database: writeConfig(t, "database.yaml", ""),
		redis:    writeConfig(t, "redis.yaml", ""),
		s3:       writeConfig(t, "s3.yaml", "posters_bucket: posters\n"),
		logger:   writeConfig(t, "logger.yaml", fmt.Sprintf("file: %q\n", filepath.Join(t.TempDir(), "app.log"))),
	}
}

func (p configPaths) run() error {
	return app.Run(p.server, p.database, p.redis, p.s3, p.logger)
}

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

func TestRun_MissingConfigFiles(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(p *configPaths, missing string)
		wantErr string
	}{
		{"нет основного конфига", func(p *configPaths, m string) { p.server = m }, "read config"},
		{"нет конфига базы", func(p *configPaths, m string) { p.database = m }, "read database config"},
		{"нет конфига redis", func(p *configPaths, m string) { p.redis = m }, "read redis config"},
		{"нет конфига S3", func(p *configPaths, m string) { p.s3 = m }, "read s3 config"},
		{"нет конфига логгера", func(p *configPaths, m string) { p.logger = m }, "read logger config"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			t.Setenv(config.EnvDatabaseURL, dummyDatabaseURL)
			p := newConfigPaths(t)
			tt.mutate(&p, missingFile(t))

			require.ErrorContains(t, p.run(), tt.wantErr)
		})
	}
}

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

func TestRun_S3BucketCheck(t *testing.T) {
	tests := []struct {
		name         string
		avatarStatus int
		posterStatus int
		wantErr      string
		wantChecked  []string
	}{
		{
			name:         "both buckets available, startup proceeds to database",
			avatarStatus: http.StatusOK,
			posterStatus: http.StatusOK,
			wantErr:      "open database",
			wantChecked:  []string{"/avatars", "/posters"},
		},
		{
			name:         "avatar bucket missing",
			avatarStatus: http.StatusNotFound,
			posterStatus: http.StatusOK,
			wantErr:      `check s3 bucket "avatars"`,
			wantChecked:  []string{"/avatars"},
		},
		{
			name:         "avatar access denied",
			avatarStatus: http.StatusForbidden,
			posterStatus: http.StatusOK,
			wantErr:      `check s3 bucket "avatars"`,
			wantChecked:  []string{"/avatars"},
		},
		{
			name:         "poster bucket missing",
			avatarStatus: http.StatusOK,
			posterStatus: http.StatusNotFound,
			wantErr:      `check s3 bucket "posters"`,
			wantChecked:  []string{"/avatars", "/posters"},
		},
		{
			name:         "poster access denied",
			avatarStatus: http.StatusOK,
			posterStatus: http.StatusForbidden,
			wantErr:      `check s3 bucket "posters"`,
			wantChecked:  []string{"/avatars", "/posters"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(config.EnvDatabaseURL, dummyDatabaseURL)
			p := newConfigPaths(t)
			checked := make(chan string, 10)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodHead {
					t.Errorf("unexpected S3 method: %s", r.Method)
				}
				checked <- r.URL.Path
				switch r.URL.Path {
				case "/avatars":
					w.WriteHeader(tt.avatarStatus)
				case "/posters":
					w.WriteHeader(tt.posterStatus)
				default:
					t.Errorf("unexpected S3 bucket: %s", r.URL.Path)
					w.WriteHeader(http.StatusBadRequest)
				}
			}))
			defer server.Close()
			t.Setenv(config.EnvS3Endpoint, server.URL)
			require.ErrorContains(t, runWithTimeout(t, p), tt.wantErr)
			var gotChecked []string
			for len(checked) > 0 {
				gotChecked = append(gotChecked, <-checked)
			}
			require.Equal(t, tt.wantChecked, gotChecked)
		})
	}
}
