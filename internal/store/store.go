package store

import (
	"log"

	"github.com/boltdb/bolt"
)

type Store struct {
	db *bolt.DB
}

func NewStore(bucketNames []string) *Store {
	store := Store{}
	store.init(bucketNames)
	return &store
}

func (s *Store) init(bucketNames []string) {
	var err error
	s.db, err = bolt.Open("app.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	for _, bucketName := range bucketNames {
		s.db.Update(func(tx *bolt.Tx) error {
			_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
			if err != nil {
				return err
			}
			return err
		})
	}

}

func (s *Store) Close() {
	s.db.Close()
}
