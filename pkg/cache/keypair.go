package cache

import (
	"encoding/json"
	"time"
)

// KeyPair is a struct that represents a cache key in database
type KeyPair struct {
	Key       string
	Timestamp int64
	Size      int
}

// UpdateTimestamp allows for updating of a KeyPair timestamp field
func (a *KeyPair) UpdateTimestamp() { a.Timestamp = time.Now().Unix() }

// ToJSON marshals a KeyPair into a JSON byte slice
func (a *KeyPair) ToJSON() ([]byte, error) {
	return json.Marshal(a)
}

// FromJSON unmarshals a KeyPair from a JSON byte slice
func (a *KeyPair) FromJSON(data []byte) error {
	err := json.Unmarshal(data, &a)
	return err
}

// ByTimestamp is a sortable slice of KeyPair based off timestamp
type ByTimestamp []KeyPair

func (a ByTimestamp) Len() int           { return len(a) }
func (a ByTimestamp) Less(i, j int) bool { return a[i].Timestamp < a[j].Timestamp }
func (a ByTimestamp) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
