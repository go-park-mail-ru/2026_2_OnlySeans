package config

import sharedconfig "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/config"

type HTTPConfig struct {
	Addr           string   `yaml:"addr"`
	AllowedOrigins []string `yaml:"allowed_origins"`
	CookieSecure   bool     `yaml:"cookie_secure"`
}

type Config struct {
	Logger             sharedconfig.LoggerConfig `yaml:"logger"`
	HTTP               HTTPConfig                `yaml:"http"`
	AuthServiceAddr    string                    `yaml:"auth_service_addr"`
	UserServiceAddr    string                    `yaml:"user_service_addr"`
	PersonsServiceAddr string                    `yaml:"persons_service_addr"`
}

func Load() *Config {
	var cfg Config
	sharedconfig.MustLoad(&cfg)
	return &cfg
}
