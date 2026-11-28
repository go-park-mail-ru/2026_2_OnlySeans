package config

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	EnvOnlineConfHost                = "ONLINECONF_HOST"
	EnvOnlineConfPort                = "ONLINECONF_PORT"
	EnvOnlineConfServiceToken        = "ONLINECONF_SERVICE_TOKEN"
	EnvOnlineConfAllowedOrigins      = "ONLINECONF_ALLOWED_ORIGINS"
	DefaultOnlineConfPort            = 8081
	DefaultOnlineConfRefreshInterval = 10 * time.Second
	DefaultOnlineConfLoadTimeout     = 2 * time.Second
)

type OnlineConf struct {
	Server       Server             `yaml:"server"`
	Snapshot     OnlineConfSnapshot `yaml:"snapshot"`
	ServiceToken string             `yaml:"-"`
}

type OnlineConfSnapshot struct {
	RefreshInterval time.Duration `yaml:"refresh_interval"`
	LoadTimeout     time.Duration `yaml:"load_timeout"`
}

func DefaultOnlineConf() *OnlineConf {
	server := Default().Server
	server.Port = DefaultOnlineConfPort

	return &OnlineConf{Server: server, Snapshot: OnlineConfSnapshot{RefreshInterval: DefaultOnlineConfRefreshInterval, LoadTimeout: DefaultOnlineConfLoadTimeout}}
}

func LoadOnlineConf(path string) (*OnlineConf, error) {
	return loadFile(path, "onlineconf config", ParseOnlineConf)
}

func ParseOnlineConf(raw []byte, lookupEnv func(string) (string, bool)) (*OnlineConf, error) {
	cfg := DefaultOnlineConf()
	if err := decodeStrict(raw, cfg, "onlineconf config"); err != nil {
		return nil, err
	}

	if host, ok := lookupEnv(EnvOnlineConfHost); ok {
		cfg.Server.Host = host
	}

	if rawPort, ok := lookupEnv(EnvOnlineConfPort); ok {
		port, err := strconv.Atoi(rawPort)
		if err != nil {
			return nil, fmt.Errorf("%s must be an integer", EnvOnlineConfPort)
		}

		cfg.Server.Port = port
	}

	if origins, ok := lookupEnv(EnvOnlineConfAllowedOrigins); ok {
		cfg.Server.AllowedOrigins = strings.Split(origins, ",")

		for i := range cfg.Server.AllowedOrigins {
			cfg.Server.AllowedOrigins[i] = strings.TrimSpace(cfg.Server.AllowedOrigins[i])
		}
	}

	cfg.ServiceToken, _ = lookupEnv(EnvOnlineConfServiceToken)

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *OnlineConf) Validate() error {
	serverConfig := Config{Server: c.Server}
	if err := serverConfig.Validate(); err != nil {
		return fmt.Errorf("onlineconf server config: %w", err)
	}

	if c.ServiceToken == "" || strings.IndexFunc(c.ServiceToken, unicode.IsSpace) >= 0 {
		return errors.New("ONLINECONF_SERVICE_TOKEN is required and must contain no whitespace")
	}

	if c.Snapshot.RefreshInterval <= 0 || c.Snapshot.RefreshInterval > DefaultOnlineConfRefreshInterval {
		return errors.New("onlineconf snapshot refresh_interval must be positive and at most 10s")
	}

	if c.Snapshot.LoadTimeout <= 0 || c.Snapshot.LoadTimeout > DefaultOnlineConfLoadTimeout {
		return errors.New("onlineconf snapshot load_timeout must be positive and at most 2s")
	}

	protection := http.NewCrossOriginProtection()

	for _, origin := range c.Server.AllowedOrigins {
		if err := protection.AddTrustedOrigin(origin); err != nil {
			return fmt.Errorf("onlineconf allowed origin: %w", err)
		}
	}

	return nil
}
