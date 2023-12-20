package migrations

import (
	"errors"
	"time"

	"github.com/dgraph-io/badger/v4"
)

// Migrate the database to the current version or ensure that the database is current.
func Migrate(txn *badger.Txn) (err error) {
	// If there are no migrations to apply, do nothing
	if len(migrations) == 0 {
		return nil
	}

	// Load the current migration from the database
	var current *Migration
	if current, err = currentMigration(txn); err != nil {
		return err
	}

	// Keep applying migrations while the current migration has next
	for migrations.HasNext(current) {
		next := migrations.Next(current)
		if err := next.Migrate(txn); err != nil {
			return err
		}
		next.Applied = time.Now()
		next.Previous = current
		current = next
	}

	// Save the current migration to the database
	if err = saveMigration(txn, current); err != nil {
		return err
	}
	return nil
}

func currentMigration(txn *badger.Txn) (_ *Migration, err error) {
	var item *badger.Item
	if item, err = txn.Get(migrationKey); err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			// This is either an empty new database or has never been migrated.
			// Return a nil migration and no error
			return nil, nil
		}
		return nil, err
	}

	migration := &Migration{}
	if err = item.Value(migration.UnmarshalValue); err != nil {
		return nil, err
	}

	return migration, nil
}

func saveMigration(txn *badger.Txn, m *Migration) (err error) {
	var data []byte
	if data, err = m.MarshalValue(); err != nil {
		return err
	}
	return txn.Set(migrationKey, data)
}
