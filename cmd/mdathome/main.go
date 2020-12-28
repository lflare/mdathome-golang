package main

import (
	"flag"

	"github.com/lflare/mdathome-golang/internal/mdathome"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func main() {
	// Get arguments
	shrinkPtr := flag.Bool("shrink-database", false, "Shrink cache.db (may take a long time)")
	flag.Parse()

	// Shrink database if flag given, otherwise start server
	if *shrinkPtr {
		mdathome.ShrinkDatabase()
	} else {
		mdathome.StartServer()
	}
}
