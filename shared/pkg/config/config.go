package config

import (
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type LoggerConfig struct {
	Level string `yaml:"level"`
}

type PostgresConfig struct {
	URL string `yaml:"url"`
}

type GRPCServerConfig struct {
	Port    string        `yaml:"port"`
	Timeout time.Duration `yaml:"timeout"`
}

type RedisConfig struct {
	URL         string        `yaml:"url"`
	MaxIdle     int           `yaml:"max_idle"`
	IdleTimeout time.Duration `yaml:"idle_timeout"`
}

func MustLoad(cfg interface{}) {
	configPath := os.Getenv("CONFIG_PATH")

	if configPath != "" {
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			log.Fatalf("config file does not exist: %s", configPath)
		}
		if err := cleanenv.ReadConfig(configPath, cfg); err != nil {
			log.Fatalf("cannot read config: %v", err)
		}
	} else {
		if err := cleanenv.ReadEnv(cfg); err != nil {
			log.Fatalf("cannot read env: %v", err)
		}
	}
}
