package storage

import (
	"errors"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/rotationalio/rtnl.link/pkg/storage/models"
)

func (s *Store) Save(obj *models.ShortURL) error {
	if obj.Created.IsZero() {
		obj.Created = time.Now()
	}

	obj.Modified = time.Now()

	key := obj.Key()
	val, err := obj.MarshalValue()
	if err != nil {
		return err
	}

	entry := badger.NewEntry(key, val)
	if !obj.Expires.IsZero() {
		entry = entry.WithTTL(time.Until(obj.Expires))
	}

	err = s.db.Update(func(txn *badger.Txn) error {
		// If the entry already exists, do not overwrite it
		if _, err := txn.Get(key); !errors.Is(err, badger.ErrKeyNotFound) {
			if err == nil {
				return ErrAlreadyExists
			}
			return err
		}

		return txn.SetEntry(entry)
	})
	return err
}

// TODO: support pagination when listing links.
func (s *Store) List() ([]*models.ShortURL, error) {
	urls := make([]*models.ShortURL, 0)

	err := s.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		prefix := models.LinksBucket[:]
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			obj := &models.ShortURL{}

			err := item.Value(func(v []byte) error {
				return obj.UnmarshalValue(v)
			})

			if err != nil {
				return err
			}

			urls = append(urls, obj)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return urls, nil
}

func (s *Store) Load(key uint64) (string, error) {
	obj := &models.ShortURL{ID: key}
	keyb := obj.Key()

	err := s.db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get(keyb)
		if err != nil {
			return err
		}

		err = item.Value(func(val []byte) error {
			if err := obj.UnmarshalValue(val); err != nil {
				return err
			}
			return nil
		})

		if err != nil {
			return err
		}

		obj.Visits++
		obj.Modified = time.Now()

		data, err := obj.MarshalValue()
		if err != nil {
			return err
		}

		return txn.Set(keyb, data)
	})

	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return "", ErrNotFound
		}
		return "", err
	}
	return obj.URL, nil
}

func (s *Store) LoadInfo(key uint64) (*models.ShortURL, error) {
	obj := &models.ShortURL{ID: key}
	keyb := obj.Key()

	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(keyb)
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

func (s *Store) Delete(key uint64) error {
	obj := &models.ShortURL{ID: key}
	keyb := obj.Key()

	err := s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(keyb)
	})

	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}
