// +build !linux

package diskcache

import (
	bolt "go.etcd.io/bbolt"
)

func (c *Cache) getOptions() *bolt.Options {
	options := &bolt.Options{}
	return options
}
