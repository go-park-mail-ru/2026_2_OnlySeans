package config

import (
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/config"
)

type S3Config struct {
	Endpoint             string        `yaml:"endpoint"`
	PublicEndpoint       string        `yaml:"public_endpoint"`
	Region               string        `yaml:"region"`
	Bucket               string        `yaml:"bucket"`
	AccessKey            string        `yaml:"access_key"`
	SecretKey            string        `yaml:"secret_key"`
	AvatarURLLifetime    time.Duration `yaml:"avatar_url_lifetime"`
	MaxAvatarSize        int64         `yaml:"max_avatar_size"`
	MaxAvatarDimension   int           `yaml:"max_avatar_dimension"`
	BucketCheckTimeout   time.Duration `yaml:"bucket_check_timeout"`
	AvatarCleanupTimeout time.Duration `yaml:"avatar_cleanup_timeout"`
}

type Config struct {
	Logger   config.LoggerConfig     `yaml:"logger"`
	Postgres config.PostgresConfig   `yaml:"postgres"`
	GRPC     config.GRPCServerConfig `yaml:"grpc"`
	S3       S3Config                `yaml:"s3"`
}

func Load() *Config {
	var cfg Config
	config.MustLoad(&cfg)
	return &cfg
}
