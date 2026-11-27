package config

import sharedconfig "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/config"

type HTTPConfig struct {
	Addr           string   `yaml:"addr"`
	AllowedOrigins []string `yaml:"allowed_origins"`
	CookieSecure   bool     `yaml:"cookie_secure"`
	MaxPosterSize  int64    `yaml:"max_poster_size"`
}

type S3Config struct {
	Endpoint  string `yaml:"endpoint" env:"S3_ENDPOINT"`
	Region    string `yaml:"region" env:"S3_REGION"`
	Bucket    string `yaml:"bucket" env:"S3_BUCKET"`
	AccessKey string `yaml:"access_key" env:"RUSTFS_ACCESS_KEY"`
	SecretKey string `yaml:"secret_key" env:"RUSTFS_SECRET_KEY"`
}

type Config struct {
	S3                 S3Config                  `yaml:"s3"`
	Logger             sharedconfig.LoggerConfig `yaml:"logger"`
	HTTP               HTTPConfig                `yaml:"http"`
	AuthServiceAddr    string                    `yaml:"auth_service_addr"`
	UserServiceAddr    string                    `yaml:"user_service_addr"`
	PersonsServiceAddr string                    `yaml:"persons_service_addr"`
	CatalogServiceAddr string                    `yaml:"catalog_service_addr"`
}

func Load() *Config {
	cfg := Config{
		HTTP: HTTPConfig{
			MaxPosterSize: 5 << 20,
		},
	}
	sharedconfig.MustLoad(&cfg)

	return &cfg
}
