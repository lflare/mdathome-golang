package diskcache

import (
	"syscall"

	"github.com/boltdb/bolt"
)

func (c *Cache) getOptions() *bolt.Options {
	options := &bolt.Options{
		MmapFlags: syscall.MAP_POPULATE,
	}
	return options
}
