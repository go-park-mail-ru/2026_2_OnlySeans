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
	EnvOnlineConfHost                      = "ONLINECONF_HOST"
	EnvOnlineConfPort                      = "ONLINECONF_PORT"
	EnvOnlineConfServerServiceToken        = "ONLINECONF_SERVICE_TOKEN"
	EnvOnlineConfAllowedOrigins            = "ONLINECONF_ALLOWED_ORIGINS"
	DefaultOnlineConfServerPort            = 8081
	DefaultOnlineConfServerRefreshInterval = 10 * time.Second
	DefaultOnlineConfServerLoadTimeout     = 2 * time.Second
)

type OnlineConfServerConfig struct {
	Server       Server                   `yaml:"server"`
	Snapshot     OnlineConfServerSnapshot `yaml:"snapshot"`
	ServiceToken string                   `yaml:"-"`
}

type OnlineConfServerSnapshot struct {
	RefreshInterval time.Duration `yaml:"refresh_interval"`
	LoadTimeout     time.Duration `yaml:"load_timeout"`
}

func DefaultOnlineConfServer() *OnlineConfServerConfig {
	server := Default().Server
	server.Port = DefaultOnlineConfServerPort

	return &OnlineConfServerConfig{Server: server, Snapshot: OnlineConfServerSnapshot{RefreshInterval: DefaultOnlineConfServerRefreshInterval, LoadTimeout: DefaultOnlineConfServerLoadTimeout}}
}

func LoadOnlineConfServer(path string) (*OnlineConfServerConfig, error) {
	return loadFile(path, "onlineconf config", ParseOnlineConfServer)
}

func ParseOnlineConfServer(raw []byte, lookupEnv func(string) (string, bool)) (*OnlineConfServerConfig, error) {
	cfg := DefaultOnlineConfServer()
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

	cfg.ServiceToken, _ = lookupEnv(EnvOnlineConfServerServiceToken)

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *OnlineConfServerConfig) Validate() error {
	serverConfig := Config{Server: c.Server}
	if err := serverConfig.Validate(); err != nil {
		return fmt.Errorf("onlineconf server config: %w", err)
	}

	if c.ServiceToken == "" || strings.IndexFunc(c.ServiceToken, unicode.IsSpace) >= 0 {
		return errors.New("ONLINECONF_SERVICE_TOKEN is required and must contain no whitespace")
	}

	if c.Snapshot.RefreshInterval <= 0 || c.Snapshot.RefreshInterval > DefaultOnlineConfServerRefreshInterval {
		return errors.New("onlineconf snapshot refresh_interval must be positive and at most 10s")
	}

	if c.Snapshot.LoadTimeout <= 0 || c.Snapshot.LoadTimeout > DefaultOnlineConfServerLoadTimeout {
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
