package boltdb

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"time"

	"github.com/boltdb/bolt"
	"github.com/lflare/mdathome-golang/pkg/cache"
)

// BoltCache is a struct that represents a cache object
type BoltCache struct {
	directory         string
	cacheLimit        int
	cacheScanInterval int
	cacheRefreshAge   int
	maxCacheScanTime  int
	database          *bolt.DB
}

// DeleteFile takes an absolute path to a file and deletes it
func (c *BoltCache) DeleteFile(file string) error {
	dir, file := file[0:2]+"/"+file[2:4]+"/"+file[4:6], file

	// Delete file off disk
	err := os.Remove(c.directory + "/" + dir + "/" + file)
	if err != nil {
		err = fmt.Errorf("File does not seem to exist on disk: %v", err)
		return err
	}

	// Delete key off database
	err = c.deleteEntry(file)
	if err != nil {
		err = fmt.Errorf("Entry does not seem to exist on database: %v", err)
		return err
	}

	// Return nil if no errors encountered
	return nil
}

// Get takes a key, hashes it, and returns the corresponding file in the directory
func (c *BoltCache) Get(key string) (resp []byte, err error) {
	// Check for empty cache key
	if len(key) == 0 {
		return nil, fmt.Errorf("Empty cache key")
	}

	// Get cache key
	dir, key := getCacheKey(key)

	// Read image from directory
	file, err := ioutil.ReadFile(c.directory + "/" + dir + "/" + key)
	if err != nil {
		err = fmt.Errorf("Failed to read image from key %s: %v", key, err)
		return nil, err
	}

	// Attempt to get keyPair
	keyPair, err := c.getEntry(key)
	if err != nil {
		err = fmt.Errorf("Failed to get entry for cache key %s: %v", key, err)
		return nil, err
	}

	// If keyPair is older than configured cacheRefreshAge, refresh
	if keyPair.Timestamp < time.Now().Add(-1*time.Duration(c.cacheRefreshAge)*time.Second).Unix() {
		log.Printf("Updating timestamp: %+v", keyPair)
		if err != nil {
			size := len(file)
			timestamp := time.Now().Unix()
			keyPair = cache.KeyPair{Key: key, Timestamp: timestamp, Size: size}
		}

		// Update timestamp
		keyPair.UpdateTimestamp()

		// Set entry
		err := c.setEntry(keyPair)
		if err != nil {
			err = fmt.Errorf("Failed to set entry for key %s: %v", key, err)
			return nil, err
		}
	}

	// Return file
	return file, nil
}

// Set takes a key, hashes it, and saves the `resp` bytearray into the corresponding file
func (c *BoltCache) Set(key string, resp []byte) error {
	// Check for empty cache key
	if len(key) == 0 {
		return fmt.Errorf("Empty cache key")
	}

	// Get cache key
	dir, key := getCacheKey(key)

	// Create necessary cache subfolder
	err := os.MkdirAll(c.directory+"/"+dir, os.ModePerm)
	if err != nil {
		err = fmt.Errorf("Failed to create cache folder for key %s: %v", key, err)
		return err
	}

	// Write image
	err = ioutil.WriteFile(c.directory+"/"+dir+"/"+key, resp, 0644)
	if err != nil {
		err = fmt.Errorf("Failed to write image to disk for key %s: %v", key, err)
		return err
	}

	// Update database
	size := len(resp)
	timestamp := time.Now().Unix()
	keyPair := cache.KeyPair{Key: key, Timestamp: timestamp, Size: size}

	// Set database entry
	err = c.setEntry(keyPair)
	if err != nil {
		err = fmt.Errorf("Failed to set entry for key %s: %v", key, err)
		return err
	}

	// Return no error
	return nil
}

// UpdateCacheLimit allows for updating of cache limit=
func (c *BoltCache) UpdateCacheLimit(cacheLimit int) {
	c.cacheLimit = cacheLimit
}

