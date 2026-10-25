package app_test

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/app"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/config"
)

func TestRun_InvalidConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("server:\n  port: 0\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := app.Run(path); err == nil {
		t.Error("expected error for invalid config")
	}
}

func TestRun_PortInUse(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	path := filepath.Join(t.TempDir(), "config.yaml")
	raw := fmt.Sprintf("server:\n  host: 127.0.0.1\n  port: %d\n", port)
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- app.Run(path) }()

	select {
	case err := <-done:
		if err == nil {
			t.Error("expected error when port is already in use")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return on busy port")
	}
}

func TestRun_MissingConfig(t *testing.T) {
	if err := app.Run(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Error("expected error for missing config")
	}
}

func TestRun_DatabaseUnavailable(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	closedAddr := listener.Addr().String()
	listener.Close()

	t.Setenv(config.EnvDatabaseURL, "postgres://user@"+closedAddr+"/kinopoisk?sslmode=disable")
	path := filepath.Join(t.TempDir(), "config.yaml")
	raw := "server:\n  host: 127.0.0.1\n  port: 1\ndatabase:\n  connect_timeout: 2s\n"
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- app.Run(path) }()

	select {
	case err := <-done:
		if err == nil {
			t.Error("expected error when the database is unavailable")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not give up on an unavailable database")
	}
}
