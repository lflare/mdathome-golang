package diskcache

import (
    "encoding/json"
    "fmt"
    "os"

    "github.com/boltdb/bolt"
)

// setEntry sets an entry into the database
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

// deleteEntry deletes an entry from database
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
        json.Unmarshal(keyPairBytes, &keyPair)
        return nil
    })

    // Return keyPair and error if any
    return keyPair, err
}

// getAllKeys returns a full slice of keyPairs from database
func (c *Cache) getAllKeys() ([]KeyPair, error) {
    // Prepare empty []keyPair slice
    var keyPairs []KeyPair

    // Retrieve all entries from database, unmarshaling and appending to []keyPair slice
    err := c.database.View(func(tx *bolt.Tx) error {
        b := tx.Bucket([]byte("KEYS"))
        b.ForEach(func(_, keyPairBytes []byte) error {
            // Prepare empty keyPair struct
            var keyPair KeyPair

            // Unmarshal bytes
            json.Unmarshal(keyPairBytes, &keyPair)

            // Append to keyPairs
            keyPairs = append(keyPairs, keyPair)
            return nil
        })
        return nil
    })

    // Return keyPairs and errors if any
    return keyPairs, err
}

// setupDB opens 
func (c *Cache) setupDB() error {
    // Create cache directory if not exists
    os.MkdirAll(c.directory, os.ModePerm)

    // Open BoltDB database
    database, err := bolt.Open(c.directory + "/cache.db", 0600, nil)
    if err != nil {
        return fmt.Errorf("Could not open database: %v", err)
    }

    // Set database to cache struct
    c.database = database

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
    fmt.Println("Database ready!")
    return nil
}
