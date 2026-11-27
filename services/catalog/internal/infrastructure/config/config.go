package config

import (
	"errors"
	"net"
	"strconv"
	"strings"
	"time"

	sharedconfig "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/config"
)

type S3Config struct {
	Endpoint             string        `yaml:"endpoint" env:"S3_ENDPOINT"`
	PublicEndpoint       string        `yaml:"public_endpoint" env:"S3_PUBLIC_ENDPOINT"`
	Region               string        `yaml:"region" env:"S3_REGION"`
	Bucket               string        `yaml:"bucket" env:"S3_BUCKET"`
	AccessKey            string        `yaml:"access_key" env:"RUSTFS_ACCESS_KEY"`
	SecretKey            string        `yaml:"secret_key" env:"RUSTFS_SECRET_KEY"`
	PosterURLLifetime    time.Duration `yaml:"poster_url_lifetime"`
	MaxPosterSize        int64         `yaml:"max_poster_size"`
	MaxPosterDimension   int           `yaml:"max_poster_dimension"`
	BucketCheckTimeout   time.Duration `yaml:"bucket_check_timeout"`
	PosterCleanupTimeout time.Duration `yaml:"poster_cleanup_timeout"`
}

type Config struct {
	PersonsServiceAddr string                        `yaml:"persons_service_addr" env:"PERSONS_SERVICE_ADDR"`
	Logger             sharedconfig.LoggerConfig     `yaml:"logger"`
	Postgres           sharedconfig.PostgresConfig   `yaml:"postgres"`
	GRPC               sharedconfig.GRPCServerConfig `yaml:"grpc"`
	S3                 S3Config                      `yaml:"s3"`
	ShutdownTimeout    time.Duration                 `yaml:"shutdown_timeout"`
}

func Default() *Config {
	return &Config{
		PersonsServiceAddr: "127.0.0.1:50053",
		Logger: sharedconfig.LoggerConfig{
			Level: "info",
		},
		GRPC: sharedconfig.GRPCServerConfig{
			Host: "127.0.0.1",
			Port: "50054",
		},
		S3: S3Config{
			Endpoint:             "http://localhost:8050",
			PublicEndpoint:       "http://localhost:8050",
			Region:               "us-east-1",
			Bucket:               "avatars",
			PosterURLLifetime:    15 * time.Minute,
			MaxPosterSize:        5 << 20,
			MaxPosterDimension:   4096,
			BucketCheckTimeout:   5 * time.Second,
			PosterCleanupTimeout: 5 * time.Second,
		},
		ShutdownTimeout: 10 * time.Second,
	}
}

func Load() *Config {
	cfg := Default()
	sharedconfig.MustLoad(cfg)

	return cfg
}

func (c *Config) Validate() error {
	if strings.TrimSpace(c.PersonsServiceAddr) == "" {
		return errors.New("persons_service_addr is required")
	}
	if strings.TrimSpace(c.GRPC.Host) == "" {
		return errors.New("grpc.host is required")
	}

	port, err := strconv.Atoi(c.GRPC.Port)
	if err != nil || port < 1 || port > 65535 {
		return errors.New("grpc.port must be between 1 and 65535")
	}

	if c.ShutdownTimeout <= 0 {
		return errors.New("shutdown_timeout must be positive")
	}

	for _, field := range []struct {
		name  string
		value string
	}{
		{
			name:  "s3.endpoint",
			value: c.S3.Endpoint,
		},
		{
			name:  "s3.region",
			value: c.S3.Region,
		},
		{
			name:  "s3.bucket",
			value: c.S3.Bucket,
		},
		{
			name:  "s3.access_key",
			value: c.S3.AccessKey,
		},
		{
			name:  "s3.secret_key",
			value: c.S3.SecretKey,
		},
	} {
		if strings.TrimSpace(field.value) == "" {
			return errors.New(field.name + " is required")
		}
	}

	if c.S3.PosterURLLifetime < time.Second || c.S3.PosterURLLifetime > 7*24*time.Hour {
		return errors.New("s3.poster_url_lifetime must be between 1s and 168h")
	}

	if c.S3.BucketCheckTimeout <= 0 {
		return errors.New("s3.bucket_check_timeout must be positive")
	}

	if c.S3.MaxPosterSize <= 0 || c.S3.MaxPosterSize > int64(int(^uint(0)>>1))-(1<<20) {
		return errors.New("s3.max_poster_size must be positive and leave room for message overhead")
	}

	if c.S3.MaxPosterDimension <= 0 || c.S3.PosterCleanupTimeout <= 0 {
		return errors.New("poster dimension and cleanup timeout must be positive")
	}

	return nil
}

func (c *Config) Address() string {
	return net.JoinHostPort(c.GRPC.Host, c.GRPC.Port)
}
