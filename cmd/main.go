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
)

func main() {
	configPath := flag.String("config", defaultConfigPath, "path to config file")
	databaseConfigPath := flag.String("database-config", defaultDatabaseConfigPath, "path to database config file")
	redisConfigPath := flag.String("redis-config", defaultRedisConfigPath, "path to redis config file")
	flag.Parse()

	if err := app.Run(*configPath, *databaseConfigPath, *redisConfigPath); err != nil {
		log.Fatal(err)
	}
}
