package cache

import (
	"time"
)

// ChapterCache caches a chapter locally
type ChapterCache interface {
	Get(key string) ([]byte, error)
	Put(key string, resp []byte) error
	GetCacheLimit() int
	SetCacheLimit(limit int)
}

// SelfCleaningCache periodically cleans itself
type SelfCleaningCache interface {
	GetCleanInterval() time.Duration
	SetCleanInterval(interval time.Duration)

	Clean() error
	BackgroundThread()
}

// ExpiringCache automatically refreshes its keys from upstream when its TTL expires
type ExpiringCache interface {
	GetTTL() time.Duration
	SetTTL(time.Duration)
}

// FSCache is located at a filesystem path
type FSCache interface {
	GetDirectory() string
}
