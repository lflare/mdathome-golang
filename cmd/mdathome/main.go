package main

import (
	"flag"

	"github.com/lflare/mdathome-golang/internal/mdathome"
)

func main() {
	// Prepare logging
	logFile := mdathome.GetLogWriter()
	defer logFile.Close()

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
