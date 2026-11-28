package config_test

import (
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/config"
)

func TestLoadOnlineConfDefaultsAndOverrides(t *testing.T) {
	t.Parallel()

	defaults, err := config.LoadOnlineConfRuntime(func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatalf("load defaults: %v", err)
	}
	if defaults.RefreshInterval != 10*time.Second || defaults.HTTPTimeout != 2*time.Second {
		t.Fatalf("unexpected defaults: %+v", defaults)
	}
	if defaults.SnapshotPath != "/internal/onlineconf/snapshot" {
		t.Fatalf("unexpected snapshot path: %q", defaults.SnapshotPath)
	}

	env := map[string]string{
		config.EnvOnlineConfURL:                 "http://onlineconf:8081",
		config.EnvOnlineConfRuntimeServiceToken: "internal-secret",
		config.EnvOnlineConfRefreshInterval:     "15s",
		config.EnvOnlineConfHTTPTimeout:         "750ms",
		config.EnvOnlineConfSnapshotPath:        "/internal/snapshot",
	}
	overridden, err := config.LoadOnlineConfRuntime(func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	})
	if err != nil {
		t.Fatalf("load overrides: %v", err)
	}
	if overridden.URL != env[config.EnvOnlineConfURL] ||
		overridden.ServiceToken != env[config.EnvOnlineConfRuntimeServiceToken] ||
		overridden.RefreshInterval != 15*time.Second ||
		overridden.HTTPTimeout != 750*time.Millisecond ||
		overridden.SnapshotPath != "/internal/snapshot" {
		t.Fatalf("unexpected overrides: %+v", overridden)
	}
}

func TestLoadOnlineConfRejectsUnsafeConfiguration(t *testing.T) {
	t.Parallel()

	tests := []map[string]string{
		{config.EnvOnlineConfURL: "postgres://onlineconf", config.EnvOnlineConfRuntimeServiceToken: "secret"},
		{config.EnvOnlineConfURL: "http://onlineconf:8081"},
		{config.EnvOnlineConfRefreshInterval: "0s"},
		{config.EnvOnlineConfHTTPTimeout: "not-a-duration"},
		{config.EnvOnlineConfSnapshotPath: "internal/snapshot"},
	}
	for _, env := range tests {
		env := env
		t.Run("invalid", func(t *testing.T) {
			t.Parallel()
			if _, err := config.LoadOnlineConfRuntime(func(key string) (string, bool) {
				value, ok := env[key]
				return value, ok
			}); err == nil {
				t.Fatal("expected configuration error")
			}
		})
	}
}
