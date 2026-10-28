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
	EnvServerHost  = "SERVER_HOST"
	EnvServerPort  = "SERVER_PORT"
	EnvDatabaseURL = "DATABASE_URL"
)

const (
	MinPort = 1
	MaxPort = 65535
)

const (
	defaultHost            = "0.0.0.0"
	defaultPort            = 8080
	defaultReadTimeout     = 5 * time.Second
	defaultWriteTimeout    = 10 * time.Second
	defaultIdleTimeout     = 60 * time.Second
	defaultShutdownTimeout = 10 * time.Second
	defaultSessionTTL      = 24 * time.Hour
	defaultAllowedOrigin   = "http://localhost:3000"

	defaultMaxOpenConns    = 10               //максимум соединений с базой
	defaultMaxIdleConns    = 5                //сколько из них откытыми прозапас когда запросов нет
	defaultConnMaxLifetime = 30 * time.Minute //через сколько пересоздавать соединение чтобы зависшие не копились
	defaultConnectTimeout  = 5 * time.Second  //сколько ждать продключения при запуске
)

type Config struct {
	Server Server `yaml:"server"`
}

type Server struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	AllowedOrigin   string        `yaml:"allowed_origin"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	IdleTimeout     time.Duration `yaml:"idle_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
	SessionTTL      time.Duration `yaml:"session_ttl"`
}

type Database struct {
	URL             string        `yaml:"url"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
	ConnectTimeout  time.Duration `yaml:"connect_timeout"`
}

func (s *Server) Addr() string {
	return net.JoinHostPort(s.Host, strconv.Itoa(s.Port))
}

func Default() *Config {
	return &Config{
		Server: Server{
			Host:            defaultHost,
			Port:            defaultPort,
			AllowedOrigin:   defaultAllowedOrigin,
			ReadTimeout:     defaultReadTimeout,
			WriteTimeout:    defaultWriteTimeout,
			IdleTimeout:     defaultIdleTimeout,
			ShutdownTimeout: defaultShutdownTimeout,
			SessionTTL:      defaultSessionTTL,
		},
	}
}

func DefaultDatabase() *Database {
	return &Database{
		MaxOpenConns:    defaultMaxOpenConns,
		MaxIdleConns:    defaultMaxIdleConns,
		ConnMaxLifetime: defaultConnMaxLifetime,
		ConnectTimeout:  defaultConnectTimeout,
	}
}

func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	return Parse(raw, os.LookupEnv)
}

func Parse(raw []byte, lookupEnv func(string) (string, bool)) (*Config, error) {
	cfg := Default()

	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	if err := decoder.Decode(cfg); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := cfg.applyEnv(lookupEnv); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func LoadDatabase(path string) (*Database, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read database config: %w", err)
	}
	return ParseDatabase(raw, os.LookupEnv)
}

func ParseDatabase(raw []byte, lookupEnv func(string) (string, bool)) (*Database, error) {
	cfg := DefaultDatabase()

	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	if err := decoder.Decode(cfg); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("parse database config: %w", err)
	}

	if dbURL, ok := lookupEnv(EnvDatabaseURL); ok {
		cfg.URL = dbURL
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// проверяет настройки пула соединений
func (d *Database) Validate() error {
	var errs []error

	if d.MaxOpenConns < 1 {
		errs = append(errs, errors.New("database.max_open_conns must be positive"))
	}

	if d.MaxIdleConns < 0 || d.MaxIdleConns > d.MaxOpenConns {
		errs = append(errs, errors.New("database.max_idle_conns must be between 0 and database.max_open_conns"))
	}

	timeouts := []struct {
		name  string
		value time.Duration
	}{
		{
			name:  "database.conn_max_lifetime",
			value: d.ConnMaxLifetime,
		},
		{
			name:  "database.connect_timeout",
			value: d.ConnectTimeout,
		},
	}
	for _, t := range timeouts {
		if t.value <= 0 {
			errs = append(errs, fmt.Errorf("%s must be positive", t.name))
		}
	}
	return errors.Join(errs...)
}

func (c *Config) applyEnv(lookupEnv func(string) (string, bool)) error {
	if host, ok := lookupEnv(EnvServerHost); ok {
		c.Server.Host = host
	}
	if rawPort, ok := lookupEnv(EnvServerPort); ok {
		port, err := strconv.Atoi(rawPort)
		if err != nil {
			return fmt.Errorf("%s must be an integer, got %q", EnvServerPort, rawPort)
		}
		c.Server.Port = port
	}
	return nil
}

func (c *Config) Validate() error {
	var errs []error

	if strings.TrimSpace(c.Server.Host) == "" {
		errs = append(errs, errors.New("server.host is required"))
	}

	if strings.TrimSpace(c.Server.AllowedOrigin) == "" {
		errs = append(errs, errors.New("server.allowed_origin is required"))
	}

	if c.Server.Port < MinPort || c.Server.Port > MaxPort {
		errs = append(errs, fmt.Errorf("server.port must be between %d and %d, got %d", MinPort, MaxPort, c.Server.Port))
	}

	timeouts := []struct {
		name  string
		value time.Duration
	}{
		{
			name:  "server.read_timeout",
			value: c.Server.ReadTimeout,
		},
		{
			name:  "server.write_timeout",
			value: c.Server.WriteTimeout,
		},
		{
			name:  "server.idle_timeout",
			value: c.Server.IdleTimeout,
		},
		{
			name:  "server.shutdown_timeout",
			value: c.Server.ShutdownTimeout,
		},
		{
			name:  "server.session_ttl",
			value: c.Server.SessionTTL,
		},
	}
	for _, t := range timeouts {
		if t.value <= 0 {
			errs = append(errs, fmt.Errorf("%s must be positive", t.name))
		}
	}

	return errors.Join(errs...)
}
