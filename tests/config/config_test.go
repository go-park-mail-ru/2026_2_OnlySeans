package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/config"
)

func env(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

func TestParse(t *testing.T) {
	defaults := config.Default().Server

	tests := []struct {
		name string
		raw  string
		env  map[string]string
		want config.Server
	}{
		{
			name: "all fields from file",
			raw: `
server:
  host: localhost
  port: 9000
  read_timeout: 1s
  write_timeout: 2s
  idle_timeout: 3s
  shutdown_timeout: 4s
`,
			want: config.Server{
				Host:            "localhost",
				Port:            9000,
				AllowedOrigin:   defaults.AllowedOrigin,
				ReadTimeout:     time.Second,
				WriteTimeout:    2 * time.Second,
				IdleTimeout:     3 * time.Second,
				ShutdownTimeout: 4 * time.Second,
			},
		},
		{
			name: "empty file gives defaults",
			raw:  "",
			want: defaults,
		},
		{
			name: "missing fields fall back to defaults",
			raw:  "server:\n  port: 9000\n",
			want: config.Server{
				Host:            defaults.Host,
				Port:            9000,
				AllowedOrigin:   defaults.AllowedOrigin,
				ReadTimeout:     defaults.ReadTimeout,
				WriteTimeout:    defaults.WriteTimeout,
				IdleTimeout:     defaults.IdleTimeout,
				ShutdownTimeout: defaults.ShutdownTimeout,
			},
		},
		{
			name: "environment overrides file",
			raw:  "server:\n  host: localhost\n  port: 9000\n",
			env: map[string]string{
				config.EnvServerHost: "127.0.0.1",
				config.EnvServerPort: "8081",
			},
			want: config.Server{
				Host:            "127.0.0.1",
				Port:            8081,
				AllowedOrigin:   defaults.AllowedOrigin,
				ReadTimeout:     defaults.ReadTimeout,
				WriteTimeout:    defaults.WriteTimeout,
				IdleTimeout:     defaults.IdleTimeout,
				ShutdownTimeout: defaults.ShutdownTimeout,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := config.Parse([]byte(tt.raw), env(tt.env))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if cfg.Server != tt.want {
				t.Errorf("Server = %+v, want %+v", cfg.Server, tt.want)
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
		{
			name: "broken yaml",
			raw:  "server: [",
		},
		{
			name: "unknown field",
			raw:  "server:\n  prot: 8080\n",
		},
		{
			name: "port above range",
			raw:  "server:\n  port: 70000\n",
		},
		{
			name: "port below range",
			raw:  "server:\n  port: 0\n",
		},
		{
			name: "empty host",
			raw:  "server:\n  host: \"\"\n",
		},
		{
			name: "zero timeout",
			raw:  "server:\n  read_timeout: 0s\n",
		},
		{
			name: "negative shutdown timeout",
			raw:  "server:\n  shutdown_timeout: -1s\n",
		},
		{
			name: "env port is not a number",
			env:  map[string]string{config.EnvServerPort: "abc"},
		},
		{
			name: "env port out of range",
			env:  map[string]string{config.EnvServerPort: "0"},
		},
		{
			name: "empty allowed origin",
			raw:  "server:\n  allowed_origin: \"\"\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := config.Parse([]byte(tt.raw), env(tt.env)); err == nil {
				t.Error("expected error, got nil")
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
		{
			host: "localhost",
			port: 8080,
			want: "localhost:8080",
		},
		{
			host: "0.0.0.0",
			port: 80,
			want: "0.0.0.0:80",
		},
		{
			host: "::1",
			port: 8080,
			want: "[::1]:8080",
		},
	}

	for _, tt := range tests {
		server := config.Server{
			Host: tt.host,
			Port: tt.port,
		}
		if got := server.Addr(); got != tt.want {
			t.Errorf("Addr() = %q, want %q", got, tt.want)
		}
	}
}

func TestLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("server:\n  port: 9000\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv(config.EnvServerPort, "9001")

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.Port != 9001 {
		t.Errorf("Port = %d, want 9001", cfg.Server.Port)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	if _, err := config.Load(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Error("expected error for missing file")
	}
}

func TestRepositoryConfigIsValid(t *testing.T) {
	if _, err := config.Load(filepath.Join("..", "..", "configs", "config.yaml")); err != nil {
		t.Errorf("configs/config.yaml is invalid: %v", err)
	}
}
