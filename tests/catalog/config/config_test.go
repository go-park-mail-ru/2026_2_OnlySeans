package config_test

import (
	"testing"
	"time"

	config "github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/infrastructure/config"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/stretchr/testify/require"
)

func TestConfigValidation(t *testing.T) {
	cases := []struct {
		name   string
		change func(*config.Config)
		want   string
	}{
		{
			name: "valid",
		},
		{
			name:   "port",
			change: func(c *config.Config) { c.GRPC.Port = "0" },
			want:   "grpc.port",
		},
		{
			name:   "host",
			change: func(c *config.Config) { c.GRPC.Host = "" },
			want:   "grpc.host",
		},
		{
			name:   "shutdown",
			change: func(c *config.Config) { c.ShutdownTimeout = 0 },
			want:   "shutdown_timeout",
		},
		{
			name:   "bucket timeout",
			change: func(c *config.Config) { c.S3.BucketCheckTimeout = 0 },
			want:   "bucket_check_timeout",
		},
		{
			name:   "file size",
			change: func(c *config.Config) { c.S3.MaxPosterSize = -1 },
			want:   "max_poster_size",
		},
		{
			name:   "image dimension",
			change: func(c *config.Config) { c.S3.MaxPosterDimension = 0 },
			want:   "poster dimension",
		},
		{
			name:   "URL lifetime",
			change: func(c *config.Config) { c.S3.PosterURLLifetime = 8 * 24 * time.Hour },
			want:   "poster_url_lifetime",
		},
		{
			name:   "credentials",
			change: func(c *config.Config) { c.S3.AccessKey = "" },
			want:   "access_key",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Default()
			cfg.S3.AccessKey = "test-access"
			cfg.S3.SecretKey = "test-secret"

			if tc.change != nil {
				tc.change(cfg)
			}

			err := cfg.Validate()
			if tc.want == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.want)
			}
		})
	}
}

func TestConfigFile(t *testing.T) {
	cases := []struct {
		name    string
		host    string
		address string
	}{
		{
			name:    "default listener",
			host:    "127.0.0.1",
			address: "127.0.0.1:50054",
		},
		{
			name:    "container listener",
			host:    "0.0.0.0",
			address: "0.0.0.0:50054",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("GRPC_HOST", tc.host)
			t.Setenv("RUSTFS_ACCESS_KEY", "test-access")
			t.Setenv("RUSTFS_SECRET_KEY", "test-secret")

			cfg := config.Default()
			require.NoError(t, cleanenv.ReadConfig("../../../services/catalog/config.yaml", cfg))
			require.NoError(t, cfg.Validate())
			require.Equal(t, tc.address, cfg.Address())
			require.Equal(t, int64(5<<20), cfg.S3.MaxPosterSize)
			require.Equal(t, 4096, cfg.S3.MaxPosterDimension)
			require.Equal(t, 15*time.Minute, cfg.S3.PosterURLLifetime)
			require.Equal(t, 10*time.Second, cfg.ShutdownTimeout)
		})
	}
}
