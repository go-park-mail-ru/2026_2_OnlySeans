package main

import (
	"flag"
	"log"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/app"
)

const (
	defaultConfigPath         = "configs/config.yaml"
	defaultDatabaseConfigPath = "configs/database.yaml"
	defaultRedisConfigPath    = "configs/redis.yaml"
	defaultLoggerConfigPath   = "configs/logger.yaml"
	defaultS3ConfigPath       = "configs/s3.yaml"
)

func main() {
	configPath := flag.String("config", defaultConfigPath, "path to config file")
	databaseConfigPath := flag.String("database-config", defaultDatabaseConfigPath, "path to database config file")
	redisConfigPath := flag.String("redis-config", defaultRedisConfigPath, "path to redis config file")
	s3ConfigPath := flag.String("s3-config", defaultS3ConfigPath, "path to s3 config file")
	loggerConfigPath := flag.String("logger-config", defaultLoggerConfigPath, "path to logger config file")
	flag.Parse()

	if err := app.Run(*configPath, *databaseConfigPath, *redisConfigPath, *s3ConfigPath, *loggerConfigPath); err != nil {
		log.Fatal(err)
	}
}
