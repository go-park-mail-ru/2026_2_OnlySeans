package config

import (
	"time"

	sharedconfig "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/config"
)

type Config struct {
	Logger          sharedconfig.LoggerConfig     `yaml:"logger"`
	Redis           sharedconfig.RedisConfig      `yaml:"redis"`
	GRPC            sharedconfig.GRPCServerConfig `yaml:"grpc"`
	UserServiceAddr string                        `yaml:"user_service_addr"`
	SessionTTL      time.Duration                 `yaml:"session_ttl"`
}

func Load() *Config {
	var cfg Config
	sharedconfig.MustLoad(&cfg)
	return &cfg
}
