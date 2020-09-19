// +build !linux

package boltdb

import (
	"github.com/boltdb/bolt"
)

func (c *BoltCache) getOptions() *bolt.Options {
	options := &bolt.Options{}
	return options
}
