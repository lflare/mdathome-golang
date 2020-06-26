package diskcache

import (
    "crypto/md5"
    "encoding/hex"
    "io"
    "sort"
    "io/ioutil"
    "time"
    "log"
    "os"
)

// DeleteFile takes an absolute path to a file and deletes it
func (c *Cache) DeleteFile(file string) {
    dir, file := file[0:2] + "/" + file[2:4] + "/" + file[4:6], file

    // Delete file off disk
    err := os.Remove(c.directory + "/" + dir + "/" + file)
    if err != nil {
        log.Println("File does not seem to exist on disk: %v", err)
    }

    // Delete key off database
    err = c.deleteEntry(file)
    if err != nil {
        log.Println("Entry does not seem to exist on database: %v", err)
    }
}

// Get takes a key, hashes it, and returns the corresponding file in the directory
func (c *Cache) Get(key string) (resp []byte, ok bool) {
    dir, key := getCacheKey(key)

    // Read image from directory
    file, err := ioutil.ReadFile(c.directory + "/" + dir + "/" + key)
    if err != nil {
        return nil, false
    }

    // Update timing if older than 1 hour
    keyPair, err := c.getEntry(key)
    if keyPair.Timestamp < time.Now().Add(-1 * time.Hour).Unix() {
        log.Printf("Updating timestamp: %+v", keyPair)
        if err != nil {
            size := len(file)
            timestamp := time.Now().Unix()
            keyPair = KeyPair{key, timestamp, size}
        }
        keyPair.UpdateTimestamp()
        c.setEntry(keyPair)
    }

    // Return file
    return file, true
}

// Set takes a key, hashes it, and saves the `resp` bytearray into the corresponding file
func (c *Cache) Set(key string, resp []byte) {
    dir, key := getCacheKey(key)

    // Save image
    os.MkdirAll(c.directory + "/" + dir, os.ModePerm)
    err := ioutil.WriteFile(c.directory + "/" + dir + "/" + key, resp, 0644)
    if err != nil {
        log.Fatal(err)
    }

    // Update database
    size := len(resp)
    timestamp := time.Now().Unix()
    keyPair := KeyPair{key, timestamp, size}
    c.setEntry(keyPair)
}

func (c *Cache) UpdateCacheLimit(cacheLimit int) {
    c.cacheLimit = cacheLimit
}

func (c *Cache) UpdateCacheScanInterval(cacheScanInterval int) {
    c.cacheScanInterval = cacheScanInterval
}

// StartBackgroundThread starts a background thread that automatically scans the directory and removes older files
// when cache exceeds size limits
func (c *Cache) StartBackgroundThread() {
    for {
        // Retrieve cache information
        size, keys, err := c.loadCacheInfo()
        if err != nil {
            log.Fatal(err)
        }

        // Log
        log.Printf("Current diskcache size: %s, limit: %s", ByteCountIEC(size), ByteCountIEC(c.cacheLimit))

        // If size is bigger than configured byte limit, keep deleting last recently used files
        if size > c.cacheLimit {
            // Get ready to shrink cache
            log.Printf("Shrinking diskcache size: %s, limit: %s", ByteCountIEC(size), ByteCountIEC(c.cacheLimit))
            deletedSize := 0
            deletedItems := 0

            // Loop over keys and delete till we are under threshold
            for _, v := range keys {
                // Delete file
                c.DeleteFile(v.Key)

                // Add to deletedSize
                deletedSize += v.Size
                deletedItems += 1

                // Check if we are under threshold
                if size - deletedSize < c.cacheLimit {
                    break
                }
            }

            // Log success
            log.Printf("Successfully shrunk diskcache by: %s, %d items", ByteCountIEC(deletedSize), deletedItems)
        }

        // Sleep till next execution
        time.Sleep(time.Duration(c.cacheScanInterval) * time.Second)
    }
}

// loadCacheInfo 
func (c *Cache) loadCacheInfo() (int, []KeyPair, error) {
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
    sort.Sort(ByTimestamp(keyPairs))

    // Return running variables
    return totalSize, keyPairs, err
}

func (c *Cache) Close() {
    c.database.Close()
}

func getCacheKey(key string) (string, string) {
    h := md5.New()
    io.WriteString(h, key)
    hash := hex.EncodeToString(h.Sum(nil))

    return hash[0:2] + "/" + hash[2:4] + "/" + hash[4:6], hash
}

// New returns a new Cache that will store files in basePath
func New(directory string, cacheLimit int, cacheScanInterval int) *Cache {
    cache := Cache{
        directory: directory,
        cacheLimit: cacheLimit,
        cacheScanInterval: cacheScanInterval,
    }

    // Setup BoltDB
    cache.setupDB()

    // Start background clean-up thread
    go cache.StartBackgroundThread()

    // Return cache object
    return &cache
}
