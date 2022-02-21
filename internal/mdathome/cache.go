package mdathome

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/spf13/viper"
	bolt "go.etcd.io/bbolt"
)

var (
	clientCacheSize    = metrics.NewCounter("client_cache_size_bytes")
	clientCacheLimit   = metrics.NewCounter("client_cache_limit_bytes")
	clientCacheEvicted = metrics.NewCounter("client_cache_evicted_bytes")
)

type KeyPair struct {
	Key       string
	Timestamp int64
	Size      int
}

func (a *KeyPair) UpdateTimestamp() {
	a.Timestamp = time.Now().Unix()
}

func getPathFromHash(hash string) (string, string) {
	dir := hash[0:2] + "/" + hash[2:4] + "/" + hash[4:6]
	parent := viper.GetString("cache.directory") + "/" + dir
	path := parent + "/" + hash
	return parent, path
}

func hashRequestURI(requestURI string) string {
	// Create MD5 hasher
	h := md5.New()

	// Write key to MD5 hasher (should not ever fail)
	_, _ = io.WriteString(h, requestURI)

	// Encode MD5 hash to hexadecimal
	hash := hex.EncodeToString(h.Sum(nil))

	// Return hash
	return hash
}

type Cache struct {
	cacheLimitInBytes int
	database          *bolt.DB
}

func (c *Cache) DeleteFileByKey(hash string) error {
	_, path := getPathFromHash(hash)

	// Delete file off disk
	if err := os.Remove(path); err != nil {
		log.Errorf("File does not seem to exist on disk, ignoring: %v", err)
	}

	// Delete key off database
	if err := c.database.Update(func(tx *bolt.Tx) error {
		if err := tx.Bucket([]byte("KEYS")).Delete([]byte(hash)); err != nil {
			return fmt.Errorf("could not delete entry: %v", err)
		}

		// Return with no errors
		return nil
	}); err != nil {
		return fmt.Errorf("entry does not exist on database: %v", err)
	}

	// Return nil if no errors encountered
	return nil
}

// setEntry adds or modifies an entry in the database from a keyPair
func (c *Cache) setEntry(keyPair KeyPair) error {
	// Marshal keyPair struct into bytes
	keyPairBytes, err := json.Marshal(keyPair)
	if err != nil {
		return fmt.Errorf("unable to marshal keyPair: %v", err)
	}

	// Update database with marshaled keyPair
	err = c.database.Update(func(tx *bolt.Tx) error {
		err = tx.Bucket([]byte("KEYS")).Put([]byte(keyPair.Key), keyPairBytes)
		if err != nil {
			return fmt.Errorf("could not set entry: %v", err)
		}
		return nil
	})

	// Return error if any
	return err
}

func (c *Cache) Get(requestURI string) (reader *os.File, size int64, mtime time.Time, err error) {
	// Check for empty cache key
	if len(requestURI) == 0 {
		return nil, 0, time.Now(), fmt.Errorf("empty cache key")
	}

	// Get cache key
	hash := hashRequestURI(requestURI)
	_, path := getPathFromHash(hash)

	// Read image from directory
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, time.Now(), fmt.Errorf("failed to read image from '%s': %v", path, err)
	}

	// Get file information
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, 0, time.Now(), fmt.Errorf("failed to retrieve file information from '%s': %v", path, err)
	}

	// Attempt to get keyPair
	keyPair, err := c.getEntry(hash)
	if err != nil {
		return nil, 0, time.Now(), fmt.Errorf("failed to get entry for cache key %s: %v", path, err)
	}

	// If keyPair is older than configured cacheRefreshAge, refresh
	if keyPair.Timestamp < time.Now().Add(-1*time.Duration(viper.GetInt("cache.refresh_age_seconds"))*time.Second).Unix() {
		log.Debugf("Updating timestamp: %+v", keyPair)
		if err != nil {
			size := fileInfo.Size()
			timestamp := time.Now().Unix()
			keyPair = KeyPair{hash, timestamp, int(size)}
		}

		// Update timestamp
		keyPair.UpdateTimestamp()

		// Set entry
		err := c.setEntry(keyPair)
		if err != nil {
			return nil, 0, time.Now(), fmt.Errorf("failed to set entry for key %s: %v", requestURI, err)
		}
	}

	// Return file
	return file, fileInfo.Size(), fileInfo.ModTime(), nil
}