// UpdateCacheScanInterval allows for updating of cache scanning interval
func (c *BoltCache) UpdateCacheScanInterval(cacheScanInterval int) {
	c.cacheScanInterval = cacheScanInterval
}

// UpdateCacheRefreshAge allows for updating of cache refresh age
func (c *BoltCache) UpdateCacheRefreshAge(cacheRefreshAge int) {
	c.cacheRefreshAge = cacheRefreshAge
}

// StartBackgroundThread starts a background thread that automatically scans the directory and removes older files
// when cache exceeds size limits
func (c *BoltCache) StartBackgroundThread() {
	for {
		// Retrieve cache information
		size, keys, err := c.loadCacheInfo()
		if err != nil {
			log.Fatal(err)
		}

		// Log
		log.Printf("Current diskcache size: %s, limit: %s", cache.ByteCountIEC(size), cache.ByteCountIEC(c.cacheLimit))

		// If size is bigger than configured byte limit, keep deleting last recently used files
		if size > c.cacheLimit {
			// Get ready to shrink cache
			log.Printf("Shrinking diskcache size: %s, limit: %s", cache.ByteCountIEC(size), cache.ByteCountIEC(c.cacheLimit))
			deletedSize := 0
			deletedItems := 0

			// Prepare timer
			startTime := time.Now()

			// Loop over keys and delete till we are under threshold
			for _, v := range keys {
				// Delete file
				err := c.DeleteFile(v.Key)
				if err != nil {
					log.Printf("Unable to delete file in key %s: %v", v.Key, err)
					continue
				}

				// Add to deletedSize
				deletedSize += v.Size
				deletedItems++

				// Check if we are under threshold
				if size-deletedSize < c.cacheLimit {
					break
				}

				// Check time elapsed
				if timeElapsed := time.Since(startTime).Seconds(); timeElapsed > float64(c.maxCacheScanTime) {
					break
				}
			}

			// Log success
			log.Printf("Successfully shrunk diskcache by: %s, %d items", cache.ByteCountIEC(deletedSize), deletedItems)
		}

		// Sleep till next execution
		time.Sleep(time.Duration(c.cacheScanInterval) * time.Second)
	}
}

// loadCacheInfo
func (c *BoltCache) loadCacheInfo() (int, []cache.KeyPair, error) {
	// Create running variables
	totalSize := 0

	// Pull keys from BoltDB
	keyPairs, err := c.getAllKeys()
	if err != nil {
		log.Fatal(err)
	}

	// Count total size
	for _, keyPair := range keyPairs {
		totalSize += keyPair.Size
	}

	// Sort cache by access time
	sort.Sort(cache.ByTimestamp(keyPairs))

	// Return running variables
	return totalSize, keyPairs, err
}

// Close closes the database
func (c *BoltCache) Close() {
	c.database.Close()
}

func getCacheKey(key string) (string, string) {
	// Create MD5 hasher
	h := md5.New()

	// Write key to MD5 hasher (should not ever fail)
	_, _ = io.WriteString(h, key)

	// Encode MD5 hash to hexadecimal
	hash := hex.EncodeToString(h.Sum(nil))

	// Return cache key
	return hash[0:2] + "/" + hash[2:4] + "/" + hash[4:6], hash
}

// New returns a new Cache that will store files in basePath
func New(directory string, cacheLimit int, cacheScanInterval int, cacheRefreshAge int, maxCacheScanTime int) *BoltCache {
	cache := BoltCache{
		directory:         directory,
		cacheLimit:        cacheLimit,
		cacheScanInterval: cacheScanInterval,
		cacheRefreshAge:   cacheRefreshAge,
		maxCacheScanTime:  maxCacheScanTime,
	}

	// Setup BoltDB
	err := cache.setupDB()
	if err != nil {
		log.Fatalf("Failed to setup BoltDB: %v", err)
	}

	// Start background clean-up thread
	if cacheScanInterval != 0 {
		go cache.StartBackgroundThread()
	}

	// Return cache object
	return &cache
}
