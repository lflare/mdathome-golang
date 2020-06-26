package main

import (
	"io"
)

// Returns a list of dictionary keys to use for the cache file key
func SimpleTransform (key string) []string {
    return []string{key[0:2], key[2:4], key[4:6]}
}

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
