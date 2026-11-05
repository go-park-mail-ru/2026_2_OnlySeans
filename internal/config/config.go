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

	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"gopkg.in/yaml.v3"
)

const (
	EnvServerHost  = "SERVER_HOST"
	EnvServerPort  = "SERVER_PORT"
	EnvDatabaseURL = "DATABASE_URL"
	EnvRedisURL    = "REDIS_URL"
	EnvLogLevel    = "LOG_LEVEL"
	EnvLogFormat   = "LOG_FORMAT"

	EnvS3Endpoint       = "S3_ENDPOINT"
	EnvS3PublicEndpoint = "S3_PUBLIC_ENDPOINT"
	EnvS3Region         = "S3_REGION"
	EnvS3Bucket         = "S3_BUCKET"
	EnvS3AccessKey      = "RUSTFS_ACCESS_KEY"
	EnvS3SecretKey      = "RUSTFS_SECRET_KEY"
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

	defaultRedisURL = "redis://localhost:6379"

	defaultLogLvl    = logger.LogLevelInfo
	defaultLogFormat = "text"

	defaultEndpoint = "http://localhost:9000"
	defaultRegion   = "us-east-1"
	defaultBucket   = "avatars"
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

type Redis struct {
	URL string `yaml:"url"`
}

type S3 struct {
	Endpoint       string `yaml:"endpoint"`
	PublicEndpoint string `yaml:"public_endpoint"`
	Region         string `yaml:"region"`
	Bucket         string `yaml:"bucket"`
	AccessKey      string `yaml:"access_key"`
	SecretKey      string `yaml:"secret_key"`
}

type LoggerConfig struct {
	Level  logger.LogLevelType `yaml:"level"`
	Format string              `yaml:"format"`
	File   string              `yaml:"file"`
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

func DefaultRedis() *Redis {
	return &Redis{URL: defaultRedisURL}
}

func DefaultS3() *S3 {
	return &S3{
		Endpoint: defaultEndpoint,
		Region:   defaultRegion,
		Bucket:   defaultBucket,
	}
}

func DefaultLogger() *LoggerConfig {
	return &LoggerConfig{
		Level:  defaultLogLvl,
		Format: defaultLogFormat,
		File:   "logs/app.log",
	}
}

func Load(path string) (*Config, error) {
	return loadFile(path, "config", Parse)
}

func LoadDatabase(path string) (*Database, error) {
	return loadFile(path, "database config", ParseDatabase)
}

func LoadRedis(path string) (*Redis, error) {
	return loadFile(path, "redis config", ParseRedis)
}

func LoadS3(path string) (*S3, error) {
	return loadFile(path, "s3 config", ParseS3)
}

func LoadLogger(path string) (*LoggerConfig, error) {
	return loadFile(path, "logger config", ParseLogger)
}

func Parse(raw []byte, lookupEnv func(string) (string, bool)) (*Config, error) {
	cfg := Default()

	if err := decodeStrict(raw, cfg, "config"); err != nil {
		return nil, err
	}

	if err := cfg.applyEnv(lookupEnv); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func ParseDatabase(raw []byte, lookupEnv func(string) (string, bool)) (*Database, error) {
	cfg := DefaultDatabase()

	if err := decodeStrict(raw, cfg, "database config"); err != nil {
		return nil, err
	}

	if dbURL, ok := lookupEnv(EnvDatabaseURL); ok {
		cfg.URL = dbURL
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func ParseRedis(raw []byte, lookupEnv func(string) (string, bool)) (*Redis, error) {
	cfg := DefaultRedis()

	if err := decodeStrict(raw, cfg, "redis config"); err != nil {
		return nil, err
	}

	if redisURL, ok := lookupEnv(EnvRedisURL); ok {
		cfg.URL = redisURL
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func ParseS3(raw []byte, lookupEnv func(string) (string, bool)) (*S3, error) {
	cfg := DefaultS3()

	if err := decodeStrict(raw, cfg, "s3 config"); err != nil {
		return nil, err
	}

	if s3Endpoint, ok := lookupEnv(EnvS3Endpoint); ok {
		cfg.Endpoint = s3Endpoint
	}
	if publicEndpoint, ok := lookupEnv(EnvS3PublicEndpoint); ok {
		cfg.PublicEndpoint = publicEndpoint
	}

	if region, ok := lookupEnv(EnvS3Region); ok {
		cfg.Region = region
	}

	if bucket, ok := lookupEnv(EnvS3Bucket); ok {
		cfg.Bucket = bucket
	}

	if accessKey, ok := lookupEnv(EnvS3AccessKey); ok {
		cfg.AccessKey = accessKey
	}

	if secretKey, ok := lookupEnv(EnvS3SecretKey); ok {
		cfg.SecretKey = secretKey
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func ParseLogger(raw []byte, lookupEnv func(string) (string, bool)) (*LoggerConfig, error) {
	cfg := DefaultLogger()

	if err := decodeStrict(raw, cfg, "logger config"); err != nil {
		return nil, err
	}

	if loggerLvl, ok := lookupEnv(EnvLogLevel); ok {
		cfg.Level = logger.LogLevelType(loggerLvl)
	}

	if loggerFormat, ok := lookupEnv(EnvLogFormat); ok {
		cfg.Format = loggerFormat
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

func (r *Redis) Validate() error {
	if strings.TrimSpace(r.URL) == "" {
		return errors.New("redis.url is required")
	}
	return nil
}

func (s *S3) Validate() error {
	if strings.TrimSpace(s.Endpoint) == "" {
		return errors.New("s3.endpoint is required")
	}

	if strings.TrimSpace(s.Region) == "" {
		return errors.New("s3.region is required")
	}

	if strings.TrimSpace(s.Bucket) == "" {
		return errors.New("s3.bucket is required")
	}

	if strings.TrimSpace(s.AccessKey) == "" {
		return errors.New("s3.access_key is required")
	}

	if strings.TrimSpace(s.SecretKey) == "" {
		return errors.New("s3.secret_key is required")
	}

	return nil
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

func loadFile[T any](
	path, name string,
	parse func(raw []byte, lookupEnv func(string) (string, bool)) (*T, error),
) (*T, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}
	return parse(raw, os.LookupEnv)
}

func decodeStrict(raw []byte, dst any, name string) error {
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	if err := decoder.Decode(dst); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("parse %s: %w", name, err)
	}
	return nil
}
