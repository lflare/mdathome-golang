package main

import (
	"flag"

	"github.com/lflare/mdathome-golang/internal/mdathome"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func main() {
	// Define arguments
	configFile := flag.String("config", "settings.json", "Location of config.json file")
	printVersion := flag.Bool("version", false, "Prints version of client")
	shrinkDatabase := flag.Bool("shrink-database", false, "Shrink cache.db (may take a long time)")

	// Parse arguments
	flag.Parse()

	// Set configuration file path
	mdathome.ConfigFilePath = *configFile

	// Shrink database if flag given, otherwise start server
	if *printVersion {
		log.Infof("MD@Home Client %s (%d) written in Golang by @lflare", mdathome.ClientVersion, mdathome.ClientSpecification)
	} else if *shrinkDatabase {
		mdathome.ShrinkDatabase()
	} else {
		mdathome.StartServer()
	}
}
