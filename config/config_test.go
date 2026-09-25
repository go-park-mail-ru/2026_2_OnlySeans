package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func env(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

func TestParse(t *testing.T) {
	defaults := Default().Server

	tests := []struct {
		name string
		raw  string
		env  map[string]string
		want Server
	}{
		{
			name: "все поля из файла",
			raw: `
server:
  host: localhost
  port: 9000
  read_timeout: 1s
  write_timeout: 2s
  idle_timeout: 3s
  shutdown_timeout: 4s
`,
			want: Server{
				Host:            "localhost",
				Port:            9000,
				ReadTimeout:     time.Second,
				WriteTimeout:    2 * time.Second,
				IdleTimeout:     3 * time.Second,
				ShutdownTimeout: 4 * time.Second,
			},
		},
		{
			name: "пустой файл даёт значения по умолчанию",
			raw:  "",
			want: defaults,
		},
		{
			name: "незаданные поля берутся по умолчанию",
			raw:  "server:\n  port: 9000\n",
			want: Server{
				Host:            defaults.Host,
				Port:            9000,
				ReadTimeout:     defaults.ReadTimeout,
				WriteTimeout:    defaults.WriteTimeout,
				IdleTimeout:     defaults.IdleTimeout,
				ShutdownTimeout: defaults.ShutdownTimeout,
			},
		},
		{
			name: "переменные окружения важнее файла",
			raw:  "server:\n  host: localhost\n  port: 9000\n",
			env:  map[string]string{EnvServerHost: "127.0.0.1", EnvServerPort: "8081"},
			want: Server{
				Host:            "127.0.0.1",
				Port:            8081,
				ReadTimeout:     defaults.ReadTimeout,
				WriteTimeout:    defaults.WriteTimeout,
				IdleTimeout:     defaults.IdleTimeout,
				ShutdownTimeout: defaults.ShutdownTimeout,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Parse([]byte(tt.raw), env(tt.env))
			if err != nil {
				t.Fatalf("Parse вернул ошибку: %v", err)
			}
			if cfg.Server != tt.want {
				t.Errorf("Server = %+v, ожидали %+v", cfg.Server, tt.want)
			}
		})
	}
}

func TestParse_Errors(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		env  map[string]string
	}{
		{"битый YAML", "server: [", nil},
		{"неизвестное поле", "server:\n  prot: 8080\n", nil},
		{"порт вне диапазона", "server:\n  port: 70000\n", nil},
		{"пустой хост", "server:\n  host: \"\"\n", nil},
		{"нулевой таймаут", "server:\n  read_timeout: 0s\n", nil},
		{"отрицательный таймаут остановки", "server:\n  shutdown_timeout: -1s\n", nil},
		{"порт из окружения не число", "", map[string]string{EnvServerPort: "abc"}},
		{"порт из окружения вне диапазона", "", map[string]string{EnvServerPort: "0"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Parse([]byte(tt.raw), env(tt.env)); err == nil {
				t.Error("ожидали ошибку, получили nil")
			}
		})
	}
}

func TestServer_Addr(t *testing.T) {
	tests := []struct {
		host string
		port int
		want string
	}{
		{"localhost", 8080, "localhost:8080"},
		{"0.0.0.0", 80, "0.0.0.0:80"},
		{"::1", 8080, "[::1]:8080"},
	}

	for _, tt := range tests {
		if got := (Server{Host: tt.host, Port: tt.port}).Addr(); got != tt.want {
			t.Errorf("Addr() = %q, ожидали %q", got, tt.want)
		}
	}
}

func TestLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("server:\n  port: 9000\n"), 0o600); err != nil {
		t.Fatalf("не удалось записать файл: %v", err)
	}
	t.Setenv(EnvServerPort, "9001")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load вернул ошибку: %v", err)
	}
	if cfg.Server.Port != 9001 {
		t.Errorf("Port = %d, ожидали 9001", cfg.Server.Port)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Error("ожидали ошибку для несуществующего файла")
	}
}

func TestRepositoryConfigIsValid(t *testing.T) {
	if _, err := Load(filepath.Join("..", "configs", "config.yaml")); err != nil {
		t.Errorf("configs/config.yaml невалиден: %v", err)
	}
}
