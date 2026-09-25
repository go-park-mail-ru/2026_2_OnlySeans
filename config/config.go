package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	EnvServerHost = "SERVER_HOST"
	EnvServerPort = "SERVER_PORT"
)

type Config struct {
	Server Server `yaml:"server"`
}

type Server struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	IdleTimeout     time.Duration `yaml:"idle_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

func (s Server) Addr() string {
	return net.JoinHostPort(s.Host, strconv.Itoa(s.Port))
}

func Default() Config {
	return Config{
		Server: Server{
			Host:            "0.0.0.0",
			Port:            8080,
			ReadTimeout:     5 * time.Second,
			WriteTimeout:    10 * time.Second,
			IdleTimeout:     60 * time.Second,
			ShutdownTimeout: 10 * time.Second,
		},
	}
}

func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("чтение конфигурации: %w", err)
	}
	return Parse(raw, os.LookupEnv)
}

func Parse(raw []byte, lookupEnv func(string) (string, bool)) (Config, error) {
	cfg := Default()

	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil && !errors.Is(err, io.EOF) {
		return Config{}, fmt.Errorf("разбор конфигурации: %w", err)
	}

	if err := cfg.applyEnv(lookupEnv); err != nil {
		return Config{}, err
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c *Config) applyEnv(lookupEnv func(string) (string, bool)) error {
	if host, ok := lookupEnv(EnvServerHost); ok {
		c.Server.Host = host
	}
	if rawPort, ok := lookupEnv(EnvServerPort); ok {
		port, err := strconv.Atoi(rawPort)
		if err != nil {
			return fmt.Errorf("%s должен быть числом, получено %q", EnvServerPort, rawPort)
		}
		c.Server.Port = port
	}
	return nil
}

func (c Config) Validate() error {
	var errs []error

	if strings.TrimSpace(c.Server.Host) == "" {
		errs = append(errs, errors.New("server.host не задан"))
	}
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		errs = append(errs, fmt.Errorf("server.port должен быть в диапазоне 1–65535, получено %d", c.Server.Port))
	}

	timeouts := []struct {
		name  string
		value time.Duration
	}{
		{"server.read_timeout", c.Server.ReadTimeout},
		{"server.write_timeout", c.Server.WriteTimeout},
		{"server.idle_timeout", c.Server.IdleTimeout},
		{"server.shutdown_timeout", c.Server.ShutdownTimeout},
	}
	for _, t := range timeouts {
		if t.value <= 0 {
			errs = append(errs, fmt.Errorf("%s должен быть больше нуля", t.name))
		}
	}

	return errors.Join(errs...)
}
