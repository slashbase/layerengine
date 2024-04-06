package store

import (
	"encoding/json"
	"fmt"

	"github.com/boltdb/bolt"
)

// Read retrieves a value from the store based on the given key.
func (s *Store) Read(bucket, key string, result interface{}) error {
	return s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		data := b.Get([]byte(key))
		if data == nil {
			return fmt.Errorf("key not found: %s", key)
		}
		return json.Unmarshal(data, result)
	})
}

// Update updates the value associated with the given key in the store.
func (s *Store) Update(bucket, key string, value interface{}) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		b := tx.Bucket([]byte(bucket))
		return b.Put([]byte(key), data)
	})
}

// Delete removes a key-value pair from the store based on the given key.
func (s *Store) Delete(bucket, key string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		return b.Delete([]byte(key))
	})
}

// ReadAll retrieves all key-value pairs from the store.
func (s *Store) ReadAll(bucket string) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("bucket not found")
		}
		return b.ForEach(func(k, v []byte) error {
			var value interface{}
			if err := json.Unmarshal(v, &value); err != nil {
				return err
			}
			result[string(k)] = value
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
