package diskcache

import (
	"syscall"

	bolt "go.etcd.io/bbolt"
)

func (c *Cache) getOptions() *bolt.Options {
	options := &bolt.Options{
		MmapFlags: syscall.MAP_POPULATE,
	}
	return options
}
