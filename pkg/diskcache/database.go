package diskcache

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/boltdb/bolt"
)

// setEntry adds or modifies an entry in the database from a keyPair
func (c *Cache) setEntry(keyPair KeyPair) error {
	// Marshal keyPair struct into bytes
	keyPairBytes, err := json.Marshal(keyPair)
	if err != nil {
		return fmt.Errorf("Unable to marshal keyPair: %v", err)
	}

	// Update database with marshaled keyPair
	err = c.database.Update(func(tx *bolt.Tx) error {
		err = tx.Bucket([]byte("KEYS")).Put([]byte(keyPair.Key), keyPairBytes)
		if err != nil {
			return fmt.Errorf("Could not set entry: %v", err)
		}
		return nil
	})

	// Return error if any
	return err
}

// deleteEntry deletes an entry from database from a key
func (c *Cache) deleteEntry(key string) error {
	// Update database and delete entry
	err := c.database.Update(func(tx *bolt.Tx) error {
		err := tx.Bucket([]byte("KEYS")).Delete([]byte(key))
		if err != nil {
			return fmt.Errorf("Could not delete entry: %v", err)
		}
		return nil
	})

	// Return error if any
	return err
}

// getEntry retrieves an entry from the database from a key
func (c *Cache) getEntry(key string) (KeyPair, error) {
	// Prepare empty keyPair variable
	var keyPair KeyPair

	// Retrieve entry from database
	err := c.database.View(func(tx *bolt.Tx) error {
		// Retrieve key value
		keyPairBytes := tx.Bucket([]byte("KEYS")).Get([]byte(key))
		if keyPairBytes == nil {
			return fmt.Errorf("Could not retrieve entry")
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

// getAllKeys returns a full slice of keyPairs from the database
func (c *Cache) getAllKeys() ([]KeyPair, error) {
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
			if timeElapsed := time.Since(startTime).Seconds(); timeElapsed > float64(c.maxCacheScanTime) {
				break
			}
		}

		return nil
	})

	// Return keyPairs and errors if any
	return keyPairs, err
}

// ShrinkDatabase manually re-creates the cache.db file and effectively shrinks it
func (c *Cache) ShrinkDatabase() error {
	// Hook on to SIGTERM
	sigtermChannel := make(chan os.Signal)
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
				os.Remove(c.directory + "/cache.db.tmp")

				// Exit properly
				close(handler)
				os.Exit(0)
			case <-handler:
				close(sigtermChannel)
				return
			}
		}
	}()

	// Read old database
	data := readDatabase(c.database)
	defer c.database.Close()
	saveDatabase(data, c.directory+"/cache.db.tmp")
	c.database.Close()

	// Rename database files
	os.Rename(c.directory+"/cache.db", c.directory+"/cache.db.bak")
	os.Rename(c.directory+"/cache.db.tmp", c.directory+"/cache.db")
	log.Println("Database backed up!")

	// Stop goroutine
	handler <- struct{}{}
	return nil
}

func (c *Cache) openDB() error {
	// Open BoltDB database
	options := c.getOptions()
	database, err := bolt.Open(c.directory+"/cache.db", 0600, options)
	if err != nil {
		return fmt.Errorf("Could not open database: %v", err)
	}

	// Set database to cache struct
	c.database = database
	return nil
}

// setupDB initialises the BoltDB database
func (c *Cache) setupDB() error {
	// Create cache directory if not exists
	err := os.MkdirAll(c.directory, os.ModePerm)
	if err != nil {
		return fmt.Errorf("Could not create cache directory %s: %v", c.directory, err)
	}

	// Open database
	err = c.openDB()
	if err != nil {
		return fmt.Errorf("Failed to open database: %v", err)
	}

	// Create bucket if not exists
	err = c.database.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("KEYS"))
		if err != nil {
			return fmt.Errorf("Could not create bucket: %v", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("Failed to setup bucket, %v", err)
	}

	// Database ready!
	log.Println("Database ready!")
	return nil
}
