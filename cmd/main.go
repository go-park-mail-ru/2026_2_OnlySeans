package main

import (
	"flag"
	"log"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/app"
)

const defaultConfigPath = "configs/config.yaml"
const defaultDatabaseConfigPath = "configs/database.yaml"

func main() {
	configPath := flag.String("config", defaultConfigPath, "path to config file")
	databaseConfigPath := flag.String("database-config", defaultDatabaseConfigPath, "path to database config file")

	flag.Parse()

	if err := app.Run(*configPath, *databaseConfigPath); err != nil {
		log.Fatal(err)
	}
}
