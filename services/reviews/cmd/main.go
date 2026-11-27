package main

import (
	"flag"
	"log"
)

func main() {
	listenAddress := flag.String("listen", defaultListenAddress, "gRPC listen address")
	databaseConfigPath := flag.String("database-config", defaultDatabaseConfigPath, "path to database config file")
	loggerConfigPath := flag.String("logger-config", defaultLoggerConfigPath, "path to logger config file")
	flag.Parse()

	if err := run(*listenAddress, *databaseConfigPath, *loggerConfigPath); err != nil {
		log.Fatal(err)
	}
}
