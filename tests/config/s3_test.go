package config_test

import (
	"strings"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/config"
	"github.com/stretchr/testify/require"
)

func TestParseS3(t *testing.T) {
	const required = "endpoint: http://localhost:9000\nregion: us-east-1\nbucket: avatars\n"
	credentials := env(map[string]string{
		config.EnvS3AccessKey: "test-key",
		config.EnvS3SecretKey: "test-secret",
	})
	tests := []struct {
		name    string
		raw     string
		wantErr string
	}{
		{
			name: "required connection settings with default limits",
			raw:  required,
		},
		{
			name:    "missing endpoint is not replaced with default",
			raw:     strings.ReplaceAll(required, "endpoint: http://localhost:9000\n", ""),
			wantErr: "s3.endpoint is required",
		},
		{
			name:    "missing region is not replaced with default",
			raw:     strings.ReplaceAll(required, "region: us-east-1\n", ""),
			wantErr: "s3.region is required",
		},
		{
			name:    "missing bucket is not replaced with default",
			raw:     strings.ReplaceAll(required, "bucket: avatars\n", ""),
			wantErr: "s3.bucket is required",
		},
		{
			name: "custom limits and timeouts",
			raw:  required + "avatar_url_lifetime: 2m\nmax_avatar_size: 1024\nmax_avatar_dimension: 64\nbucket_check_timeout: 1s\navatar_cleanup_timeout: 2s\n",
		},
		{
			name:    "zero URL lifetime",
			raw:     required + "avatar_url_lifetime: 0s\n",
			wantErr: "s3.avatar_url_lifetime",
		},
		{
			name:    "URL lifetime exceeds signing limit",
			raw:     required + "avatar_url_lifetime: 169h\n",
			wantErr: "s3.avatar_url_lifetime",
		},
		{
			name:    "zero size",
			raw:     required + "max_avatar_size: 0\n",
			wantErr: "s3.max_avatar_size",
		},
		{
			name:    "size overflows HTTP body limit",
			raw:     required + "max_avatar_size: 9223372036854775807\n",
			wantErr: "s3.max_avatar_size",
		},
		{
			name:    "negative dimension",
			raw:     required + "max_avatar_dimension: -1\n",
			wantErr: "s3.max_avatar_dimension",
		},
		{
			name:    "zero bucket check timeout",
			raw:     required + "bucket_check_timeout: 0s\n",
			wantErr: "s3.bucket_check_timeout",
		},
		{
			name:    "negative cleanup timeout",
			raw:     required + "avatar_cleanup_timeout: -1s\n",
			wantErr: "s3.avatar_cleanup_timeout",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := config.ParseS3([]byte(tt.raw), credentials)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, "test-key", cfg.AccessKey)
			require.Equal(t, "test-secret", cfg.SecretKey)
			if tt.name == "custom limits and timeouts" {
				require.Equal(t, 2*time.Minute, cfg.AvatarURLLifetime)
				require.Equal(t, int64(1024), cfg.MaxAvatarSize)
				require.Equal(t, 64, cfg.MaxAvatarDimension)
				require.Equal(t, time.Second, cfg.BucketCheckTimeout)
				require.Equal(t, 2*time.Second, cfg.AvatarCleanupTimeout)
			} else {
				require.Equal(t, 15*time.Minute, cfg.AvatarURLLifetime)
				require.Equal(t, int64(5<<20), cfg.MaxAvatarSize)
				require.Equal(t, 4096, cfg.MaxAvatarDimension)
				require.Equal(t, 5*time.Second, cfg.BucketCheckTimeout)
				require.Equal(t, 5*time.Second, cfg.AvatarCleanupTimeout)
			}
		})
	}
}
