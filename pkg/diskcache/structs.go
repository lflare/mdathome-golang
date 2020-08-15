package diskcache

import (
	"time"

	"github.com/boltdb/bolt"
)

// Cache is a struct that represents a cache object
type Cache struct {
	directory         string
	cacheLimit        int
	cacheScanInterval int
	cacheRefreshAge   int
	maxCacheScanTime  int
	database          *bolt.DB
}

// KeyPair is a struct that represents a cache key in database
type KeyPair struct {
	Key       string
	Timestamp int64
	Size      int
}

// UpdateTimestamp allows for updating of a KeyPair timestamp field
func (a *KeyPair) UpdateTimestamp() { a.Timestamp = time.Now().Unix() }

// ByTimestamp is a sortable slice of KeyPair based off timestamp
type ByTimestamp []KeyPair

func (a ByTimestamp) Len() int           { return len(a) }
func (a ByTimestamp) Less(i, j int) bool { return a[i].Timestamp < a[j].Timestamp }
func (a ByTimestamp) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
