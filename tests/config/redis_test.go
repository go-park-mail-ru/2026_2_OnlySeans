package config_test

import (
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/config"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRedisConnectionConfig(t *testing.T) {
	for _, tc := range []struct {
		name, raw, override, want string
		invalid                   bool
	}{
		{name: "defaults", want: "redis://localhost:8020"},
		{name: "host and port", raw: "host: cache\nport: 6379", want: "redis://cache:6379"},
		{name: "IPv6", raw: "host: '::1'\nport: 6379", want: "redis://[::1]:6379"},
		{name: "legacy URL", raw: "url: redis://cache:6380", want: "redis://cache:6380"},
		{name: "environment override", raw: "host: cache\nport: 6379", override: "rediss://user:secret@prod:6380/2", want: "rediss://user:secret@prod:6380/2"},
		{name: "invalid port", raw: "port: 70000", invalid: true},
		{name: "missing host", raw: "host: ''", invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			values := map[string]string{}
			if tc.override != "" {
				values[config.EnvRedisURL] = tc.override
			}
			cfg, err := config.ParseRedis([]byte(tc.raw), env(values))
			if tc.invalid {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, cfg.URL)
		})
	}
}
