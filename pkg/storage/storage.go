package storage

import (
	"io"

	"github.com/dgraph-io/badger/v4"
	"github.com/rotationalio/rtnl.link/pkg/config"
	"github.com/rotationalio/rtnl.link/pkg/storage/migrations"
	"github.com/rotationalio/rtnl.link/pkg/storage/models"
)

type Storage interface {
	io.Closer
	LinkStorage
	APIKeyStorage
}

type LinkStorage interface {
	Save(*models.ShortURL) error
	Load(uint64) (string, error)
	LoadInfo(uint64) (*models.ShortURL, error)
	Delete(uint64) error
}

type APIKeyStorage interface {
	Register(*models.APIKey) error
	Retrieve(string) (*models.APIKey, error)
}

func Open(conf config.StorageConfig) (_ Storage, err error) {
	opts := badger.DefaultOptions(conf.DataPath)
	opts.ReadOnly = conf.ReadOnly
	opts.Logger = nil

	store := &Store{}
	if store.db, err = badger.Open(opts); err != nil {
		return nil, err
	}

	// Run the migrations to ensure the database is up to date.
	if err = store.db.Update(migrations.Migrate); err != nil {
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

func (s *Store) DB() *badger.DB {
	return s.db
}
