package main

import (
	_ "embed"
	"flag"
	"os"

	"github.com/lflare/mdathome-golang/internal/mdathome"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

//go:embed assets/config.example.toml
var defaultConfiguration string

var log *logrus.Logger

func loadConfiguration() {
	log.Infof("%+v", viper.AllSettings())
}

func init() {
	// Initialise logger
	log = logrus.New()

	// Configure Viper
	viper.AddConfigPath(".")
	viper.SetConfigName("config.toml")
	viper.SetConfigType("toml")

	// Load in configuration
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			log.Info("Configuration not found, creating!")
			if err := os.WriteFile("config.toml", []byte(defaultConfiguration), 0600); err != nil {
				log.Fatalf("Failed to write default configuration to 'config.toml'!")
			} else {
				log.Fatalf("Default configuration written to 'config.toml', please modify before running client again!")
			}
		} else {
			// Config file was found but another error was produced
			log.Errorf("Failed to read configuration: %v", err)
		}
	}

	// Reload configuration
	loadConfiguration()
}

func main() {
	// Define arguments
	printVersion := flag.Bool("version", false, "Prints version of client")
	shrinkDatabase := flag.Bool("shrink-database", false, "Shrink cache.db (may take a long time)")

	// Parse arguments
	flag.Parse()

	// Shrink database if flag given, otherwise start server
	if *printVersion {
		log.Infof("MD@Home Client %s (%d) written in Golang by @lflare", mdathome.ClientVersion, mdathome.ClientSpecification)
	} else if *shrinkDatabase {
		mdathome.ShrinkDatabase()
	} else {
		mdathome.StartServer()
	}
}
