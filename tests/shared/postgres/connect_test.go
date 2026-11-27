package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/postgres"
)

func TestConnectRejectsInvalidOrCancelledRequests(t *testing.T) {
	tests := []struct {
		name      string
		dsn       string
		timeout   time.Duration
		cancelled bool
		message   string
	}{
		{
			name:    "invalid DSN",
			dsn:     "postgres://user@host:not-a-port/db",
			message: "invalid port",
		},
		{
			name:      "cancelled context",
			dsn:       "postgres://user@localhost:1/db?sslmode=disable",
			cancelled: true,
			message:   "ping db",
		},
		{
			name:    "expired deadline",
			dsn:     "postgres://user@localhost:1/db?sslmode=disable",
			timeout: -time.Second,
			message: "ping db",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			if tt.timeout != 0 {
				cancel()
				ctx, cancel = context.WithTimeout(t.Context(), tt.timeout)
			}

			defer cancel()

			if tt.cancelled {
				cancel()
			}

			db, err := postgres.Connect(ctx, tt.dsn)
			require.ErrorContains(t, err, tt.message)
			require.Nil(t, db)
		})
	}
}
