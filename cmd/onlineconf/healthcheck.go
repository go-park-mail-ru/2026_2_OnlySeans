package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/config"
)

const (
	healthcheckTimeout = 2 * time.Second
	healthcheckPath    = "/healthz"
	healthcheckScheme  = "http"
	ipv4Unspecified    = "0.0.0.0"
	ipv4Loopback       = "127.0.0.1"
	ipv6Unspecified    = "::"
	ipv6Loopback       = "::1"
)

func checkHealth(configPath string) (checkErr error) {
	cfg, err := config.LoadOnlineConfServer(configPath)
	if err != nil {
		return fmt.Errorf("load healthcheck config: %w", err)
	}

	host := cfg.Server.Host
	switch host {
	case ipv4Unspecified:
		host = ipv4Loopback
	case ipv6Unspecified:
		host = ipv6Loopback
	}

	target := url.URL{Scheme: healthcheckScheme, Host: net.JoinHostPort(host, strconv.Itoa(cfg.Server.Port)), Path: healthcheckPath}

	ctx, cancel := context.WithTimeout(context.Background(), healthcheckTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return fmt.Errorf("create healthcheck request: %w", err)
	}

	client := http.Client{Timeout: healthcheckTimeout}

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("request onlineconf health: %w", err)
	}

	defer func() {
		if err := response.Body.Close(); err != nil {
			checkErr = errors.Join(checkErr, fmt.Errorf("close healthcheck response: %w", err))
		}
	}()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("onlineconf health status: %d", response.StatusCode)
	}

	return nil
}
