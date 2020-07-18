package main

import (
	"io"
	"log"
	"os"
	"time"
)

// Declares a prefix writer for enhancing logs
type prefixWriter struct {
	f func() string
	w io.Writer
}

func (p prefixWriter) Write(b []byte) (n int, err error) {
	if n, err = p.w.Write([]byte(p.f())); err != nil {
		return
	}
	nn, err := p.w.Write(b)
	return n + nn, err
}

func GetLogWriter() *os.File {
	// Create log directory if it does not exist
	err := os.MkdirAll("log", os.ModePerm)
	if err != nil {
		log.Fatalf("Failed to create log/ directory: %v", err)
	}

	// Open file handler and return file handler for defer
	file, err := os.OpenFile("log/latest.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log/latest.log: %v", err)
	}

	// Set logging parameters
	logWriter := io.MultiWriter(os.Stdout, file)
	writer := prefixWriter{
		f: func() string { return time.Now().Format(time.RFC3339) + " " },
		w: logWriter,
	}
	log.SetFlags(0)
	log.SetOutput(writer)

	// Return file pointer
	return file
}
