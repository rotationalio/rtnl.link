package storage

import (
	"errors"
	"io"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/rotationalio/rtnl.link/pkg/config"
)

type Storage interface {
	io.Closer
	Save(*ShortURL) error
	Load(key uint64) (string, error)
	LoadInfo(key uint64) (*ShortURL, error)
	Delete(key uint64) error
}

func Open(conf config.StorageConfig) (_ Storage, err error) {
	opts := badger.DefaultOptions(conf.DataPath)
	opts.ReadOnly = conf.ReadOnly

	store := &Store{}
	if store.db, err = badger.Open(opts); err != nil {
		return nil, err
	}
	return store, nil
}

type Store struct {
	db *badger.DB
}

var _ Storage = &Store{}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Save(obj *ShortURL) error {
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
		// TODO: check if object already exists before overwrite
		return txn.SetEntry(entry)
	})
	return err
}

func (s *Store) Load(key uint64) (string, error) {
	obj := &ShortURL{ID: key}
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

func (s *Store) LoadInfo(key uint64) (*ShortURL, error) {
	obj := &ShortURL{ID: key}
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
	}
	return obj, nil
}

func (s *Store) Delete(key uint64) error {
	obj := &ShortURL{ID: key}
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
