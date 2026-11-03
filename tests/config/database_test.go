package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/config"
)

func TestParseDatabase(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		env  map[string]string
		want config.Database
	}{
		{
			name: "defaults for empty file",
			raw:  "",
			want: *config.DefaultDatabase(),
		},
		{
			name: "pool settings from file",
			raw: `
max_open_conns: 20
max_idle_conns: 0
conn_max_lifetime: 1h
connect_timeout: 2s
`,
			want: config.Database{
				MaxOpenConns:    20,
				MaxIdleConns:    0,
				ConnMaxLifetime: time.Hour,
				ConnectTimeout:  2 * time.Second,
			},
		},
		{
			name: "url from environment",
			env:  map[string]string{config.EnvDatabaseURL: "postgres://user:secret@db:5432/kinopoisk"},
			want: func() config.Database {
				db := *config.DefaultDatabase()
				db.URL = "postgres://user:secret@db:5432/kinopoisk"
				return db
			}(),
		},
		{
			name: "environment overrides file",
			raw:  "url: postgres://from-file/db\n",
			env:  map[string]string{config.EnvDatabaseURL: "postgres://from-env/db"},
			want: func() config.Database {
				db := *config.DefaultDatabase()
				db.URL = "postgres://from-env/db"
				return db
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := config.ParseDatabase([]byte(tt.raw), env(tt.env))
			if err != nil {
				t.Fatalf("ParseDatabase: %v", err)
			}
			if *cfg != tt.want {
				t.Errorf("Database = %+v, want %+v", *cfg, tt.want)
			}
		})
	}
}

func TestParseDatabase_Errors(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr string
	}{
		{"no open connections", "max_open_conns: 0\n", "database.max_open_conns"},
		{"negative idle connections", "max_idle_conns: -1\n", "database.max_idle_conns"},
		{"more idle than open", "max_open_conns: 2\nmax_idle_conns: 3\n", "database.max_idle_conns"},
		{"zero lifetime", "conn_max_lifetime: 0s\n", "database.conn_max_lifetime"},
		{"negative connect timeout", "connect_timeout: -1s\n", "database.connect_timeout"},
		{"old section header", "database:\n  max_open_conns: 5\n", "parse database config"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := config.ParseDatabase([]byte(tt.raw), env(nil))
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not mention %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoadDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "database.yaml")
	if err := os.WriteFile(path, []byte("max_open_conns: 3\nmax_idle_conns: 1\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv(config.EnvDatabaseURL, "postgres://from-env/db")

	cfg, err := config.LoadDatabase(path)
	if err != nil {
		t.Fatalf("LoadDatabase: %v", err)
	}
	if cfg.MaxOpenConns != 3 || cfg.MaxIdleConns != 1 {
		t.Errorf("pool = %d/%d, want 3/1", cfg.MaxOpenConns, cfg.MaxIdleConns)
	}
	if cfg.URL != "postgres://from-env/db" {
		t.Errorf("URL = %q, want it from %s", cfg.URL, config.EnvDatabaseURL)
	}
}

func TestLoadDatabase_MissingFile(t *testing.T) {
	if _, err := config.LoadDatabase(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Error("expected error for missing file")
	}
}
