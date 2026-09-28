package main

import (
	"flag"
	"log"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/app"
)

const defaultConfigPath = "configs/config.yaml"

func main() {
	configPath := flag.String("config", defaultConfigPath, "path to config file")
	flag.Parse()

	if err := app.Run(*configPath); err != nil {
		log.Fatal(err)
	}
}
