package main

import (
	"os"
	"io"
	"log"
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


func PrepareLogger() {
	os.MkdirAll("log", os.ModePerm)
    f, err := os.OpenFile("log/latest.log", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
    if err != nil {
        log.Fatalf("Failed to open log/latest.log: %v", err)
    }

    defer f.Close()
    logWriter := io.MultiWriter(os.Stdout, f)
    log.SetFlags(0)
    log.SetOutput(prefixWriter{
        f: func() string { return time.Now().Format(time.RFC3339) + " " },
        w: logWriter,
    })
}
