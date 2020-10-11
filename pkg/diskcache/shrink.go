package diskcache

import (
	"log"
	"os"

	bolt "go.etcd.io/bbolt"
)

////
// Code was shamelessly stolen from
// https://github.com/cydside/sybod
////

// Entry Model for Keys and Values bucket's data
type Entry struct {
	Key   []byte
	Value []byte
}

// Container Model for BoltDB's buckets structure
type Container struct {
	Name       []byte
	SubBuckets []Container
	Entries    []Entry
}

func readDatabase(db *bolt.DB) *Container {
	data := new(Container)

	db.View(func(tx *bolt.Tx) error {
		tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			cb := new(Container)
			cb.Name = name
			readBucket(cb, b)
			data.SubBuckets = append(data.SubBuckets, *cb)

			return nil
		})
		return nil
	})

	return data
}

func readBucket(bkt *Container, b *bolt.Bucket) {
	b.ForEach(func(k, v []byte) error {
		if subB := b.Bucket(k); subB != nil {
			sb := new(Container)
			sb.Name = k
			readBucket(sb, subB)

			bkt.SubBuckets = append(bkt.SubBuckets, *sb)

			return nil
		}

		et := new(Entry)
		et.Key = k
		et.Value = v
		bkt.Entries = append(bkt.Entries, *et)

		return nil
	})

	return
}

func saveDatabase(bkt *Container, fileName string) {
	db, err := bolt.Open(fileName, 0600, nil)
	if err != nil {
		log.Println("Can't open destination db", err)
		os.Exit(1)
	}

	var bktPath []string
	readStruct(db, bkt, bktPath)

	err = db.Close()
	if err != nil {
		log.Println("Can't close destination db", err)
		os.Exit(1)
	}
}

func readStruct(db *bolt.DB, bkt *Container, bktPath []string) {
	var err error

	for _, cb := range bkt.SubBuckets {
		bktPath = append(bktPath, string(cb.Name))
		log.Printf("Entries: %d\n", len(cb.Entries))

		if cb.SubBuckets != nil {
			readStruct(db, &cb, bktPath)
		}

		err = makeBucket(db, &bktPath)
		if err != nil {
			log.Printf("Can't create bucket %s: %s", string(cb.Name), err)
			os.Exit(1)
		}

		err = insertEntry(db, cb.Entries, &bktPath)
		if err != nil {
			log.Printf("Can't insert into bucket %s: %s", string(bkt.Name), err)
			os.Exit(1)
		}

		bktPath = bktPath[:len(bktPath)-1]
	}
}

func makeBucket(db *bolt.DB, bktPath *[]string) error {
	var err error
	var b *bolt.Bucket

	err = db.Update(func(tx *bolt.Tx) (err error) {
		for _, bktName := range *bktPath {
			if b != nil {
				if b, err = b.CreateBucketIfNotExists([]byte(bktName)); err != nil {
					return err
				}

			} else {
				if b, err = tx.CreateBucketIfNotExists([]byte(bktName)); err != nil {
					return err
				}
			}
		}

		return err
	})

	return err
}

func getBucket(tx *bolt.Tx, bktPath *[]string) *bolt.Bucket {
	var b *bolt.Bucket

	for _, bktName := range *bktPath {
		if b != nil {
			if b = b.Bucket([]byte(bktName)); b == nil {
				return nil
			}
		} else {
			if b = tx.Bucket([]byte(bktName)); b == nil {
				return nil
			}
		}
	}

	return b
}

func insertEntry(db *bolt.DB, entry []Entry, bktPath *[]string) error {
	var err error

	for _, et := range entry {
		err = db.Update(func(tx *bolt.Tx) error {
			var b = getBucket(tx, bktPath)
			err = b.Put(et.Key, et.Value)
			return err
		})
		if err != nil {
			return err
		}
	}

	return err
}
