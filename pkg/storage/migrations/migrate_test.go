package migrations_test

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/dgraph-io/badger/v4"
	"github.com/rotationalio/rtnl.link/pkg/storage/migrations"
	"github.com/rotationalio/rtnl.link/pkg/storage/models"
	"github.com/stretchr/testify/require"
)

// NOTE: must update this value when new migrations are added!
const latestMigration = uint16(1)

func TestMigrate(t *testing.T) {
	t.Run("MIG0000", func(t *testing.T) {
		db := makeFixtureDB(t, "testdata/mig0.tgz")

		// Check the fixture
		records, err := countsByKeyLength(db)
		require.NoError(t, err, "could not count contents of database")
		require.Len(t, records, 2, "unexpected number of objects, have the fixtures changed?")
		require.Equal(t, 6, records["🔗"], "unexpected number of links, have the fixtures changed?")
		require.Equal(t, 1, records["🔑"], "unexpected number of apikeys, have the fixtures changed?")

		// Apply the migrtions
		err = db.Update(migrations.Migrate)
		require.NoError(t, err)

		// Check that we're at the latest registered migration
		err = checkLatest(db)
		require.NoError(t, err, "not at latest registered migration")

		// Check that everything in the database is now prefixed
		newRecords, err := counts(db)
		delete(newRecords, "meta")

		require.NoError(t, err, "could not count contents of database")
		require.Equal(t, records, newRecords, "counts do not match original counts")
	})
}

func counts(db *badger.DB) (map[string]int, error) {
	counter := make(map[string]int)
	err := db.View(func(txn *badger.Txn) error {
		iter := txn.NewIterator(badger.DefaultIteratorOptions)
		defer iter.Close()

		for iter.Rewind(); iter.Valid(); iter.Next() {
			item := iter.Item()
			key := item.Key()
			bucket := models.Bucket(key[0:4])
			counter[bucket.String()]++
		}
		return nil
	})

	return counter, err
}

// Counts returns the number of items in the database by key length
func countsByKeyLength(db *badger.DB) (map[string]int, error) {
	counter := make(map[string]int)
	err := db.View(func(txn *badger.Txn) error {
		iter := txn.NewIterator(badger.DefaultIteratorOptions)
		defer iter.Close()

		for iter.Rewind(); iter.Valid(); iter.Next() {
			item := iter.Item()
			key := item.Key()

			switch len(key) {
			case 16:
				counter["🔑"]++
			case 8:
				counter["🔗"]++
			default:
				counter["unknown"]++
			}
		}
		return nil
	})

	return counter, err
}

func checkLatest(db *badger.DB) error {
	var current *migrations.Migration
	err := db.View(func(txn *badger.Txn) (err error) {
		if current, err = migrations.Current(txn); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}

	var version uint16
	if current != nil {
		version = current.Version
	}

	if version != latestMigration {
		return fmt.Errorf("current migration version %d does not match latest version %d; does latestMigration in tests need to be updated?", version, latestMigration)
	}
	return nil
}

// Create a badger database using the fixtures at a temporary directory.
func makeFixtureDB(t *testing.T, fixturePath string) (db *badger.DB) {
	dbpath := unzipFixtures(t, fixturePath)

	opts := badger.DefaultOptions(dbpath)
	opts.ReadOnly = false
	opts.Logger = nil

	db, err := badger.Open(opts)
	require.NoError(t, err, "could not open badger database")
	return db
}

// Unzip the fixture path to a temporary directory, ready to load a database.
func unzipFixtures(t *testing.T, fixturePath string) (dbpath string) {
	f, err := os.Open(fixturePath)
	require.NoError(t, err, "could not open fixture path")
	defer f.Close()

	gz, err := gzip.NewReader(f)
	require.NoError(t, err, "could not open gzip stream")
	defer gz.Close()

	tarball := tar.NewReader(gz)
	dbpath = t.TempDir()

	for {
		header, err := tarball.Next()
		if err == io.EOF {
			break
		}

		require.NoError(t, err, "could not read next item in tarball")

		switch header.Typeflag {
		case tar.TypeDir:
			err = os.Mkdir(filepath.Join(dbpath, header.Name), 0755)
			require.NoError(t, err, "could not make directory %s", header.Name)
		case tar.TypeReg:
			out, err := os.Create(filepath.Join(dbpath, header.Name))
			require.NoError(t, err, "could not create file %s", header.Name)

			_, err = io.Copy(out, tarball)
			require.NoError(t, err, "could not write contents to %s", header.Name)
			out.Close()
		default:
			require.NoError(t, fmt.Errorf("unknown type %d in %s", header.Typeflag, header.Name))
		}
	}

	return dbpath
}