// Set takes a key, hashes it, and saves the `resp` bytearray into the corresponding file
func (c *Cache) Set(requestURI string, mtime time.Time, resp []byte) error {
	// Check for empty cache key
	if len(requestURI) == 0 {
		return fmt.Errorf("empty cache key")
	}

	// Get cache key
	hash := hashRequestURI(requestURI)
	parent, path := getPathFromHash(hash)

	// Create necessary cache subfolder
	if err := os.MkdirAll(parent, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create parent folder for '%s' at '%s': %v", requestURI, parent, err)
	}

	// Write image
	if err := os.WriteFile(path, resp, 0644); err != nil {
		return fmt.Errorf("failed to write image to disk for '%s' at '%s': %v", requestURI, path, err)
	}

	// Update modification time
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		return fmt.Errorf("failed to set modification time of image '%s': %v", path, err)
	}

	// Update database
	size := len(resp)
	timestamp := time.Now().Unix()
	keyPair := KeyPair{hash, timestamp, size}

	// Set database entry
	if err := c.setEntry(keyPair); err != nil {
		return fmt.Errorf("failed to write image to database of key '%s' at '%s' : %v", hash, path, err)
	}

	// Update Prometheus metrics
	clientCacheSize.Add(size)

	// Return no error
	return nil
}

// UpdateCacheLimit allows for updating of cache limit=
func (c *Cache) UpdateCacheLimit(cacheLimit int) {
	c.cacheLimitInBytes = cacheLimit
	clientCacheLimit.Set(uint64(cacheLimit))
}

