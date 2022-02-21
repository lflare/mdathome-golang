//go:build !linux
// +build !linux

package mdathome

import (
	bolt "go.etcd.io/bbolt"
)

func (c *Cache) getOptions() *bolt.Options {
	// Return no custom options because Windows does not support
	options := &bolt.Options{}
	return options
}

func configureConfigAutoReload() {
	// Do absolutely nothing because Windows does not support
}
