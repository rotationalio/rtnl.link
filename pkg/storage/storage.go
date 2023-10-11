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
	Load(uint64) (string, error)
	LoadInfo(uint64) (*ShortURL, error)
	Delete(uint64) error
	Register(*APIKey) error
	Retrieve(string) (*APIKey, error)
}

func Open(conf config.StorageConfig) (_ Storage, err error) {
	opts := badger.DefaultOptions(conf.DataPath)
	opts.ReadOnly = conf.ReadOnly
	opts.Logger = nil

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
		return nil, err
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

func (s *Store) Register(obj *APIKey) error {
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

func (s *Store) Retrieve(clientID string) (*APIKey, error) {
	obj := &APIKey{ClientID: clientID}
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
