package onlineconf_test

import (
	"strings"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/config"
)

func TestOnlineConfConfiguration(t *testing.T) {
	lookup := func(key string) (string, bool) {
		values := map[string]string{config.EnvOnlineConfServiceToken: accessTestToken, config.EnvOnlineConfPort: "9081", config.EnvOnlineConfAllowedOrigins: accessTestOrigin}
		value, ok := values[key]

		return value, ok
	}

	cfg, err := config.ParseOnlineConf(nil, lookup)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Server.Port != 9081 || cfg.ServiceToken != accessTestToken || cfg.Snapshot.RefreshInterval != config.DefaultOnlineConfRefreshInterval {
		t.Fatal("environment or defaults not applied")
	}

	for _, raw := range []string{"service_token: secret", "snapshot:\n  refresh_interval: 11s", "snapshot:\n  load_timeout: 3s", "server:\n  read_timeout: 0s"} {
		if _, err := config.ParseOnlineConf([]byte(raw), lookup); err == nil {
			t.Fatal("invalid configuration accepted", raw)
		}
	}

	if _, err := config.ParseOnlineConf(nil, func(string) (string, bool) { return "", false }); err == nil {
		t.Fatal("missing token accepted")
	}

	if _, err := config.ParseOnlineConf(nil, func(key string) (string, bool) {
		if key == config.EnvOnlineConfServiceToken {
			return "private bad token", true
		}
		return "", false
	}); err == nil || strings.Contains(err.Error(), "private bad token") {
		t.Fatal("invalid token or leaked secret", err)
	}
}