func (c *Cache) loadCacheInfo() (int, []KeyPair, error) {
	// Create running variables
	totalSize := 0

	// Pull keys from BoltDB
	keyPairs, err := c.Scan()
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

func (c *Cache) StartCompanionThread(keys []KeyPair) {
	for {
		// Sleep for 15 seconds before continuing
		time.Sleep(15 * time.Second)

		// Continue if clientCacheSize == 0
		if clientCacheSize.Get() == 0 {
			continue
		}

		// Calculate usage
		usage := 100 * (float32(clientCacheSize.Get()) / float32(c.cacheLimitInBytes))
		log.Debugf("Current diskcache size: %s, limit: %s, usage: %0.3f%%", ByteCountIEC(int(clientCacheSize.Get())), ByteCountIEC(c.cacheLimitInBytes), usage)

		// Continue if clientCacheSize under limit
		if int(clientCacheSize.Get()) < c.cacheLimitInBytes {
			continue
		}

		// Get ready to shrink cache
		deletedSize := 0
		deletedItems := 0
		startTime := time.Now()

		// Loop over keys and delete till we are under threshold
		for {
			// Pop key
			v := keys[0]
			keys = keys[1:]

			// Delete file
			err := c.DeleteFileByKey(v.Key)
			if err != nil {
				log.Warnf("Unable to delete file in key '%s': %v", v.Key, err)
			}

			// Add to deletedSize
			clientCacheSize.Add(-1 * v.Size)
			clientCacheEvicted.Add(v.Size)
			deletedSize += v.Size
			deletedItems++

			// Check if we are under threshold
			if int(clientCacheSize.Get()) < c.cacheLimitInBytes {
				break
			}

			// Check time elapsed
			if timeElapsed := time.Since(startTime).Seconds(); timeElapsed > float64(viper.GetInt("cache.max_scan_time_seconds")) {
				break
			}
		}
	}
}

func (c *Cache) StartBackgroundThread() {
	var keys []KeyPair

	// Rescan every scan interval for fresh keys
	companionRunning := false
	for {
		// Retrieve cache information
		var err error
		var size int
		if size, keys, err = c.loadCacheInfo(); err != nil {
			log.Fatal(err)
		}

		// Update Prometheus metrics
		if size > 0 {
			clientCacheSize.Set(uint64(size))
		}

		// If partner thread not running, run now
		if !companionRunning {
			go c.StartCompanionThread(keys)
			companionRunning = true
		}

		// Sleep till next execution
		time.Sleep(viper.GetDuration("cache.max_scan_interval_seconds") * time.Second)
	}

}

// Close closes the database
func (c *Cache) Close() {
	c.database.Close()
}

// getEntry retrieves an entry from the database from a key
func (c *Cache) getEntry(hash string) (KeyPair, error) {
	// Prepare empty keyPair variable
	var keyPair KeyPair

	// Retrieve entry from database
	err := c.database.View(func(tx *bolt.Tx) error {
		// Retrieve key value
		keyPairBytes := tx.Bucket([]byte("KEYS")).Get([]byte(hash))
		if keyPairBytes == nil {
			return fmt.Errorf("key does not exist")
		}

		// Unmarshal keyPairBytes into previously declared keyPair
		err := json.Unmarshal(keyPairBytes, &keyPair)
		if err != nil {
			return err
		}

		return nil
	})

	// Return keyPair and error if any
	return keyPair, err
}

func (c *Cache) Scan() ([]KeyPair, error) {
	// Prepare empty keyPairs reference
	var keyPairs []KeyPair

	// Retrieve all entries from database, unmarshaling and appending to []keyPair slice
	err := c.database.View(func(tx *bolt.Tx) error {
		// Get bucket
		b := tx.Bucket([]byte("KEYS"))

		// Create slice of keypairs of size of bucket
		keyPairs = make([]KeyPair, b.Stats().KeyN)
		index := 0

		// Prepare timer
		startTime := time.Now()

		// Cursor
		cur := b.Cursor()
		for key, keyPairBytes := cur.First(); key != nil; key, keyPairBytes = cur.Next() {
			// Prepare empty keyPair struct
			var keyPair KeyPair

			// Unmarshal bytes
			err := json.Unmarshal(keyPairBytes, &keyPair)
			if err != nil {
				return err
			}

			// Append to keyPairs
			keyPairs[index] = keyPair
			index++

			// Check time
			if timeElapsed := time.Since(startTime).Seconds(); timeElapsed > float64(viper.GetInt("cache.max_scan_time_seconds")) {
				break
			}
		}

		return nil
	})

	// Return keyPairs and errors if any
	return keyPairs, err
}

func (c *Cache) Shrink() error {
	// Hook on to SIGTERM
	sigtermChannel := make(chan os.Signal, 1)
	signal.Notify(sigtermChannel, os.Interrupt, syscall.SIGTERM)

	// Start coroutine to wait for SIGTERM
	handler := make(chan struct{})
	go func() {
		for {
			select {
			case <-sigtermChannel:
				// Prepare to shutdown server
				log.Println("Aborted database shrinking!")

				// Delete half-shrunk database
				os.Remove(viper.GetString("cache.directory") + "/cache.db.tmp")

				// Exit properly
				close(handler)
				os.Exit(0)
			case <-handler:
				close(sigtermChannel)
				return
			}
		}
	}()

	// Prepare new database location
	newDB, err := bolt.Open(viper.GetString("cache.directory")+"/cache.db.tmp", 0600, nil)
	if err != nil {
		log.Errorf("failed to open new database location: %v", err)
		os.Exit(1)
	}

	// Attempt to compact database
	if err = bolt.Compact(newDB, c.database, 0); err != nil {
		log.Fatalf("failed to compact database: %v", err)
	}

	// Close new database
	if err = newDB.Close(); err != nil {
		log.Errorf("failed to close new database: %v", err)
	}

	// Close old database
	if err = c.database.Close(); err != nil {
		log.Errorf("failed to close old database: %v", err)
		os.Exit(1)
	}

	// Rename database files
	if err := os.Rename(viper.GetString("cache.directory")+"/cache.db", viper.GetString("cache.directory")+"/cache.db.bak"); err != nil {
		log.Fatalf("failed to backup database: %v", err)
	}
	if err := os.Rename(viper.GetString("cache.directory")+"/cache.db.tmp", viper.GetString("cache.directory")+"/cache.db"); err != nil {
		log.Fatalf("failed to restore new database: %v", err)
	}
	log.Infof("Database backed up and renamed!")

	// Stop goroutine
	handler <- struct{}{}
	return nil
}

func (c *Cache) Setup() (err error) {
	// Create cache directory if not exists
	if err = os.MkdirAll(viper.GetString("cache.directory"), os.ModePerm); err != nil {
		return fmt.Errorf("could not create cache directory '%s': %v", viper.GetString("cache.directory"), err)
	}

	// Open BoltDB database
	options := c.getOptions()
	if c.database, err = bolt.Open(viper.GetString("cache.directory")+"/cache.db", 0600, options); err != nil {
		return fmt.Errorf("could not open database: %v", err)
	}

	// Create bucket if not exists
	if err := c.database.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte("KEYS")); err != nil {
			return fmt.Errorf("could not create bucket: %v", err)
		}

		// Return with no errors
		return nil
	}); err != nil {
		return fmt.Errorf("failed to craete bucket: %v", err)
	}

	// Database ready!
	log.Infof("Database ready!")
	return nil
}

func OpenCache(directory string, cacheLimit int) *Cache {
	cache := Cache{
		cacheLimitInBytes: cacheLimit,
	}

	// Setup BoltDB
	err := cache.Setup()
	if err != nil {
		log.Fatalf("failed to setup BoltDB: %v", err)
	}

	// Prep metrics counter
	clientCacheLimit.Set(uint64(cacheLimit))

	// Start background clean-up thread
	if viper.GetDuration("cache.max_scan_interval_seconds") > 0 {
		go cache.StartBackgroundThread()
	}

	// Return cache object
	return &cache
}
