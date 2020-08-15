// +build !linux

package diskcache

import (
	"github.com/boltdb/bolt"
)

func (c *Cache) getOptions() *bolt.Options {
	options := &bolt.Options{}
	return options
}
