package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/rueidis"
)

const (
	pingTimeout = 3 * time.Second
)

func NewClientWithPing(ctx context.Context, redisURL string) (rueidis.Client, error) {
	options, err := rueidis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}

	client, err := rueidis.NewClient(options)
	if err != nil {
		return nil, fmt.Errorf("connect to redis: %w", err)
	}

	var success bool
	defer func() {
		if !success {
			client.Close()
		}
	}()

	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()

	res := client.Do(pingCtx, client.B().Ping().Build())

	resultErr := res.Error()
	if resultErr != nil {
		client.Close()
		return nil, fmt.Errorf("ping redis: %w", resultErr)
	}

	success = true

	return client, nil
}
