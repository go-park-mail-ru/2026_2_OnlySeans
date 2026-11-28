package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/app"
)

const (
	defaultConfigPath         = "configs/onlineconf.yaml"
	defaultDatabaseConfigPath = "configs/database.yaml"
	defaultRedisConfigPath    = "configs/redis.yaml"
	defaultLoggerConfigPath   = "configs/logger.yaml"
)

func main() {
	configPath := flag.String("config", defaultConfigPath, "path to OnlineConf config file")
	databasePath := flag.String("database-config", defaultDatabaseConfigPath, "path to database config file")
	redisPath := flag.String("redis-config", defaultRedisConfigPath, "path to Redis config file")
	loggerPath := flag.String("logger-config", defaultLoggerConfigPath, "path to logger config file")
	flag.Parse()

	if err := run(*configPath, *databasePath, *redisPath, *loggerPath); err != nil {
		log.Fatal(err)
	}
}

func run(configPath, databasePath, redisPath, loggerPath string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.RunOnlineConf(ctx, configPath, databasePath, redisPath, loggerPath); err != nil {
		return err
	}

	return nil
}
