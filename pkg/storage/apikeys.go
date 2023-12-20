package storage

import (
	"errors"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/rotationalio/rtnl.link/pkg/storage/models"
)

func (s *Store) Register(obj *models.APIKey) error {
	if obj.Created.IsZero() {
		obj.Created = time.Now()
	}
	obj.Modified = time.Now()

	key, err := obj.Key()
	if err != nil {
		return err
	}

	val, err := obj.MarshalValue()
	if err != nil {
		return err
	}

	err = s.db.Update(func(txn *badger.Txn) error {
		// If the entry already exists, do not overwrite it
		if _, err := txn.Get(key); !errors.Is(err, badger.ErrKeyNotFound) {
			if err == nil {
				return ErrAlreadyExists
			}
			return err
		}

		return txn.Set(key, val)
	})
	return err
}

func (s *Store) Retrieve(clientID string) (*models.APIKey, error) {
	obj := &models.APIKey{ClientID: clientID}
	key, err := obj.Key()
	if err != nil {
		return nil, err
	}

	err = s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			if err := obj.UnmarshalValue(val); err != nil {
				return err
			}
			return nil
		})
	})

	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return obj, nil
}
