package config_test

import (
	"strings"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/config"
)

func TestParse_Database(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		env  map[string]string
		want config.Database
	}{
		{
			name: "defaults without database section",
			raw:  "",
			want: config.Default().Database,
		},
		{
			name: "pool settings from file",
			raw: `
database:
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
				db := config.Default().Database
				db.URL = "postgres://user:secret@db:5432/kinopoisk"
				return db
			}(),
		},
		{
			name: "environment overrides file",
			raw: `
database:
  url: postgres://from-file/db
`,
			env: map[string]string{config.EnvDatabaseURL: "postgres://from-env/db"},
			want: func() config.Database {
				db := config.Default().Database
				db.URL = "postgres://from-env/db"
				return db
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := config.Parse([]byte(tt.raw), env(tt.env))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if cfg.Database != tt.want {
				t.Errorf("Database = %+v, want %+v", cfg.Database, tt.want)
			}
		})
	}
}

func TestParse_DatabaseErrors(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr string
	}{
		{"no open connections", "database:\n  max_open_conns: 0\n", "database.max_open_conns"},
		{"negative idle connections", "database:\n  max_idle_conns: -1\n", "database.max_idle_conns"},
		{"more idle than open", "database:\n  max_open_conns: 2\n  max_idle_conns: 3\n", "database.max_idle_conns"},
		{"zero lifetime", "database:\n  conn_max_lifetime: 0s\n", "database.conn_max_lifetime"},
		{"negative connect timeout", "database:\n  connect_timeout: -1s\n", "database.connect_timeout"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := config.Parse([]byte(tt.raw), env(nil))
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not mention %q", err, tt.wantErr)
			}
		})
	}
}
